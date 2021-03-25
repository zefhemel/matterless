package main

import (
	"fmt"
	"github.com/zefhemel/matterless/pkg/util"
	"os"
)

type packageJSON struct {
	Name string `json:"name"`
	Main string `json:"main"`
}

func createModule(name string, code string) error {
	if err := os.MkdirAll(fmt.Sprintf("node_modules/%s", name), 0600); err != nil {
		return err
	}
	if err := os.WriteFile(fmt.Sprintf("node_modules/%s/package.json", name), []byte(util.MustJsonString(packageJSON{
		Name: name,
		Main: "main.mjs",
	})), 0600); err != nil {
		return err
	}
	if err := os.WriteFile(fmt.Sprintf("node_modules/%s/main.mjs", name), []byte(code), 0600); err != nil {
		return err
	}
	return nil
}
