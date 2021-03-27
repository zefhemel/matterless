package definition

import (
	"bytes"
	_ "embed"
	log "github.com/sirupsen/logrus"
	"strings"
	"text/template"
)

type FunctionID string
type FunctionInvokeFunc func(name FunctionID, event interface{}) interface{}

//go:embed template/definition.template
var markdownTemplate string

type Definitions struct {
	Environment map[string]string
	Functions   map[FunctionID]*FunctionDef
	Modules     map[string]*FunctionDef
	Jobs        map[FunctionID]*JobDef

	// Sources
	Events            map[string][]FunctionID
	MattermostClients map[string]*MattermostClientDef
	APIs              []*EndpointDef
	SlashCommands     map[string]*SlashCommandDef
	Bots              map[string]*BotDef
	Crons             []*CronDef
}

type FunctionDef struct {
	Name     string
	Language string
	Code     string
}

type JobDef struct {
	Name     string
	Language string
	Code     string
}

type MattermostClientDef struct {
	URL    string                  `yaml:"url"`
	Token  string                  `yaml:"token"`
	Events map[string][]FunctionID `yaml:"events"`
}

type BotDef struct {
	TeamNames   []string `yaml:"team_names"`
	Username    string   `yaml:"username"`
	DisplayName string   `yaml:"display_name"`
	Description string   `yaml:"description"`

	Events map[string][]FunctionID `yaml:"events"`
}

type EndpointDef struct {
	Path     string     `yaml:"path"`
	Methods  []string   `yaml:"methods"`
	Function FunctionID `yaml:"function"`
	// Not super happy with this solution, but allows you to put in custom behavior
	Decorate func(event *APIGatewayRequestEvent, invokeFunc FunctionInvokeFunc) *APIGatewayResponse
}

type APIDef []EndpointDef

type SlashCommandDef struct {
	TeamName string `yaml:"team_name"`
	Trigger  string `yaml:"trigger"`

	AutoComplete     bool   `yaml:"auto_complete"`
	AutoCompleteDesc string `yaml:"auto_complete_desc"`
	AutoCompleteHint string `yaml:"auto_complete_hint"`

	Function FunctionID `yaml:"function"`
}

type CronDef struct {
	Schedule string     `yaml:"schedule"`
	Function FunctionID `yaml:"function"`
}

// CompileFunctionCode appends all library code to a function to be eval'ed
func (decls *Definitions) CompileFunctionCode(code string) string {
	codeParts := []string{code}
	for _, libDef := range decls.Modules {
		codeParts = append(codeParts, libDef.Code)
	}
	return strings.Join(codeParts, "\n\n")
}

func (decls *Definitions) FunctionExists(id FunctionID) bool {
	_, found := decls.Functions[id]
	return found
}

func (decls *Definitions) Markdown() string {
	tmpl, err := template.New("sourceTemplate").Parse(markdownTemplate)
	if err != nil {
		log.Error("Could not render markdown:", err)
		return ""
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, decls); err != nil {
		log.Error("Could not render markdown:", err)
		return ""
	}
	return strings.TrimSpace(out.String())
}

func (defs *Definitions) ModulesForLanguage(lang string) map[string]string {
	codeMap := make(map[string]string)
	for name, def := range defs.Modules {
		// TODO: Implemenet filtering
		//if def.Language == lang {
		codeMap[name] = def.Code
		//}
	}
	return codeMap
}
