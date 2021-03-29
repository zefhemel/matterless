package definition

import (
	"github.com/buildkite/interpolate"
	"gopkg.in/yaml.v3"
)

// Normalize replaces environment variables with their values
func (decls *Definitions) Normalize() {
	mapEnv := interpolate.NewMapEnv(decls.Environment)
	for k, v := range decls.Environment {
		if newV, err := interpolate.Interpolate(mapEnv, v); err == nil {
			decls.Environment[k] = newV
		}
	}

	for _, def := range decls.Jobs {
		for k, v := range def.Config.Config {
			yamlBuf, _ := yaml.Marshal(v)
			if interPolatedYaml, err := interpolate.Interpolate(mapEnv, string(yamlBuf)); err == nil {
				var val interface{}
				yaml.Unmarshal([]byte(interPolatedYaml), &val)
				def.Config.Config[k] = val
			}
		}
	}
	for _, def := range decls.Functions {
		for k, v := range def.Config.Config {
			yamlBuf, _ := yaml.Marshal(v)
			if interPolatedYaml, err := interpolate.Interpolate(mapEnv, string(yamlBuf)); err == nil {
				var val interface{}
				yaml.Unmarshal([]byte(interPolatedYaml), &val)
				def.Config.Config[k] = val
			}
		}
	}
}
