package definition_test

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"strings"
	"testing"

	_ "embed"

	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/definition"
)

func TestParser(t *testing.T) {
	decls, err := definition.Parse(strings.ReplaceAll(`
# function TestFunction1

|||
function handle(event) {
	console.log("Hello world!");
}
|||

# function TestFunction2

|||javascript
function handle(event) {
	console.log("Hello world 2!");
}
|||
`, "|||", "```"))
	assert.NoError(t, err)
	assert.Equal(t, "TestFunction1", decls.Functions["TestFunction1"].Name)
	assert.Equal(t, "TestFunction2", decls.Functions["TestFunction2"].Name)
	assert.Equal(t, "javascript", decls.Functions["TestFunction2"].Language)
}

func TestFunctionParserParameterBlock(t *testing.T) {
	defs, err := definition.Parse(strings.ReplaceAll(`# function MyFunc
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
	assert.Equal(t, "Zef", defs.Functions["MyFunc"].Config.Init.(map[string]interface{})["arg1"])
	assert.Equal(t, "bla/bla", defs.Functions["MyFunc"].Config.DockerImage)
	assert.Contains(t, defs.Functions["MyFunc"].Code, "handle")
}

func TestFunctionParserParameterBlockFail(t *testing.T) {
	_, err := definition.Parse(strings.ReplaceAll(`# function MyFunc
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
	defs, err := definition.Parse(strings.ReplaceAll(`# function MyFunc
|||javascript
function handle() {

}
|||
`, "|||", "```"))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(defs.Functions))
	log.Infof("%+v", defs.Functions["MyFunc"])
	assert.Contains(t, defs.Functions["MyFunc"].Code, "handle")
}

func TestJobParserParameterBlock(t *testing.T) {
	defs, err := definition.Parse(strings.ReplaceAll(`# job MyJob
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
	assert.Equal(t, "Zef", defs.Jobs["MyJob"].Config.Init.(map[string]interface{})["arg1"])
	assert.Equal(t, "bla/bla", defs.Jobs["MyJob"].Config.DockerImage)
	assert.Contains(t, defs.Jobs["MyJob"].Code, "handle")
}

func TestJobParser(t *testing.T) {
	defs, err := definition.Parse(strings.ReplaceAll(`# job MyJob
|||javascript
function run() {

}
|||
`, "|||", "```"))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(defs.Jobs))
	log.Infof("%+v", defs.Jobs["MyJob"])
	assert.Contains(t, defs.Jobs["MyJob"].Code, "run()")
}

func TestTemplateParser(t *testing.T) {
	defs, err := definition.Parse(strings.ReplaceAll(`# macro helloJobNoSchema

	# job {{$name}}

    |||
    init:
       name: {{$arg.name_arg}}
    |||

    |||
    function init(config) {
       setInterval(() => {
           console.log("Hello, ", config.name);
       }, 1000);
    }
    |||

# helloJobNoSchema TheJob
|||
name_arg: Zef
|||
`, "|||", "```"))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(defs.Macros))
	assert.Contains(t, defs.Macros["helloJobNoSchema"].TemplateCode, "job")

	assert.Equal(t, 1, len(defs.MacroInstances))
	assert.Equal(t, definition.MacroID("helloJobNoSchema"), defs.MacroInstances["TheJob"].Macro)
	assert.Equal(t, "Zef", defs.MacroInstances["TheJob"].Arguments.(map[string]interface{})["name_arg"])

	assert.NoError(t, defs.ExpandMacros())

	defs, err = definition.Parse(strings.ReplaceAll(`# macro helloJob
|||
schema:
   type: object
   properties:
      name:
         type: string
|||

	# job {{$name}}

    |||
    init:
       name: {{$arg.name}}
    |||

    |||
    function init(config) {
       setInterval(() => {
           console.log("Hello, ", config.name);
       }, 1000);
    }
    |||

# helloJob TheJob
|||
name: Zef
|||
`, "|||", "```"))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(defs.Macros))
	assert.Contains(t, defs.Macros["helloJob"].TemplateCode, "job")

	assert.Equal(t, 1, len(defs.MacroInstances))
	assert.Equal(t, definition.MacroID("helloJob"), defs.MacroInstances["TheJob"].Macro)
	assert.Equal(t, "Zef", defs.MacroInstances["TheJob"].Arguments.(map[string]interface{})["name"])

	assert.NoError(t, defs.ExpandMacros())
}

func TestTemplateParserNonExisting(t *testing.T) {
	defs, err := definition.Parse(strings.ReplaceAll(`
# helloJob TheJob
|||
name: Zef
|||
`, "|||", "```"))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(defs.MacroInstances))
	assert.Error(t, defs.ExpandMacros())
}

func TestImportParsing(t *testing.T) {
	defs, err := definition.Parse(strings.ReplaceAll(`
# import
* http://bla.com

# import
- http://bla2.com

`, "|||", "```"))
	assert.NoError(t, err)
	assert.Len(t, defs.Imports, 2)
	assert.Equal(t, "http://bla.com", defs.Imports[0])
	assert.Equal(t, "http://bla2.com", defs.Imports[1])
}

func TestMergeJobs(t *testing.T) {
	defs1, err := definition.Parse(strings.ReplaceAll(`
# job MyCron
|||
init:
- schedule: "* * * * * *"
  function: MyFunc
|||

|||javascript
function handle() {

}
|||
`, "|||", "```"))
	assert.NoError(t, err)

	defs2, err := definition.Parse(strings.ReplaceAll(`
# job MyCron
|||
init:
- schedule: "*/2 * * * * *"
  function: MyFunc2
|||

|||javascript
function handle() {

}
|||
`, "|||", "```"))

	assert.NoError(t, defs1.MergeFrom(defs2))
	assert.Equal(t, 2, len(defs1.Jobs["MyCron"].Config.Init.([]interface{})))
	//fmt.Println(defs1.Markdown())

	// validate with nested slice
	defs1, err = definition.Parse(strings.ReplaceAll(`
# job MyCron
|||
init:
   bla1:
     schedule: "* * * * * *"
     function: MyFunc
|||

|||javascript
function handle() {

}
|||
`, "|||", "```"))
	assert.NoError(t, err)

	defs2, err = definition.Parse(strings.ReplaceAll(`
# job MyCron
|||
init:
   bla2:
     schedule: "*/2 * * * * *"
     function: MyFunc2
|||

|||javascript
function handle() {

}
|||
`, "|||", "```"))

	assert.NoError(t, defs1.MergeFrom(defs2))
	//assert.Equal(t, 2, len(defs1.Jobs["MyCron"].Config.Init.(map[string]interface{})["bla"].([]interface{})))
	fmt.Println(defs1.Markdown())

}
