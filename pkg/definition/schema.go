package definition

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/zefhemel/matterless/pkg/util"
	"strings"
)

// Subset of JSON schema
type TypeSchema struct {
	Type        string `yaml:"type"` // string | number | boolean | object | array
	Description string `yaml:"description,omitempty"`

	// type object
	Properties           map[string]*TypeSchema `yaml:"properties,omitempty"`           // for objects
	AdditionalProperties *TypeSchema            `yaml:"additionalProperties,omitempty"` // Allow additional properties
	Required             []string               `yaml:"required,omitempty"`             // Required properties

	// type array
	Items *TypeSchema `yaml:"items,omitempty"` // for array
}

func NewSchema(yamlSource string) (*TypeSchema, error) {
	var typeSchema TypeSchema
	if err := util.StrictYamlUnmarshal(yamlSource, &typeSchema); err != nil {
		return nil, err
	}
	return &typeSchema, nil
}

func MustNewSchema(yamlSource string) *TypeSchema {
	ts, err := NewSchema(yamlSource)
	if err != nil {
		panic(err)
	}
	return ts
}

type MultiError struct {
	errs []error
}

func (me *MultiError) Error() string {
	errs := make([]string, len(me.errs))
	for i, err := range me.errs {
		errs[i] = err.Error()
	}
	return strings.Join(errs, "\n")
}

func (ts *TypeSchema) ValidateString(yamlString string) error {
	d, err := util.YamlUnmarshal(yamlString)
	if err != nil {
		return errors.Wrap(err, "yaml unmarshal")
	}
	return ts.Validate(d)
}

func (ts *TypeSchema) Validate(val interface{}) error {
	switch ts.Type {
	case "string":
		if _, ok := val.(string); !ok {
			return fmt.Errorf("should be a string value: %s", val)
		}
	case "number":
		switch val.(type) {
		case float64, float32, int32, uint32, int64, uint64, int, uint:
			return nil
		default:
			return fmt.Errorf("should be a number: %s", val)
		}
	case "bool", "boolean":
		if _, ok := val.(bool); !ok {
			return fmt.Errorf("should be a boolean value: %s", val)
		}
	case "array":
		errs := []error{}
		l, ok := val.([]interface{})
		if !ok {
			return fmt.Errorf("should be an array value: %s", val)
		}
		for _, item := range l {
			if err := ts.Items.Validate(item); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return &MultiError{errs}
		}
	case "object":
		errs := []error{}
		m, ok := val.(map[string]interface{})
		if !ok {
			return fmt.Errorf("should be an object value: %s (%T)", val, val)
		}
		for k, v := range m {
			prop, ok := ts.Properties[k]
			if !ok {
				if ts.AdditionalProperties != nil {
					if err := ts.AdditionalProperties.Validate(v); err != nil {
						errs = append(errs, err)
					}
				} else {
					errs = append(errs, fmt.Errorf("undefined property: %s", k))
				}
			} else {
				if err := prop.Validate(v); err != nil {
					errs = append(errs, err)
				}
			}
		}
		if len(errs) > 0 {
			return &MultiError{errs}
		}
	}
	return nil
}
