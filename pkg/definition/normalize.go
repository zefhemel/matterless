package definition

import (
	"bytes"
	"fmt"
	"github.com/buildkite/interpolate"
	"github.com/pkg/errors"
	"github.com/zefhemel/yamlschema"
	"gopkg.in/yaml.v3"
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
		for k, v := range def.Config.Init {
			yamlBuf, _ := yaml.Marshal(v)
			if interPolatedYaml, err := interpolate.Interpolate(mapEnv, string(yamlBuf)); err == nil {
				var val interface{}
				yaml.Unmarshal([]byte(interPolatedYaml), &val)
				def.Config.Init[k] = val
			}
		}
	}
	for _, def := range defs.Functions {
		for k, v := range def.Config.Init {
			yamlBuf, _ := yaml.Marshal(v)
			if interPolatedYaml, err := interpolate.Interpolate(mapEnv, string(yamlBuf)); err == nil {
				var val interface{}
				yaml.Unmarshal([]byte(interPolatedYaml), &val)
				def.Config.Init[k] = val
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
