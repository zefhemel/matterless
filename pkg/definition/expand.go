package definition

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/imdario/mergo"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/util"
	"github.com/zefhemel/yamlschema"
)

func (defs *Definitions) ExpandMacros() error {
	for len(defs.MacroInstances) > 0 {
		for name, def := range defs.MacroInstances {
			macro, ok := defs.Macros[def.Macro]
			if !ok {
				return fmt.Errorf("No such macro: %s", def.Macro)
			}

			if macro.Config.ArgumentsSchema != nil {
				if err := yamlschema.ValidateObjects(macro.Config.ArgumentsSchema, def.Arguments); err != nil {
					return errors.Wrapf(err, "macro expansion: %s", name)
				}
			}

			t := template.New("template")
			t.Funcs(CodeGenFuncs)
			t2, err := t.Parse(fmt.Sprintf(`
{{- $name := .Name -}}
{{- $arg := .Arguments -}}
%s`, macro.TemplateCode))
			if err != nil {
				return errors.Wrap(err, "parsing template")
			}
			var out bytes.Buffer
			if err := t2.Execute(&out, struct {
				Name      string
				Arguments interface{}
			}{
				Name:      name,
				Arguments: def.Arguments,
			}); err != nil {
				return errors.Wrap(err, "render template")
			}

			moreDefs, err := Parse(out.String())
			if err != nil {
				return fmt.Errorf("Error parsing instantiated template. Error: %s.\nCode:\n\n%s", err, out.String())
			}
			delete(defs.MacroInstances, name)
			if err := defs.MergeFrom(moreDefs); err != nil {
				return errors.Wrap(err, "merging definitions")
			}
		}
	}
	return nil
}

func (defs *Definitions) MergeFrom(moreDefs *Definitions) error {
	defs.Imports = append(defs.Imports, moreDefs.Imports...)

	for name, def := range moreDefs.Macros {
		defs.Macros[name] = def
	}

	for name, def := range moreDefs.MacroInstances {
		defs.MacroInstances[name] = def
	}

	for name, def := range moreDefs.Libraries {
		defs.Libraries[name] = def
	}

	for name, schema := range moreDefs.Config {
		defs.Config[name] = schema
	}

	for eventName, newFns := range moreDefs.Events {
		if existingFns, ok := defs.Events[eventName]; ok {
			// Already has other listeners, add additional ones
			defs.Events[eventName] = append(existingFns, newFns...)
		} else {
			defs.Events[eventName] = newFns
		}
	}

	// For existing functions and jobs we attempt to merge their configuration blocks (if any)
	for name, def := range moreDefs.Functions {
		if _, ok := defs.Functions[name]; ok {
			// Exists, attempt merge
			var (
				map1 map[string]interface{}
				map2 map[string]interface{}
			)

			// First decode into a map
			if err := mapstructure.Decode(defs.Functions[name].Config, &map1); err != nil {
				return errors.Wrap(err, "could not map dest")
			}
			if err := mapstructure.Decode(def.Config, &map2); err != nil {
				return errors.Wrap(err, "could not map src")
			}

			// Then merge
			if err := mergo.Merge(&map1, map2, mergo.WithAppendSlice); err != nil {
				return errors.Wrap(err, "function merge")
			}

			// Then map back into a struct
			if err := mapstructure.Decode(map1, defs.Functions[name].Config); err != nil {
				return errors.Wrap(err, "map back")
			}
		} else {
			defs.Functions[name] = def
		}
	}

	for name, def := range moreDefs.Jobs {
		if _, ok := defs.Jobs[name]; ok {
			// Exists, attempt merge
			var (
				map1 map[string]interface{}
				map2 map[string]interface{}
			)

			// First decode into a map
			if err := mapstructure.Decode(defs.Jobs[name].Config, &map1); err != nil {
				return errors.Wrap(err, "could not map dest")
			}
			if err := mapstructure.Decode(def.Config, &map2); err != nil {
				return errors.Wrap(err, "could not map src")
			}

			// Then merge
			if err := mergo.Merge(&map1, map2, mergo.WithAppendSlice); err != nil {
				return errors.Wrap(err, "job merge")
			}

			// Then map back into a struct
			if err := mapstructure.Decode(map1, defs.Jobs[name].Config); err != nil {
				return errors.Wrap(err, "map back")
			}
		} else {
			defs.Jobs[name] = def
		}
	}

	return nil
}

func (defs *Definitions) InlineImports(currentPath string, cacheDir string) error {
	if cacheDir != "" {
		if err := os.MkdirAll(cacheDir, 0700); err != nil {
			return errors.Wrap(err, "create cache dir")
		}
	}
	// Track imported URLs
	allImports := map[string]bool{
		currentPath: true,
	}
importLoop:
	for len(defs.Imports) > 0 {
		for _, importUrl := range defs.Imports {
			if allImports[importUrl] {
				// Remove this one from the list
				newImports := []string{}
				for _, url := range defs.Imports {
					if url != importUrl {
						newImports = append(newImports, url)
					}
				}
				defs.Imports = newImports
				continue importLoop
			}
			log.Debug("Now importing ", importUrl)
			cachedPath := fmt.Sprintf("%s/%s", cacheDir, util.SafeFilename(importUrl))
			var fileContent string
			if strings.HasPrefix(importUrl, "./") || strings.HasPrefix(importUrl, "../") {
				if currentPath == "" {
					return errors.New("local imports not supported")
				}
				// Fetch from local path
				buf, err := os.ReadFile(filepath.Join(filepath.Dir(currentPath), importUrl))
				if err != nil {
					return errors.Wrap(err, "reading local file")
				}
				fileContent = string(buf)
			} else if _, err := os.Stat(cachedPath); err != nil {
				log.Info("Now fetching ", importUrl)
				// Fetch it now
				resp, err := http.Get(importUrl)
				if err != nil {
					return errors.Wrapf(err, "Error fetching %s", importUrl)
				}
				buf, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					return errors.Wrap(err, "reading body")
				}
				if resp.StatusCode != http.StatusOK {
					return fmt.Errorf("HTTP error (%d): %s", resp.StatusCode, buf)
				}
				if cacheDir != "" {
					if err := os.WriteFile(cachedPath, buf, 0600); err != nil {
						return errors.Wrap(err, "writing cached file")
					}
				}
				fileContent = string(buf)
			} else if cacheDir != "" {
				// Load from local file
				buf, err := os.ReadFile(cachedPath)
				if err != nil {
					return errors.Wrap(err, "reading cached file")
				}
				fileContent = string(buf)
			}
			moreDefs, err := Parse(fileContent)
			if err != nil {
				return fmt.Errorf("Error parsing imported URL (%s): %s", importUrl, err)
			}
			// Prefix imports with import path dir
			for i, imp := range moreDefs.Imports {
				moreDefs.Imports[i] = fmt.Sprintf("./%s", filepath.Join(filepath.Dir(importUrl), imp))
			}
			if err := defs.MergeFrom(moreDefs); err != nil {
				return errors.Wrap(err, "merging definitions during import")
			}
			allImports[importUrl] = true
		}
	}

	return nil
}
