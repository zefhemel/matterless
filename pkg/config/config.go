package config

import (
	"os"
	"strings"
)

type Config struct {
	APIBindPort int
	DataDir     string
	AdminToken  string
	GlobalEnv   map[string]string
	PersistApps bool // Whether to write deployed app code to disk
}

func FromEnv() *Config {
	globalEnv := map[string]string{}
	for _, envCombo := range os.Environ() {
		parts := strings.Split(envCombo, "=")
		if strings.HasPrefix(parts[0], "MLS_") {
			globalEnv[parts[0]] = parts[1]
		}
	}
	return &Config{
		GlobalEnv: globalEnv,
	}
}
