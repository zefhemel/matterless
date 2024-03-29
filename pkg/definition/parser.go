package definition

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"github.com/zefhemel/matterless/pkg/util"
	"regexp"
	"strings"
)

var eventsSchema = MustNewSchema(`
type: object
additionalProperties:
  type: array
  items:
    type: string
`)

var headerRegex = regexp.MustCompile("^\\s*([a-z][\\w\\.]+)\\s*(.*)")

// Parse uses the GoldMark Markdown parser to parse definitions
func Parse(code string) (*Definitions, error) {
	mdParser := goldmark.DefaultParser()

	decls := NewDefinitions()
	codeBytes := []byte(code)
	node := mdParser.Parse(text.NewReader(codeBytes))
	var (
		currentDeclarationType string
		currentDeclarationName string
		currentBody            string
		currentBody2           string
		currentCodeBlock       string
		currentLanguage        string
		listItems              []string
	)
	processDefinition := func() error {
		switch currentDeclarationType {
		case "":
			// Skipping
		case "function", "func":
			funcDef := &FunctionDef{
				Name:     currentDeclarationName,
				Language: currentLanguage,
				Config:   &FunctionConfig{},
			}
			if currentBody2 != "" {
				// We got a parameter clause on our hands, parse the currentBody as YAML
				if err := util.StrictYamlUnmarshal(currentBody, &funcDef.Config); err != nil {
					return fmt.Errorf("Function %s: %s", currentDeclarationName, err)
				}
				// And the second block will be the code
				funcDef.Code = currentBody2
			} else {
				// No parameter clause
				funcDef.Code = currentBody
			}
			if funcDef.Config.Instances == 0 {
				funcDef.Config.Instances = 1
			}
			if currentDeclarationName == "" {
				return fmt.Errorf("functions should have a name")
			}
			decls.Functions[FunctionID(currentDeclarationName)] = funcDef
		case "job":
			jobDef := &JobDef{
				Name:     currentDeclarationName,
				Language: currentLanguage,
				Config:   &JobConfig{},
			}
			if currentBody2 != "" {
				// We got a parameter clause on our hands, parse the currentBody as YAML
				if err := util.StrictYamlUnmarshal(currentBody, &jobDef.Config); err != nil {
					return fmt.Errorf("Job %s: %s", currentDeclarationName, err)
				}
				// And the second block will be the code
				jobDef.Code = currentBody2
			} else {
				// No parameter clause
				jobDef.Code = currentBody
			}
			if jobDef.Config.Instances == 0 {
				jobDef.Config.Instances = 1
			}
			if currentDeclarationName == "" {
				return fmt.Errorf("jobs should have a name")
			}
			decls.Jobs[FunctionID(currentDeclarationName)] = jobDef
		case "library":
			libraryDef := &LibraryDef{
				Name:     currentDeclarationName,
				Language: currentLanguage,
				Code:     currentBody,
			}
			if currentDeclarationName == "" {
				return fmt.Errorf("libraries should have a filename")
			}
			decls.Libraries[FunctionID(currentDeclarationName)] = libraryDef
		case "events":
			// TODO: Unmarshalling twice now, could use struct mapping
			if err := eventsSchema.ValidateString(currentBody); err != nil {
				return fmt.Errorf("Events: %s", err)
			}
			var def map[string][]FunctionID

			if err := util.StrictYamlUnmarshal(currentBody, &def); err != nil {
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
		case "macro":
			var config MacroConfig
			if len(currentBody) > 0 {
				err := util.StrictYamlUnmarshal(currentBody, &config)
				if err != nil {
					return err
				}
			}
			if strings.ToLower(currentDeclarationName[0:1]) != currentDeclarationName[0:1] {
				return errors.New("All macros should start with a lower-case letter")
			}
			if currentDeclarationName == "" {
				return fmt.Errorf("macros should have a name")
			}
			decls.Macros[MacroID(currentDeclarationName)] = &MacroDef{
				Config:       config,
				TemplateCode: currentCodeBlock,
			}
		case "config":
			var def map[string]*TypeSchema
			err := util.StrictYamlUnmarshal(currentBody, &def)
			if err != nil {
				return err
			}
			// Merge into other config blocks
			for name, schema := range def {
				decls.Config[name] = schema
			}
		case "import", "imports":
			decls.Imports = append(decls.Imports, listItems...)
		default: // May be a custom one, let's try
			var inputs interface{}
			err := util.StrictYamlUnmarshal(currentBody, &inputs)
			if err != nil {
				return fmt.Errorf("[%s] %s: Could not parse YAML", currentDeclarationType, currentDeclarationName)
			}
			decls.MacroInstances[currentDeclarationName] = &MacroInstanceDef{
				Macro:     MacroID(currentDeclarationType),
				Arguments: inputs,
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
			listItems = []string{}
			// Process next
			parts := headerRegex.FindStringSubmatch(string(v.Text(codeBytes)))
			if parts == nil || len(parts) == 0 {
				// Ignore, not a Matterless definition
				currentDeclarationType = ""
				currentDeclarationName = ""
			} else {
				currentDeclarationType = parts[1]
				currentDeclarationName = parts[2]
			}
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
		case *ast.List:
			currentChild := v.FirstChild()
			for currentChild != nil {
				listItems = append(listItems, strings.TrimSpace(string(currentChild.Text(codeBytes))))
				currentChild = currentChild.NextSibling()
			}
		}
	}
	if err := processDefinition(); err != nil {
		return decls, err
	}

	return decls, nil
}

func NewDefinitions() *Definitions {
	return &Definitions{
		Config:         map[string]*TypeSchema{},
		Functions:      map[FunctionID]*FunctionDef{},
		Jobs:           map[FunctionID]*JobDef{},
		Libraries:      map[FunctionID]*LibraryDef{},
		Events:         map[string][]FunctionID{},
		Macros:         map[MacroID]*MacroDef{},
		MacroInstances: map[string]*MacroInstanceDef{},
	}
}
