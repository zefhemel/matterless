package definition_test

import (
	log "github.com/sirupsen/logrus"
	"strings"
	"testing"

	_ "embed"

	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/definition"
)

//go:embed test/test1.md
var test1Md string

func TestValidation(t *testing.T) {
	err := definition.Validate("schema/config.schema.json", `
url: http://localhost
token: abc
`)
	assert.NoError(t, err)
}

func TestParser(t *testing.T) {
	decls, err := definition.Parse(test1Md)
	assert.NoError(t, err)
	assert.Equal(t, "TestFunction1", decls.Functions["TestFunction1"].Name)
	assert.Equal(t, "TestFunction2", decls.Functions["TestFunction2"].Name)
	assert.Equal(t, "javascript", decls.Functions["TestFunction2"].Language)
}

func TestFunctionParserParameterBlock(t *testing.T) {
	defs, err := definition.Parse(strings.ReplaceAll(`# Function: MyFunc
|||
init:
  arg1: Zef
  list:
  - a
  - b
  number: 10
docker_image: bla/bla
|||

|||javascript
function handle() {

}
|||
`, "|||", "```"))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(defs.Functions))
	assert.Equal(t, "Zef", defs.Functions["MyFunc"].Config.Init["arg1"])
	assert.Equal(t, "bla/bla", defs.Functions["MyFunc"].Config.DockerImage)
	assert.Contains(t, defs.Functions["MyFunc"].Code, "handle")
}

func TestFunctionParserParameterBlockFail(t *testing.T) {
	_, err := definition.Parse(strings.ReplaceAll(`# Function: MyFunc
|||
randomStuff
|||

|||javascript
function handle() {

}
|||
`, "|||", "```"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestFunctionParser(t *testing.T) {
	defs, err := definition.Parse(strings.ReplaceAll(`# Function: MyFunc
|||javascript
function handle() {

}
|||
`, "|||", "```"))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(defs.Functions))
	log.Infof("%+v", defs.Functions["MyFunc"])
	assert.Equal(t, 0, len(defs.Functions["MyFunc"].Config.Init))
	assert.Contains(t, defs.Functions["MyFunc"].Code, "handle")
}

func TestJobParserParameterBlock(t *testing.T) {
	defs, err := definition.Parse(strings.ReplaceAll(`# Job: MyJob
|||
init:
  arg1: Zef
  list:
  - a
  - b
  number: 10
docker_image: bla/bla
|||

|||javascript
function handle() {

}
|||
`, "|||", "```"))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(defs.Jobs))
	assert.Equal(t, "Zef", defs.Jobs["MyJob"].Config.Init["arg1"])
	assert.Equal(t, "bla/bla", defs.Jobs["MyJob"].Config.DockerImage)
	assert.Contains(t, defs.Jobs["MyJob"].Code, "handle")
}

func TestJobParser(t *testing.T) {
	defs, err := definition.Parse(strings.ReplaceAll(`# Job: MyJob
|||javascript
function run() {

}
|||
`, "|||", "```"))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(defs.Jobs))
	log.Infof("%+v", defs.Jobs["MyJob"])
	assert.Equal(t, 0, len(defs.Jobs["MyJob"].Config.Init))
	assert.Contains(t, defs.Jobs["MyJob"].Code, "run()")
}

func TestTemplateParser(t *testing.T) {
	defs, err := definition.Parse(strings.ReplaceAll(`# Macro: HelloJob
|||
input_schema:
   type: object
   properties:
      name:
         type: string
|||

	# Job: {{$name}}

    |||
    init:
       name: {{$input.name}}
    |||

    |||
    function init(config) {
       setInterval(() => {
           console.log("Hello, ", config.name);
       }, 1000);
    }
    |||

# HelloJob: TheJob
|||
name: Zef
|||
`, "|||", "```"))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(defs.Macros))
	assert.Contains(t, defs.Macros["HelloJob"].TemplateCode, "Job")

	assert.Equal(t, 1, len(defs.CustomDef))
	assert.Equal(t, definition.MacroID("HelloJob"), defs.CustomDef["TheJob"].Macro)
	assert.Equal(t, "Zef", defs.CustomDef["TheJob"].Input.(map[string]interface{})["name"])

	assert.NoError(t, defs.Desugar())
}

func TestTemplateParserNonExisting(t *testing.T) {
	defs, err := definition.Parse(strings.ReplaceAll(`
# HelloJob: TheJob
|||
name: Zef
|||
`, "|||", "```"))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(defs.CustomDef))
	assert.Error(t, defs.Desugar())
}

func TestImportParsing(t *testing.T) {
	defs, err := definition.Parse(strings.ReplaceAll(`
# Import
* http://bla.com

# Import
- http://bla2.com

`, "|||", "```"))
	assert.NoError(t, err)
	assert.Len(t, defs.Imports, 2)
	assert.Equal(t, "http://bla.com", defs.Imports[0])
	assert.Equal(t, "http://bla2.com", defs.Imports[1])
}
