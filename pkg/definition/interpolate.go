package definition

import (
	"encoding/json"
	"fmt"
	"github.com/zefhemel/matterless/pkg/util"
	"reflect"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/store"
)

var interPolationRegexp = regexp.MustCompile(`\$\{[^\}]+\}`)

func isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		return reflect.ValueOf(i).IsNil()
	}
	return false
}
func interpolateStoreValues(store store.Store, s string, logCallback func(string)) string {
	return interPolationRegexp.ReplaceAllStringFunc(s, func(s string) string {
		key := strings.TrimSuffix(strings.TrimPrefix(s, "${"), "}")
		result, err := store.Get(key)
		if err != nil {
			logCallback(fmt.Sprintf("Interpolation store lookup fail for key '%s': %s", key, err))
			return ""
		}
		if isNil(result) {
			logCallback(fmt.Sprintf("String interpolation: no store value for '%s'", key))
			return ""
		}
		return fmt.Sprintf("%s", result)
	})
}

// Normalize replaces environment variables with their values
func (defs *Definitions) InterpolateStoreValues(store store.Store) {
	logCallback := func(message string) {
		log.Error(message)
	}

	for _, def := range defs.Jobs {
		interPolatedJSON := interpolateStoreValues(store, util.MustJsonString(def.Config.Init), logCallback)
		var val interface{}
		json.Unmarshal([]byte(interPolatedJSON), &val)
		def.Config.Init = val
	}
	for _, def := range defs.Functions {
		interPolatedJSON := interpolateStoreValues(store, util.MustJsonString(def.Config.Init), logCallback)
		var val interface{}
		json.Unmarshal([]byte(interPolatedJSON), &val)
		def.Config.Init = val
	}

	for _, def := range defs.MacroInstances {
		interPolatedJSON := interpolateStoreValues(store, util.MustJsonString(def.Arguments), logCallback)
		var val interface{}
		json.Unmarshal([]byte(interPolatedJSON), &val)
		def.Arguments = val
	}
}
