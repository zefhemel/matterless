package sandbox

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/util"
	"io"
	"os/exec"
	"strings"
)

type NodeDockerSandbox struct {
	runningInstances map[string]*instance
}

type instance struct {
	cmd          *exec.Cmd
	stdinPipe    io.WriteCloser
	logChannel   chan string
	resultReader *bufio.Reader
}

func (inst *instance) kill() error {
	inst.stdinPipe.Close()
	if err := inst.cmd.Wait(); err != nil {
		return fmt.Errorf("Failed to run: %s", err)
	}
	log.Info("Killed instance")
	return nil
}

var pingEvent struct{} = struct{}{}

func (s *NodeDockerSandbox) boot(code string, env map[string]string) (*instance, error) {
	inst, ok := s.runningInstances[code]
	if !ok {
		var err error
		inst, err = newInstance(code, env)
		if err != nil {
			return nil, err
		}
		if _, _, err := inst.invoke(pingEvent); err != nil {
			log.Info("Failed sending ping event", err)
			return nil, err
		}

		s.runningInstances[code] = inst
	}
	return inst, nil
}

func newInstance(code string, env map[string]string) (*instance, error) {
	var err error
	inst := &instance{
		logChannel: make(chan string),
	}
	args := make([]string, 0, 10)
	args = append(args, "run", "--rm")
	for envKey, envVal := range env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", envKey, envVal))
	}
	args = append(args, "-i", "zefhemel/matterless-runner-docker-node", code)
	inst.cmd = exec.Command("docker", args...)
	stdoutPipe, err := inst.cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "stdout pipe")
	}
	stderrPipe, err := inst.cmd.StderrPipe()
	if err != nil {
		return nil, errors.Wrap(err, "stderr pipe")
	}
	inst.resultReader = bufio.NewReader(stderrPipe)

	inst.stdinPipe, err = inst.cmd.StdinPipe()
	if err != nil {
		return nil, errors.Wrap(err, "stdin pipe")
	}

	if err := inst.cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "start")
	}

	go func() {
		stdoutReader := bufio.NewReader(stdoutPipe)
		for {
			line, err := stdoutReader.ReadString('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Error("Stdout error", err)
				break
			}
			inst.logChannel <- line
		}
		close(inst.logChannel)
	}()

	return inst, nil
}

func (s *NodeDockerSandbox) Invoke(event interface{}, code string, env map[string]string) (interface{}, string, error) {
	inst, err := s.boot(code, env)
	if err != nil {
		return nil, "", err
	}

	//log.Info("Got back from boot", inst.cmd.ProcessState)

	response, logMesssages, err := inst.invoke(event)
	if err != nil {
		return nil, "", err
	}
	return response, strings.TrimSpace(strings.Join(logMesssages, "\n")), nil
}

func (s *NodeDockerSandbox) Cleanup() {
	for _, inst := range s.runningInstances {
		log.Info("Killing an instance now...")
		if err := inst.kill(); err != nil {
			log.Error("Error killing instance", err)
		}
	}
	s.runningInstances = map[string]*instance{}
}

const logRunDivider = "!!EOL!!\n"

func (inst *instance) invoke(event interface{}) (interface{}, []string, error) {
	fmt.Fprintf(inst.stdinPipe, "%s\n", util.MustJsonString(event))

	logMessages := make([]string, 0, 10)
	endOfLogChan := make(chan struct{})
	go func() {
		for logMessage := range inst.logChannel {
			if logMessage == logRunDivider {
				endOfLogChan <- struct{}{}
				break
			}
			logMessages = append(logMessages, logMessage)
		}
	}()
	resp, err := inst.resultReader.ReadString('\n')
	if err != nil {
		return nil, nil, err
	}
	var result interface{}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		// Not valid JSON may be plain text, let's get all of it
		buf, _ := io.ReadAll(inst.resultReader)
		firstLineBuf := []byte(resp)
		firstLineBuf = append(firstLineBuf, buf...)
		return nil, nil, fmt.Errorf("From process: %s", firstLineBuf)
	}
	// Wait for all log messages to flush
	<-endOfLogChan

	return result, logMessages, nil
}

func NewNodeDockerSandbox() *NodeDockerSandbox {
	return &NodeDockerSandbox{
		runningInstances: map[string]*instance{},
	}
}

var _ Sandbox = &NodeDockerSandbox{}
