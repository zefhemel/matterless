package definition

import (
	"bytes"
	_ "embed"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"strings"
	"text/template"
)

type FunctionID string
type FunctionInvokeFunc func(name FunctionID, event interface{}) interface{}

//go:embed template/summary.template
var summaryTemplate string

//go:embed template/model.template
var rerenderTemplate string

type Definitions struct {
	Config    map[string]string
	Functions map[FunctionID]*FunctionDef
	Jobs      map[FunctionID]*JobDef
	Modules   map[string]*FunctionDef
	Events    map[string][]FunctionID
	Macros    map[MacroID]*MacroDef
	CustomDef map[string]*CustomDef
}

type FunctionConfig struct {
	Init        map[string]interface{} `yaml:"init"`
	Runtime     string                 `yaml:"runtime"`
	DockerImage string                 `yaml:"docker_image"`
}

type FunctionDef struct {
	Name     string
	Config   FunctionConfig
	Language string
	Code     string
}

type JobDef struct {
	Name     string
	Config   FunctionConfig
	Language string
	Code     string
}

type MacroID string

type MacroDef struct {
	Config       MacroConfig
	TemplateCode string
}

type MacroConfig struct {
	InputSchema map[string]interface{} `yaml:"input_schema"`
}

type CustomDef struct {
	Macro MacroID
	Input interface{}
}

func (defs *Definitions) FunctionExists(id FunctionID) bool {
	_, found := defs.Functions[id]
	return found
}

var CodeGenFuncs = template.FuncMap{
	"yaml": func(obj interface{}) string {
		dataBuf, err := yaml.Marshal(obj)
		if err != nil {
			return "YAML-ERROR"
		}
		return string(dataBuf)
	},
	"prefixLines": func(prefix string, s string) string {
		lines := strings.Split(s, "\n")
		for i := range lines {
			if i > 0 {
				lines[i] = prefix + lines[i]
			}
		}
		return strings.Join(lines, "\n")
	},
}

func (defs *Definitions) Markdown() string {
	t := template.New("sourceTemplate")
	t.Funcs(CodeGenFuncs)
	tmpl, err := t.Parse(rerenderTemplate)
	if err != nil {
		log.Error("Could not render markdown:", err)
		return ""
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, defs); err != nil {
		log.Error("Could not render markdown:", err)
		return ""
	}
	return strings.TrimSpace(out.String())
}

func (defs *Definitions) SummaryMarkdown() string {
	tmpl, err := template.New("sourceTemplate").Parse(summaryTemplate)
	if err != nil {
		log.Error("Could not render markdown:", err)
		return ""
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, defs); err != nil {
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
