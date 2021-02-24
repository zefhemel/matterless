package sandbox

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/zefhemel/matterless/pkg/util"
)

type NodeSandbox struct {
	nodeBin string
}

func (s *NodeSandbox) wrapScript(event interface{}, code string) string {
	return fmt.Sprintf(`%s

function getClient() {
	require('isomorphic-fetch');
	const Client4 = require('./mattermost-redux/mattermost.client4.js').default;
	const client = new Client4();
	client.setUrl(process.env.URL);
	client.setToken(process.env.TOKEN);
	return client;
}

let result = handler(%s);
if(result) {
	console.error(JSON.stringify(result));
} else {
	console.error(JSON.stringify({}));
}
`, code, util.MustJsonString(event))
}

func (s *NodeSandbox) Invoke(event interface{}, code string, env map[string]string) (interface{}, []string, error) {
	cmd := exec.Command(s.nodeBin, "-e", s.wrapScript(event, code))
	cmd.Env = make([]string, 0, 10)
	for envKey, envVal := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envKey, envVal))
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, errors.Wrap(err, "stdout pipe")
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, errors.Wrap(err, "stderr pipe")
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, errors.Wrap(err, "start")
	}

	stdOutBuf, err := io.ReadAll(stdout)
	if err != nil {
		return nil, nil, err
	}
	stdErrBuf, err := io.ReadAll(stderr)
	if err != nil {
		return nil, nil, err
	}

	if err := cmd.Wait(); err != nil {
		return nil, nil, fmt.Errorf("Failed to run:\n%s", stdErrBuf)
	}
	var response interface{}
	err = json.Unmarshal(stdErrBuf, &response)
	if err != nil {
		return nil, nil, errors.Wrap(err, fmt.Sprintf("json response decode: %s", stdErrBuf))
	}
	logs := strings.Split(string(stdOutBuf), "\n")
	return response, logs, nil
}

func NewNodeSandbox(nodeBin string) *NodeSandbox {
	return &NodeSandbox{
		nodeBin: nodeBin,
	}
}

var _ Sandbox = &NodeSandbox{}
