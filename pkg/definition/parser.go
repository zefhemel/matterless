package definition

import (
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

var headerRegex = regexp.MustCompile("\\s*(\\w+)\\:?\\s*(.*)")

type yamlSubscription struct {
	Function string   `yaml:"Function"`
	Source   string   `yaml:"Source"`
	Channel  string   `yaml:"Channel"`
	Events   []string `yaml:"Events"`
}

type yamlSource struct {
	URL   string `yaml:"URL"`
	Token string `yaml:"Token"`
}

// Parse uses the GoldMark Markdown parser to parse definitions
func Parse(posts []*model.Post) (Definitions, error) {
	mdParser := goldmark.DefaultParser()

	definitions := Definitions{
		Functions:     map[string]FunctionDef{},
		Sources:       map[string]SourceDef{},
		Subscriptions: map[string]SubscriptionDef{},
	}
	for _, p := range posts {
		message := []byte(p.Message)

		node := mdParser.Parse(text.NewReader(message))
		var (
			currentDeclarationType string
			currentDeclarationName string
			currentBody            string
			currentLanguage        string
		)
		processDefinition := func() error {
			switch currentDeclarationType {
			case "Function":
				definitions.Functions[currentDeclarationName] = FunctionDef{
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
				definitions.Subscriptions[currentDeclarationName] = SubscriptionDef{
					Source:     yamlD.Source,
					Function:   yamlD.Function,
					EventTypes: yamlD.Events,
					Channel:    yamlD.Channel,
				}
			case "Source":
				var yamlS yamlSource
				err := yaml.Unmarshal([]byte(currentBody), &yamlS)
				if err != nil {
					return err
				}
				definitions.Sources[currentDeclarationName] = SourceDef{
					URL:   yamlS.URL,
					Token: yamlS.Token,
				}
			}
			return nil
		}
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			switch v := c.(type) {
			case *ast.Heading:
				parts := headerRegex.FindStringSubmatch(string(v.Text(message)))
				currentDeclarationType = parts[1]
				currentDeclarationName = parts[2]
			case *ast.FencedCodeBlock:
				currentLanguage = string(v.Language(message))
				allCode := make([]string, 0, 10)
				for i := 0; i < v.Lines().Len(); i++ {
					seg := v.Lines().At(i)
					allCode = append(allCode, string(seg.Value(message)))
				}
				currentBody = strings.Join(allCode, "")
			case *ast.ThematicBreak:
				if err := processDefinition(); err != nil {
					return definitions, err
				}
				// Reset all
				currentBody = ""
				currentDeclarationName = ""
				currentDeclarationType = ""
				currentLanguage = ""
			}
		}
		if err := processDefinition(); err != nil {
			return definitions, err
		}
	}
	return definitions, nil
}
