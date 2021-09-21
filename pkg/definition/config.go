package definition

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/store"
	"github.com/zefhemel/matterless/pkg/util"
)

func (defs *Definitions) CheckConfig(store store.Store) map[string]string {
	resultMap := map[string]string{}

	fullSchema, _ := NewSchema(`type: object`)
	for key, def := range defs.Config {
		fullSchema.Properties[key] = def
	}

	configObj := map[string]interface{}{}
	for key, _ := range defs.Config {
		val, _ := store.Get(key)
		if val == nil {
			resultMap[key] = fmt.Sprintf("missing, expected type: %s", fullSchema.Properties[key].Type)
		} else {
			configObj[key] = val
		}
	}

	err := fullSchema.Validate(configObj)
	if err == nil {
		return resultMap
	}
	if multiError, ok := err.(*util.MultiError); ok {
		for _, err := range multiError.Errs {
			if propertyErr, ok := err.(*PropertyError); ok {
				resultMap[propertyErr.Property] = propertyErr.Err.Error()
			}
		}
	} else {
		log.Fatalf("%s is not a multi error %T", err, err)
	}
	return resultMap
}
