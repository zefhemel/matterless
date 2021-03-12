package definition

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

var headerRegex = regexp.MustCompile("\\s*(\\w+)\\:?\\s*(.*)")

// Parse uses the GoldMark Markdown parser to parse definitions
func Parse(code string) (*Definitions, error) {
	mdParser := goldmark.DefaultParser()

	decls := &Definitions{
		Functions:         map[FunctionID]*FunctionDef{},
		MattermostClients: map[string]*MattermostClientDef{},
		APIGateways:       map[string]*APIGatewayDef{},
		Environment:       map[string]string{},
		Libraries:         map[string]*FunctionDef{},
	}
	codeBytes := []byte(code)
	node := mdParser.Parse(text.NewReader(codeBytes))
	var (
		currentDeclarationType string
		currentDeclarationName string
		currentBody            string
		currentLanguage        string
	)
	processDefinition := func() error {
		switch currentDeclarationType {
		case "Function":
			decls.Functions[FunctionID(currentDeclarationName)] = &FunctionDef{
				Name:     currentDeclarationName,
				Language: currentLanguage,
				Code:     currentBody,
			}
		case "Library":
			decls.Libraries[currentDeclarationName] = &FunctionDef{
				Name:     currentDeclarationName,
				Language: currentLanguage,
				Code:     currentBody,
			}
		case "MattermostClient":
			var def MattermostClientDef
			err := yaml.Unmarshal([]byte(currentBody), &def)
			if err != nil {
				return err
			}
			decls.MattermostClients[currentDeclarationName] = &def
		case "APIGateway":
			var def APIGatewayDef
			err := yaml.Unmarshal([]byte(currentBody), &def)
			if err != nil {
				return err
			}
			decls.APIGateways[currentDeclarationName] = &def
		case "Environment":
			err := yaml.Unmarshal([]byte(currentBody), &decls.Environment)
			if err != nil {
				return err
			}
		}
		return nil
	}
	for c := node.FirstChild(); c != nil; c = c.NextSibling() {
		switch v := c.(type) {
		case *ast.Heading:
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
			currentBody = strings.Join(allCode, "")
		case *ast.ThematicBreak:
			if err := processDefinition(); err != nil {
				return decls, err
			}
			// Reset all
			currentBody = ""
			currentDeclarationName = ""
			currentDeclarationType = ""
			currentLanguage = ""
		}
	}
	if err := processDefinition(); err != nil {
		return decls, err
	}

	return decls, nil
}
