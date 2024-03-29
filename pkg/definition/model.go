package definition

import (
	"bytes"
	_ "embed"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type FunctionID string
type MacroID string

type FunctionInvokeFunc func(name FunctionID, event interface{}) interface{}

//go:embed template/summary.template
var summaryTemplate string

//go:embed template/model.template
var rerenderTemplate string

type LibraryMap = map[FunctionID]*LibraryDef

type Definitions struct {
	Imports        []string                     `json:"imports,omitempty"`
	Config         map[string]*TypeSchema       `json:"config,omitempty"`
	Functions      map[FunctionID]*FunctionDef  `json:"functions"`
	Jobs           map[FunctionID]*JobDef       `json:"jobs"`
	Libraries      LibraryMap                   `json:"libraries"`
	Events         map[string][]FunctionID      `json:"events"`
	Macros         map[MacroID]*MacroDef        `json:"macros"`
	MacroInstances map[string]*MacroInstanceDef `json:"macro_instances,omitempty"`
}

type FunctionConfig struct {
	Init        interface{} `yaml:"init" json:"init,omitempty"`
	Runtime     string      `yaml:"runtime" json:"runtime,omitempty"`
	Hot         bool        `yaml:"hot,omitempty" json:"hot,omitempty"`              // Boot runtime immediately and don't clean it up
	Instances   int         `yaml:"instances,omitempty"  json:"instances,omitempty"` // Number of workers to start PER NODE
	DockerImage string      `yaml:"docker_image" json:"docker_image,omitempty" mapstructure:"docker_image"`
}

type FunctionDef struct {
	Name     string          `json:"name"`
	Config   *FunctionConfig `json:"config,omitempty"`
	Language string          `json:"language,omitempty"`
	Code     string          `json:"code,omitempty"`
}

type JobConfig struct {
	Init        interface{} `yaml:"init" json:"init,omitempty"`
	Runtime     string      `yaml:"runtime" json:"runtime,omitempty"`
	Instances   int         `yaml:"instances,omitempty"  json:"instances,omitempty"` // Number of instances globally for the whole cluster
	DockerImage string      `yaml:"docker_image" json:"docker_image,omitempty" mapstructure:"docker_image"`
}

type JobDef struct {
	Name     string     `json:"name"`
	Config   *JobConfig `json:"config,omitempty"`
	Language string     `json:"language,omitempty"`
	Code     string     `json:"code,omitempty"`
}

type LibraryDef struct {
	Name     string `json:"name"`
	Runtime  string `json:"runtime"`
	Language string `json:"language,omitempty"`
	Code     string `json:"code,omitempty"`
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

func Check(path string, code string, cacheDir string) (*Definitions, error) {
	defs, err := Parse(code)
	if err != nil {
		return nil, err
	}

	if err := defs.InlineImports(path, cacheDir); err != nil {
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
