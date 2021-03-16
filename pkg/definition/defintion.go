package definition

import "strings"

type FunctionID string

type Definitions struct {
	Environment map[string]string
	Functions   map[FunctionID]*FunctionDef
	Libraries   map[string]*FunctionDef // For now we just support the empty library name

	// Sources
	MattermostClients map[string]*MattermostClientDef
	APIGateways       map[string]*APIGatewayDef
	SlashCommands     map[string]*SlashCommandDef
	Bots              map[string]*BotDef
	Crons             map[string]*CronDef
}

type FunctionDef struct {
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
	// Not super happy with this solution, but allows you to pre and post process the function invocation
	PreProcess  func(event interface{}) interface{}
	PostProcess func(result interface{}) interface{}
}

type APIGatewayDef struct {
	Endpoints []EndpointDef `yaml:"endpoints"`
}

type SlashCommandDef struct {
	TeamName string     `yaml:"team_name"`
	Trigger  string     `yaml:"trigger"`
	Function FunctionID `yaml:"function"`
}

type CronDef struct {
	Schedule string     `yaml:"schedule"`
	Function FunctionID `yaml:"function"`
}

// CompileFunctionCode appends all library code to a function to be eval'ed
func (decls *Definitions) CompileFunctionCode(code string) string {
	codeParts := []string{code}
	for _, libDef := range decls.Libraries {
		codeParts = append(codeParts, libDef.Code)
	}
	return strings.Join(codeParts, "\n\n")
}

func (decls *Definitions) FunctionExists(id FunctionID) bool {
	_, found := decls.Functions[id]
	return found
}
