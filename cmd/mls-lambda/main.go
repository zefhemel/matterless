package main

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/util"
	"io"
	"os"
	"os/exec"
	"strings"
)

type JSONObject = interface{}

func run(runnerType string, code string, env map[string]string) (inputChannel chan JSONObject, resultChannel chan JSONObject, logChannel chan JSONObject, err error) {
	if err := os.WriteFile("function.mjs", []byte(code), 0600); err != nil {
		return nil, nil, nil, err
	}
	cmd := exec.Command("node", "function.mjs")
	cmd.Env = make([]string, 0, 10)
	for envKey, envVal := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envKey, envVal))
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "stdin pipe")
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "stdout pipe")
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "stderr pipe")
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, nil, errors.Wrap(err, "start")
	}

	inputChannel = make(chan JSONObject)
	resultChannel = make(chan JSONObject)
	logChannel = make(chan JSONObject)

	stdoutDecoder := json.NewDecoder(stdout)
	stdinEncoder := json.NewEncoder(stdin)

	go func() {
		for inputEvent := range inputEvents {
			if err := stdinEncoder.Encode(inputEvent); err != nil {
				log.Error("Could not encode object", err)
				continue
			}
			var result JSONObject
			if err := stdoutDecoder.Decode(&result); err != nil {
				log.Error("Could not encode from stream", err)
				continue
			}
			resultChannel <- result
		}
	}()
	err = nil
	return
}

func main() {

}
