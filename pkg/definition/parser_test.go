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
	decls.Normalize()
	assert.NoError(t, err)
	assert.Equal(t, "TestFunction1", decls.Functions["TestFunction1"].Name)
	assert.Equal(t, "TestFunction2", decls.Functions["TestFunction2"].Name)
	assert.Equal(t, "javascript", decls.Functions["TestFunction2"].Language)
	assert.Equal(t, "http://localhost:8065", decls.Config["MattermostURL"])
	assert.Equal(t, "1234", decls.Config["MattermostToken"])
	assert.Equal(t, "javascript", decls.Modules["my-module"].Language)
}

func TestFunctionParserParameterBlock(t *testing.T) {
	defs, err := definition.Parse(strings.ReplaceAll(`# Function: MyFunc
|||
config:
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
	assert.Equal(t, "Zef", defs.Functions["MyFunc"].Config.Config["arg1"])
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
	assert.Equal(t, 0, len(defs.Functions["MyFunc"].Config.Config))
	assert.Contains(t, defs.Functions["MyFunc"].Code, "handle")
}

func TestJobParserParameterBlock(t *testing.T) {
	defs, err := definition.Parse(strings.ReplaceAll(`# Job: MyJob
|||
config:
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
	assert.Equal(t, "Zef", defs.Jobs["MyJob"].Config.Config["arg1"])
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
	assert.Equal(t, 0, len(defs.Jobs["MyJob"].Config.Config))
	assert.Contains(t, defs.Jobs["MyJob"].Code, "run()")
}

func TestTemplateParser(t *testing.T) {
	defs, err := definition.Parse(strings.ReplaceAll(`# Template: HelloJob
|||
schema:
   type: object
   properties:
      name:
         type: string
|||

	# Job: {{$name}}

    |||
    config:
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
	assert.Equal(t, 1, len(defs.Template))
	assert.Contains(t, defs.Template["HelloJob"].Template, "Job")

	assert.Equal(t, 1, len(defs.CustomDef))
	assert.Equal(t, "HelloJob", defs.CustomDef["TheJob"].Template)
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
