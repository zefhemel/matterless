package definition

import (
	"embed"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/xeipuuv/gojsonschema"
	"github.com/zefhemel/matterless/pkg/util"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

//go:embed schema/*.schema.json
var jsonSchemas embed.FS

func Validate(schemaName string, yamlString string) error {
	schemaBytes, err := jsonSchemas.ReadFile(schemaName)
	if err != nil {
		log.Fatal(err)
	}
	schemaLoader := gojsonschema.NewStringLoader(string(schemaBytes))

	// We're going to parse YAML
	var obj interface{}
	if err := yaml.Unmarshal([]byte(yamlString), &obj); err != nil {
		return errors.Wrap(err, "parsing yaml")
	}
	// And then serialize it to JSON to be ready by the validator. Crazy times!
	jsonObjectLoader := gojsonschema.NewStringLoader(util.MustJsonString(obj))
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
		return fmt.Errorf("Validation errors:\n\n%s", strings.Join(errorItems, "\n"))
	}
}

var headerRegex = regexp.MustCompile("\\s*(\\w+)\\:?\\s*(.*)")

// Parse uses the GoldMark Markdown parser to parse definitions
func Parse(code string) (*Definitions, error) {
	mdParser := goldmark.DefaultParser()

	decls := &Definitions{
		Functions:         map[FunctionID]*FunctionDef{},
		MattermostClients: map[string]*MattermostClientDef{},
		APIs:              []*EndpointDef{},
		SlashCommands:     map[string]*SlashCommandDef{},
		Bots:              map[string]*BotDef{},
		Crons:             []*CronDef{},
		Environment:       map[string]string{},
		Modules:           map[string]*FunctionDef{},
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
		case "Module":
			decls.Modules[currentDeclarationName] = &FunctionDef{
				Name:     currentDeclarationName,
				Language: currentLanguage,
				Code:     currentBody,
			}
		case "MattermostClient":
			var def MattermostClientDef
			if err := Validate("schema/mattermost_client.schema.json", currentBody); err != nil {
				return fmt.Errorf("MattermostClient (%s): %s", currentDeclarationName, err)
			}
			err := yaml.Unmarshal([]byte(currentBody), &def)
			if err != nil {
				return err
			}
			decls.MattermostClients[currentDeclarationName] = &def
		case "API":
			var def []*EndpointDef
			if err := Validate("schema/api.schema.json", currentBody); err != nil {
				return fmt.Errorf("API: %s", err)
			}
			err := yaml.Unmarshal([]byte(currentBody), &def)
			if err != nil {
				return err
			}
			decls.APIs = def
		case "SlashCommand":
			var def SlashCommandDef
			if err := Validate("schema/slashcommand.schema.json", currentBody); err != nil {
				return fmt.Errorf("SlashCommand (%s): %s", currentDeclarationName, err)
			}
			err := yaml.Unmarshal([]byte(currentBody), &def)
			if err != nil {
				return err
			}
			decls.SlashCommands[currentDeclarationName] = &def
		case "Bot":
			var def BotDef
			if err := Validate("schema/bot.schema.json", currentBody); err != nil {
				return fmt.Errorf("Bot (%s): %s", currentDeclarationName, err)
			}
			err := yaml.Unmarshal([]byte(currentBody), &def)
			if err != nil {
				return err
			}
			decls.Bots[currentDeclarationName] = &def
		case "Cron":
			var def []*CronDef
			if err := Validate("schema/cron.schema.json", currentBody); err != nil {
				return fmt.Errorf("Cron: %s", err)
			}
			err := yaml.Unmarshal([]byte(currentBody), &def)
			if err != nil {
				return err
			}
			decls.Crons = def
		case "Environment":
			err := yaml.Unmarshal([]byte(currentBody), &decls.Environment)
			if err := Validate("schema/environment.schema.json", currentBody); err != nil {
				return fmt.Errorf("Environment: %s", err)
			}
			if err != nil {
				return err
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
			currentLanguage = ""
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
			currentBody = strings.Join(allCode, "")
		}
	}
	if err := processDefinition(); err != nil {
		return decls, err
	}

	return decls, nil
}
