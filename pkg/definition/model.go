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
	Imports        []string                     `json:"imports,omitempty"`
	Functions      map[FunctionID]*FunctionDef  `json:"functions"`
	Jobs           map[FunctionID]*JobDef       `json:"jobs"`
	Events         map[string][]FunctionID      `json:"events"`
	Macros         map[MacroID]*MacroDef        `json:"macros"`
	MacroInstances map[string]*MacroInstanceDef `json:"macro_instances,omitempty"`
}

type FunctionConfig struct {
	Init        interface{} `yaml:"init" json:"init,omitempty"`
	Runtime     string      `yaml:"runtime" json:"runtime,omitempty"`
	DockerImage string      `yaml:"docker_image" json:"docker_image,omitempty"`
}

type FunctionDef struct {
	Name     string          `json:"name"`
	Config   *FunctionConfig `json:"config,omitempty"`
	Language string          `json:"language,omitempty"`
	Code     string          `json:"code,omitempty"`
}

type JobDef struct {
	Name     string          `json:"name"`
	Config   *FunctionConfig `json:"config,omitempty"`
	Language string          `json:"language,omitempty"`
	Code     string          `json:"code,omitempty"`
}

type MacroDef struct {
	Config       MacroConfig `json:"config"`
	TemplateCode string      `json:"template_code"`
}

type MacroConfig struct {
	InputSchema map[string]interface{} `yaml:"input_schema" json:"input_schema"`
}

type MacroInstanceDef struct {
	Macro MacroID     `json:"macro"`
	Input interface{} `json:"input"`
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
