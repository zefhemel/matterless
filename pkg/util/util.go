package util

import (
	"crypto/rand"
	"fmt"
	"gopkg.in/yaml.v3"
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
