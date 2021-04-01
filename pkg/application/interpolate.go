package application

import (
	"fmt"
	"github.com/zefhemel/matterless/pkg/store"
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
func interPolateStoreValues(store store.Store, s string, logCallback func(string)) string {
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
