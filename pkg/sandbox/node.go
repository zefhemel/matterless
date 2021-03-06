package sandbox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"text/template"

	_ "embed"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/util"
)

type NodeSandbox struct {
}

//go:embed node/template.js
var nodeTemplate string

func (s *NodeSandbox) wrapScript(event interface{}, code string) string {
	data := struct {
		Code  string
		Event string
	}{
		Code:  code,
		Event: util.MustJsonString(event),
	}
	tmpl, err := template.New("sourceTemplate").Parse(nodeTemplate)
	if err != nil {
		log.Error("Could not render javascript:", err)
		return ""
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		log.Error("Could not render javascript:", err)
		return ""
	}
	return out.String()
}

func (s *NodeSandbox) determineNodeBin() string {
	if nodeBin := os.Getenv("nodeBin"); nodeBin != "" {
		return nodeBin
	} else {
		return "node"
	}
}

func (s *NodeSandbox) Invoke(event interface{}, code string, env map[string]string) (interface{}, string, error) {
	tmpDir, err := os.MkdirTemp(".", "function-run")
	if err != nil {
		return nil, "", err
	}
	err = os.WriteFile(fmt.Sprintf("%s/function.mjs", tmpDir), []byte(code), 0600)
	if err != nil {
		return nil, "", err
	}

	defer func() {
		// Temp dir cleanup
		if err := os.RemoveAll(tmpDir); err != nil {
			log.Info("Could not cleanup temporarily directory ", tmpDir)
		}
	}()

	cmd := exec.Command(s.determineNodeBin(), "--input-type=module", "-e", s.wrapScript(event, code))
	cmd.Dir = tmpDir
	cmd.Env = make([]string, 0, 10)
	for envKey, envVal := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envKey, envVal))
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, "", errors.Wrap(err, "stdout pipe")
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, "", errors.Wrap(err, "stderr pipe")
	}

	if err := cmd.Start(); err != nil {
		return nil, "", errors.Wrap(err, "start")
	}

	stdOutBuf, err := io.ReadAll(stdout)
	if err != nil {
		return nil, "", err
	}
	stdErrBuf, err := io.ReadAll(stderr)
	if err != nil {
		return nil, "", err
	}

	if err := cmd.Wait(); err != nil {
		return nil, "", fmt.Errorf("Failed to run:\n%s", stdErrBuf)
	}
	var response interface{}
	err = json.Unmarshal(stdErrBuf, &response)
	if err != nil {
		return nil, "", errors.Wrap(err, fmt.Sprintf("json response decode: %s", stdErrBuf))
	}
	return response, strings.TrimSpace(string(stdOutBuf)), nil
}

func NewNodeSandbox() *NodeSandbox {
	return &NodeSandbox{}
}

var _ Sandbox = &NodeSandbox{}
