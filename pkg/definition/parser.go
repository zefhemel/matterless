package definition

import (
	"embed"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"github.com/zefhemel/yamlschema"
	"gopkg.in/yaml.v3"
)

//go:embed schema/*.schema.json
var jsonSchemas embed.FS

func Validate(schemaName string, yamlString string) error {
	schemaBytes, err := jsonSchemas.ReadFile(schemaName)
	if err != nil {
		log.Fatal(err)
	}
	return yamlschema.ValidateStrings(string(schemaBytes), yamlString)
}

func ValidateObj(schemaName string, obj interface{}) error {
	schemaBytes, err := jsonSchemas.ReadFile(schemaName)
	if err != nil {
		log.Fatal(err)
	}
	var yamlSchema map[string]interface{}
	if err := yaml.Unmarshal(schemaBytes, &yamlSchema); err != nil {
		return errors.Wrap(err, "parsing schema")
	}
	return yamlschema.ValidateObjects(yamlSchema, obj)
}

var headerRegex = regexp.MustCompile("\\s*([\\w\\.]+)\\:?\\s*(.*)")

// Parse uses the GoldMark Markdown parser to parse definitions
func Parse(code string) (*Definitions, error) {
	mdParser := goldmark.DefaultParser()

	decls := &Definitions{
		Functions: map[FunctionID]*FunctionDef{},
		Jobs:      map[FunctionID]*JobDef{},
		Events:    map[string][]FunctionID{},
		Macros:    map[MacroID]*MacroDef{},
		CustomDef: map[string]*CustomDef{},
	}
	codeBytes := []byte(code)
	node := mdParser.Parse(text.NewReader(codeBytes))
	var (
		currentDeclarationType string
		currentDeclarationName string
		currentBody            string
		currentBody2           string
		currentCodeBlock       string
		currentLanguage        string
	)
	processDefinition := func() error {
		switch currentDeclarationType {
		case "Function":
			funcDef := &FunctionDef{
				Name:     currentDeclarationName,
				Language: currentLanguage,
				Config:   FunctionConfig{},
			}
			if currentBody2 != "" {
				// We got a parameter clause on our hands, parse the currentBody as YAML
				if err := yaml.Unmarshal([]byte(currentBody), &funcDef.Config); err != nil {
					return fmt.Errorf("Function %s: %s", currentDeclarationName, err)
				}
				// And the second block will be the code
				funcDef.Code = currentBody2
			} else {
				// No parameter clause
				funcDef.Code = currentBody
			}
			decls.Functions[FunctionID(currentDeclarationName)] = funcDef
		case "Job":
			jobDef := &JobDef{
				Name:     currentDeclarationName,
				Language: currentLanguage,
				Config:   FunctionConfig{},
			}
			if currentBody2 != "" {
				// We got a parameter clause on our hands, parse the currentBody as YAML
				if err := yaml.Unmarshal([]byte(currentBody), &jobDef.Config); err != nil {
					return fmt.Errorf("Job %s: %s", currentDeclarationName, err)
				}
				// And the second block will be the code
				jobDef.Code = currentBody2
			} else {
				// No parameter clause
				jobDef.Code = currentBody
			}
			decls.Jobs[FunctionID(currentDeclarationName)] = jobDef
		case "Events":
			var def map[string][]FunctionID
			if err := Validate("schema/events.schema.json", currentBody); err != nil {
				return fmt.Errorf("Events: %s", err)
			}
			err := yaml.Unmarshal([]byte(currentBody), &def)
			if err != nil {
				return err
			}
			// Merge into other Events blocks
			for eventName, newFns := range def {
				if existingFns, ok := decls.Events[eventName]; ok {
					// Already has other listeners, add additional ones
					decls.Events[eventName] = append(existingFns, newFns...)
				} else {
					decls.Events[eventName] = newFns
				}
			}
		case "Macro":
			var config MacroConfig
			err := yaml.Unmarshal([]byte(currentBody), &config)
			if err != nil {
				return err
			}
			if err := ValidateObj("schema/schema.schema.json", config.InputSchema); err != nil {
				return fmt.Errorf("Macros %s: %s", currentDeclarationName, err)
			}
			decls.Macros[MacroID(currentDeclarationName)] = &MacroDef{
				Config:       config,
				TemplateCode: currentCodeBlock,
			}
		default: // May be a custom one, let's try
			if currentBody == "" {
				// Not a template instatiation
				return nil
			}
			var inputs interface{}
			err := yaml.Unmarshal([]byte(currentBody), &inputs)
			if err != nil {
				return fmt.Errorf("[%s] %s: Could not parse YAML", currentDeclarationType, currentDeclarationName)
			}
			decls.CustomDef[currentDeclarationName] = &CustomDef{
				Macro: MacroID(currentDeclarationType),
				Input: inputs,
			}
		}
		return nil
	}
	for c := node.FirstChild(); c != nil; c = c.NextSibling() {
		switch v := c.(type) {
		case *ast.Heading:
			if err := processDefinition(); err != nil {
				return decls, err
			}
			// reset all
			currentBody = ""
			currentBody2 = ""
			currentLanguage = ""
			currentCodeBlock = ""
			// Process next
			parts := headerRegex.FindStringSubmatch(string(v.Text(codeBytes)))
			currentDeclarationType = parts[1]
			currentDeclarationName = parts[2]
		case *ast.FencedCodeBlock:
			currentLanguage = string(v.Language(codeBytes))
			allCode := make([]string, 0, 10)
			for i := 0; i < v.Lines().Len(); i++ {
				seg := v.Lines().At(i)
				allCode = append(allCode, string(seg.Value(codeBytes)))
			}
			if currentBody != "" {
				currentBody2 = strings.Join(allCode, "")
			} else {
				currentBody = strings.Join(allCode, "")
			}
		case *ast.CodeBlock:
			// Indented code block (for templates)
			allCode := make([]string, 0, 10)
			for i := 0; i < v.Lines().Len(); i++ {
				seg := v.Lines().At(i)
				allCode = append(allCode, string(seg.Value(codeBytes)))
			}
			currentCodeBlock = strings.Join(allCode, "")
		}
	}
	if err := processDefinition(); err != nil {
		return decls, err
	}

	return decls, nil
}
