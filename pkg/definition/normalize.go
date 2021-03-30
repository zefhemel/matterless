package definition

import (
	"bytes"
	"fmt"
	"github.com/buildkite/interpolate"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/xeipuuv/gojsonschema"
	"github.com/zefhemel/matterless/pkg/util"
	"gopkg.in/yaml.v3"
	"strings"
	"text/template"
)

// Normalize replaces environment variables with their values
func (defs *Definitions) Normalize() {
	mapEnv := interpolate.NewMapEnv(defs.Config)
	for k, v := range defs.Config {
		if newV, err := interpolate.Interpolate(mapEnv, v); err == nil {
			defs.Config[k] = newV
		}
	}

	for _, def := range defs.Jobs {
		for k, v := range def.Config.Config {
			yamlBuf, _ := yaml.Marshal(v)
			if interPolatedYaml, err := interpolate.Interpolate(mapEnv, string(yamlBuf)); err == nil {
				var val interface{}
				yaml.Unmarshal([]byte(interPolatedYaml), &val)
				def.Config.Config[k] = val
			}
		}
	}
	for _, def := range defs.Functions {
		for k, v := range def.Config.Config {
			yamlBuf, _ := yaml.Marshal(v)
			if interPolatedYaml, err := interpolate.Interpolate(mapEnv, string(yamlBuf)); err == nil {
				var val interface{}
				yaml.Unmarshal([]byte(interPolatedYaml), &val)
				def.Config.Config[k] = val
			}
		}
	}

	for _, def := range defs.CustomDef {
		yamlBuf, _ := yaml.Marshal(def.Input)
		if interPolatedYaml, err := interpolate.Interpolate(mapEnv, string(yamlBuf)); err == nil {
			var val interface{}
			yaml.Unmarshal([]byte(interPolatedYaml), &val)
			def.Input = val
		}
	}
}

func (defs *Definitions) Check() error {
	for name, def := range defs.CustomDef {
		tmpl, ok := defs.Template[def.Template]
		if !ok {
			return fmt.Errorf("No such template: %s", def.Template)
		}
		schema := tmpl.Config.InputSchema
		schema["$schema"] = "https://json-schema.org/draft/2020-12/schema"
		schema["$id"] = "https://matterless.dev/app"

		schemaLoader := gojsonschema.NewStringLoader(util.MustJsonString(schema))

		// And then serialize it to JSON to be ready by the validator. Crazy times!
		jsonObjectLoader := gojsonschema.NewStringLoader(util.MustJsonString(def.Input))
		result, err := gojsonschema.Validate(schemaLoader, jsonObjectLoader)
		if err != nil {
			return errors.Wrap(err, "validation")
		}
		if result.Valid() {
			return nil
		} else {
			errorItems := []string{}
			for _, err := range result.Errors() {
				errorItems = append(errorItems, fmt.Sprintf("- %s", err.String()))
			}
			return fmt.Errorf("[%s] Validation errors:\n\n%s", name, strings.Join(errorItems, "\n"))
		}
	}
	return nil
}

func (defs *Definitions) Desugar() error {
	if err := defs.Check(); err != nil {
		return err
	}
	for name, def := range defs.CustomDef {
		tmpl, ok := defs.Template[def.Template]
		if !ok {
			return fmt.Errorf("No such template: %s", def.Template)
		}
		t, err := template.New("template").Parse(fmt.Sprintf(`
{{- $name := .Name -}}
{{- $input := .Input -}}
%s`, tmpl.Template))
		if err != nil {
			return errors.Wrap(err, "parsing template")
		}
		var out bytes.Buffer
		if err := t.Execute(&out, struct {
			Name  string
			Input interface{}
		}{
			Name:  name,
			Input: def.Input,
		}); err != nil {
			return errors.Wrap(err, "render template")
		}

		log.Info("GOt this out: ", out.String())

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
	for name, def := range moreDefs.Template {
		defs.Template[name] = def
	}

	for name, def := range moreDefs.CustomDef {
		defs.CustomDef[name] = def
	}

	for k, v := range moreDefs.Config {
		defs.Config[k] = v
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

	for name, def := range moreDefs.Modules {
		defs.Modules[name] = def
	}
}
