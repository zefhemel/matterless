package definition_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/definition"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestSchema(t *testing.T) {
	checkSchemaAgainstValue(t, `type: string`, "hello")
	checkSchemaAgainstWrongValue(t, `type: string`, "10", "should be a string")
	checkSchemaAgainstValue(t, `type: number`, `20`)
	checkSchemaAgainstWrongValue(t, `type: number`, "pete", "should be a number")
	checkSchemaAgainstValue(t, `type: bool`, `true`)
	checkSchemaAgainstWrongValue(t, `type: bool`, "1", "should be a boolean")
	checkSchemaAgainstValue(t, `
type: array
items:
  type: string
`, `
- hello
- "10"
`)

	checkSchemaAgainstValue(t, `
type: object
properties:
  name:
    type: string
  age:
    type: number
`, `
name: Hank
age: 20
`)

	checkSchemaAgainstWrongValue(t, `
type: object
properties:
  name:
    type: string
  age:
    type: number
`, `
name: Hank
age: 20
additional: hello
`, "undefined property")

	complexSchema := `
type: object
properties:
  name:
    type: string
  friends:
    type: array
    items:
       type: string
`
	checkSchemaAgainstValue(t, complexSchema, `
name: Hank
friends:
- Pete
- Robert
`)

	checkSchemaAgainstWrongValue(t, complexSchema, `
name: Hank
friends: Pete
`, "should be an array")

	eventSchema := `
type: object
additionalProperties:
  type: array
  items:
    type: string
`
	checkSchemaAgainstValue(t, eventSchema, `
myevent:
- MyFunction
- MySecondFunction
other-event:
- Bla
`)

	checkSchemaAgainstWrongValue(t, eventSchema, `
myevent:
- 10
`, "should be a string")
}

func checkSchemaAgainstValue(t *testing.T, schemaYaml, valueYaml string) {
	ts, err := definition.NewSchema(schemaYaml)
	assert.NoError(t, err)
	var val interface{}
	assert.NoError(t, yaml.Unmarshal([]byte(valueYaml), &val))
	assert.NoError(t, ts.Validate(val))
}

func checkSchemaAgainstWrongValue(t *testing.T, schemaYaml, valueYaml string, errorContains string) {
	ts, err := definition.NewSchema(schemaYaml)
	assert.NoError(t, err)
	var val interface{}
	assert.NoError(t, yaml.Unmarshal([]byte(valueYaml), &val))
	assert.Contains(t, ts.Validate(val).Error(), errorContains)
}
