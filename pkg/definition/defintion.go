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
	Events      map[string][]FunctionID
}

type FunctionConfig struct {
	Config      map[string]interface{} `yaml:"config"`
	DockerImage string                 `yaml:"docker_image"`
}

type FunctionDef struct {
	Name     string
	Language string
	Config   FunctionConfig
	Code     string
}

type JobDef struct {
	Name     string
	Language string
	Config   FunctionConfig
	Code     string
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
