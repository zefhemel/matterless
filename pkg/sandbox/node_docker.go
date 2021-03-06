package sandbox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/util"
	"io"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

type NodeDockerSandbox struct {
}

func (s *NodeDockerSandbox) wrapScript(event interface{}, code string) string {
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

func (s *NodeDockerSandbox) determineNodeBin() string {
	if nodeBin := os.Getenv("nodeBin"); nodeBin != "" {
		return nodeBin
	} else {
		return "node"
	}
}

func (s *NodeDockerSandbox) Invoke(event interface{}, code string, env map[string]string) (interface{}, string, error) {
	args := make([]string, 0, 10)
	args = append(args, "run", "--rm")
	for envKey, envVal := range env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", envKey, envVal))
	}
	args = append(args, "-i", "zefhemel/matterless-runner-docker-node", "--input-type=module", "-e", s.wrapScript(event, code))
	cmd := exec.Command("docker", args...)
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

func NewNodeDockerSandbox() *NodeDockerSandbox {
	return &NodeDockerSandbox{}
}

var _ Sandbox = &NodeDockerSandbox{}
