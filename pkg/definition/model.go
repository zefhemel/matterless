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
type MacroID string

type FunctionInvokeFunc func(name FunctionID, event interface{}) interface{}

//go:embed template/summary.template
var summaryTemplate string

//go:embed template/model.template
var rerenderTemplate string

type Definitions struct {
	Imports        []string
	Functions      map[FunctionID]*FunctionDef
	Jobs           map[FunctionID]*JobDef
	Events         map[string][]FunctionID
	Macros         map[MacroID]*MacroDef
	MacroInstances map[string]*MacroInstanceDef
}

type FunctionConfig struct {
	Init        interface{} `yaml:"init,omitempty"`
	Runtime     string      `yaml:"runtime,omitempty"`
	DockerImage string      `yaml:"docker_image,omitempty"`
}

type FunctionDef struct {
	Name     string
	Config   *FunctionConfig
	Language string
	Code     string
}

type JobDef struct {
	Name     string
	Config   *FunctionConfig
	Language string
	Code     string
}

type MacroDef struct {
	Config       MacroConfig
	TemplateCode string
}

type MacroConfig struct {
	InputSchema map[string]interface{} `yaml:"input_schema"`
}

type MacroInstanceDef struct {
	Macro MacroID
	Input interface{}
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
