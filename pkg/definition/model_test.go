package definition_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/definition"
	"strings"
	"testing"
)

func TestMarkdown(t *testing.T) {
	defs, err := definition.Parse(strings.ReplaceAll(`
# Config
|||
name: bla
age: 10
|||

# Function: MyFunc
|||javascript
function handle() {
}
|||

# Function: MyFunc2
|||javascript
function handle() {
}
|||

# Macro: MyMacro
|||
schema:
  type: object
|||

	# Function: MyFunction
	|||
	function bla() {
	}
	|||
`, "|||", "```"))
	assert.NoError(t, err)
	fmt.Println(defs.Markdown())
}
