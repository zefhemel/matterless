package definition

import (
	"bytes"
	"fmt"
	"github.com/imdario/mergo"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/util"
	"github.com/zefhemel/yamlschema"
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"
)

func (defs *Definitions) ExpandMacros() error {
	for len(defs.MacroInstances) > 0 {
		for name, def := range defs.MacroInstances {
			macro, ok := defs.Macros[def.Macro]
			if !ok {
				return fmt.Errorf("No such macro: %s", def.Macro)
			}

			if err := yamlschema.ValidateObjects(macro.Config.InputSchema, def.Input); err != nil {
				return errors.Wrapf(err, "macro expansion: %s: %s", name, err)
			}

			t := template.New("template")
			t.Funcs(CodeGenFuncs)
			t2, err := t.Parse(fmt.Sprintf(`
{{- $name := .Name -}}
{{- $input := .Input -}}
%s`, macro.TemplateCode))
			if err != nil {
				return errors.Wrap(err, "parsing template")
			}
			var out bytes.Buffer
			if err := t2.Execute(&out, struct {
				Name  string
				Input interface{}
			}{
				Name:  name,
				Input: def.Input,
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

func (defs *Definitions) InlineImports(cacheDir string) error {
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return errors.Wrap(err, "create cache dir")
	}
	// Track imported URLs
	allImports := map[string]bool{}
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
			if strings.HasPrefix(importUrl, "file:") {
				// Fetch from local path
				filePath := importUrl[len("file:"):]
				buf, err := os.ReadFile(filePath)
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
				if err := os.WriteFile(cachedPath, buf, 0600); err != nil {
					return errors.Wrap(err, "writing cached file")
				}
				fileContent = string(buf)
			} else {
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
			if err := defs.MergeFrom(moreDefs); err != nil {
				return errors.Wrap(err, "merging definitions during import")
			}
			allImports[importUrl] = true
		}
	}

	return nil
}
