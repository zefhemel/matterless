package definition

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/config"
	"gopkg.in/yaml.v3"
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
	Init             interface{} `yaml:"init" json:"init,omitempty"`
	Runtime          string      `yaml:"runtime" json:"runtime,omitempty"`
	DesiredInstances int         `yaml:"desired_instances,omitempty"`

	DockerImage string `yaml:"docker_image" json:"docker_image,omitempty"`
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
	ArgumentsSchema map[string]interface{} `yaml:"schema" json:"schema"`
}

type MacroInstanceDef struct {
	Macro     MacroID     `json:"macro"`
	Arguments interface{} `json:"arguments"`
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

func Check(code string, cfg *config.Config) (*Definitions, error) {
	defs, err := Parse(code)
	if err != nil {
		return nil, err
	}

	if err := defs.InlineImports(fmt.Sprintf("%s/.importcache", cfg.DataDir)); err != nil {
		return nil, err
	}
	if err := defs.ExpandMacros(); err != nil {
		return nil, err
	}
	return defs, nil
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
