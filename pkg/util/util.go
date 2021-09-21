package util

import (
	"crypto/rand"
	"fmt"
	"gopkg.in/yaml.v3"
	"net/http"
	"regexp"
	"strings"
)

func TokenGenerator() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

var safeFilenameRE = regexp.MustCompile("[^A-Za-z0-9_\\.]")

func SafeFilename(s string) string {
	return safeFilenameRE.ReplaceAllString(s, "_")
}

func FlatStringMap(m map[string][]string) map[string]string {
	m2 := map[string]string{}
	for k, vs := range m {
		if len(vs) > 0 {
			m2[k] = vs[0]
		}
	}
	return m2
}

func StrictYamlUnmarshal(yamlString string, target interface{}) error {
	decoder := yaml.NewDecoder(strings.NewReader(yamlString))
	decoder.KnownFields(true)
	return decoder.Decode(target)
}

func YamlUnmarshal(yamlString string) (interface{}, error) {
	var target interface{}
	decoder := yaml.NewDecoder(strings.NewReader(yamlString))
	decoder.KnownFields(true)
	if err := decoder.Decode(&target); err != nil {
		return nil, err
	}
	return target, nil
}

type MultiError struct {
	Errs []error
}

func NewMultiError(errs []error) *MultiError {
	return &MultiError{errs}
}

func (me *MultiError) Error() string {
	errs := make([]string, len(me.Errs))
	for i, err := range me.Errs {
		errs[i] = err.Error()
	}
	return strings.Join(errs, "\n")
}

func HTTPWriteJSONError(w http.ResponseWriter, httpStatus int, message string, additionalData interface{}) {
	type jsonError struct {
		Error string      `json:"error"`
		Data  interface{} `json:"data,omitempty"`
	}
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(httpStatus)
	w.Write(MustJsonByteSlice(jsonError{message, additionalData}))
}
