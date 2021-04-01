package definition

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/zefhemel/yamlschema"
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
