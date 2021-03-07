package declaration

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

var headerRegex = regexp.MustCompile("\\s*(\\w+)\\:?\\s*(.*)")

type yamlSubscription struct {
	Source   string   `yaml:"Source"`
	Function string   `yaml:"Function"`
	Events   []string `yaml:"Events"`
}

type yamlSource struct {
	Type  string `yaml:"Type"`
	URL   string `yaml:"URL"`
	Token string `yaml:"Token"`
}

// Parse uses the GoldMark Markdown parser to parse definitions
func Parse(code string) (*Declarations, error) {
	mdParser := goldmark.DefaultParser()

	decls := &Declarations{
		Functions:     map[string]*FunctionDef{},
		Sources:       map[string]*SourceDef{},
		Subscriptions: map[string]*SubscriptionDef{},
		Environment:   map[string]string{},
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
			decls.Functions[currentDeclarationName] = &FunctionDef{
				Name:     currentDeclarationName,
				Language: currentLanguage,
				Code:     currentBody,
			}
		case "Subscription":
			var yamlD yamlSubscription
			err := yaml.Unmarshal([]byte(currentBody), &yamlD)
			if err != nil {
				return err
			}
			decls.Subscriptions[currentDeclarationName] = &SubscriptionDef{
				Source:     yamlD.Source,
				Function:   yamlD.Function,
				EventTypes: yamlD.Events,
			}
		case "Source":
			var yamlS yamlSource
			err := yaml.Unmarshal([]byte(currentBody), &yamlS)
			if err != nil {
				return err
			}
			decls.Sources[currentDeclarationName] = &SourceDef{
				Type:  yamlS.Type,
				URL:   yamlS.URL,
				Token: yamlS.Token,
			}
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
