package application

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/store"
	"gopkg.in/yaml.v3"
	"reflect"
	"regexp"
	"strings"
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
func (app *Application) interpolateStoreValues() {
	defs := app.definitions

	logCallback := func(message string) {
		log.Errorf("[%s] %s", app.appName, message)
	}

	for _, def := range defs.Jobs {
		yamlBuf, _ := yaml.Marshal(def.Config.Init)
		interPolatedYaml := interpolateStoreValues(app.dataStore, string(yamlBuf), logCallback)
		var val interface{}
		yaml.Unmarshal([]byte(interPolatedYaml), &val)
		def.Config.Init = val
	}
	for _, def := range defs.Functions {
		yamlBuf, _ := yaml.Marshal(def.Config.Init)
		interPolatedYaml := interpolateStoreValues(app.dataStore, string(yamlBuf), logCallback)
		var val interface{}
		yaml.Unmarshal([]byte(interPolatedYaml), &val)
		def.Config.Init = val
	}

	for _, def := range defs.MacroInstances {
		yamlBuf, _ := yaml.Marshal(def.Arguments)
		interPolatedYaml := interpolateStoreValues(app.dataStore, string(yamlBuf), logCallback)
		var val interface{}
		yaml.Unmarshal([]byte(interPolatedYaml), &val)
		def.Arguments = val
	}
}
