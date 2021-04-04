package definition

import (
	"bytes"
	"fmt"
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

func (defs *Definitions) Check() error {
	for name, def := range defs.CustomDef {
		tmpl, ok := defs.Macros[def.Macro]
		if !ok {
			return fmt.Errorf("No such template: %s", def.Macro)
		}
		if err := yamlschema.ValidateObjects(tmpl.Config.InputSchema, def.Input); err != nil {
			return fmt.Errorf("[%s] %s", name, err)
		}
	}
	return nil
}

func (defs *Definitions) Desugar() error {
	if err := defs.Check(); err != nil {
		return err
	}
	for name, def := range defs.CustomDef {
		tmpl, ok := defs.Macros[def.Macro]
		if !ok {
			return fmt.Errorf("No such template: %s", def.Macro)
		}
		t := template.New("template")
		t.Funcs(CodeGenFuncs)
		t2, err := t.Parse(fmt.Sprintf(`
{{- $name := .Name -}}
{{- $input := .Input -}}
%s`, tmpl.TemplateCode))
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
		defs.CustomDef = map[string]*CustomDef{}
		defs.MergeFrom(moreDefs)
	}
	if len(defs.CustomDef) > 0 {
		// Recursively desugar if there's new custom definitions defined
		return defs.Desugar()
	}
	return nil
}

func (defs *Definitions) MergeFrom(moreDefs *Definitions) {
	for name, def := range moreDefs.Macros {
		defs.Macros[name] = def
	}

	for name, def := range moreDefs.CustomDef {
		defs.CustomDef[name] = def
	}

	for eventName, newFns := range moreDefs.Events {
		if existingFns, ok := defs.Events[eventName]; ok {
			// Already has other listeners, add additional ones
			defs.Events[eventName] = append(existingFns, newFns...)
		} else {
			defs.Events[eventName] = newFns
		}
	}

	for name, def := range moreDefs.Functions {
		defs.Functions[name] = def
	}

	for name, def := range moreDefs.Jobs {
		defs.Jobs[name] = def
	}
}

func (defs *Definitions) InlineImports(cacheDir string) error {
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return errors.Wrap(err, "create cache dir")
	}
	for _, importUrl := range defs.Imports {
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
		defs.Imports = []string{}
		defs.MergeFrom(moreDefs)
	}
	return nil
}
