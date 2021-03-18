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
	"sync"
	"time"
)

var pingEvent = struct{}{}

const logRunDivider = "!!EOL!!\n"

// DockerSandbox implements a simple sandbox using docker
type DockerSandbox struct {
	runningInstances map[string]*instance
	cleanupInterval  time.Duration
	keepAlive        time.Duration
	ticker           *time.Ticker
	stop             chan struct{}
}

type instance struct {
	cmd          *exec.Cmd
	stdinPipe    io.WriteCloser
	logChannel   chan string
	resultReader *bufio.Reader
	lastInvoked  time.Time
	runLock      sync.Mutex
}

func (inst *instance) kill() error {
	// Don't kill until current run is over, if any
	inst.runLock.Lock()
	inst.runLock.Unlock()

	// Close stdin to signal shutdown
	inst.stdinPipe.Close()
	// TODO: Put a timeout on this, then force kill
	if err := inst.cmd.Wait(); err != nil {
		return fmt.Errorf("Failed to run: %s", err)
	}
	log.Info("Killed instance")
	return nil
}

func (s *DockerSandbox) boot(code string, env map[string]string) (*instance, error) {
	inst, ok := s.runningInstances[code]
	if !ok {
		var err error
		inst, err = newInstance(code, env)
		if err != nil {
			log.Error("Failed to instantiate", err)
			return nil, err
		}
		if _, _, err := inst.invoke(pingEvent); err != nil {
			log.Error("Failed sending ping event", err)
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
	args = append(args, "-i", "zefhemel/matterless-runner-docker", "node", code)
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

func (s *DockerSandbox) Invoke(event interface{}, code string, env map[string]string) (interface{}, string, error) {
	inst, err := s.boot(code, env)
	if err != nil {
		return nil, "", err
	}

	inst.lastInvoked = time.Now()
	response, logMesssages, err := inst.invoke(event)
	if err != nil {
		return nil, "", err
	}
	return response, strings.TrimSpace(strings.Join(logMesssages, "")), nil
}

func (s *DockerSandbox) cleanup() {
	now := time.Now()
	log.Infof("Cleaning up %d running functions...", len(s.runningInstances))
	for id, inst := range s.runningInstances {
		if inst.lastInvoked.Add(s.keepAlive).Before(now) {
			log.Info("Killing an instance now...", inst)
			if err := inst.kill(); err != nil {
				log.Error("Error killing instance", err)
			}
			delete(s.runningInstances, id)
		}
	}
}

func (s *DockerSandbox) FlushAll() {
	log.Infof("Stopping %d running functions...", len(s.runningInstances))
	for _, inst := range s.runningInstances {
		log.Info("Killing an instance now...", inst)
		if err := inst.kill(); err != nil {
			log.Error("Error killing instance", err)
		}
	}
	s.runningInstances = map[string]*instance{}
}

type jsError struct {
	Message string `json:"message"`
	Stack   string `json:"stack"`
}

func (inst *instance) invoke(event interface{}) (interface{}, []string, error) {
	inst.runLock.Lock()
	defer inst.runLock.Unlock()
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
	if errorMap, ok := result.(map[string]interface{}); ok {
		if errorObj, ok := errorMap["error"]; ok {
			var jsError jsError
			err = json.Unmarshal([]byte(util.MustJsonString(errorObj)), &jsError)
			if err != nil {
				return nil, nil, fmt.Errorf("Runtime error: %s", util.MustJsonString(errorObj))
			}
			return nil, nil, fmt.Errorf("Runtime error: %s\n%s", jsError.Message, jsError.Stack)

		}
	}
	// Wait for all log messages to flush
	<-endOfLogChan

	return result, logMessages, nil
}

func (s *DockerSandbox) Stop() {
	s.ticker.Stop()
	s.stop <- struct{}{}
	s.FlushAll()
}

func (s *DockerSandbox) cleanupJob() {
	for {
		select {
		case <-s.stop:
			return
		case <-s.ticker.C:
			s.cleanup()
		}
	}
}

func NewDockerSandbox(cleanupInterval time.Duration, keepAlive time.Duration) *DockerSandbox {
	sb := &DockerSandbox{
		cleanupInterval:  cleanupInterval,
		keepAlive:        keepAlive,
		runningInstances: map[string]*instance{},
		stop:             make(chan struct{}),
	}
	if cleanupInterval != 0 {
		sb.ticker = time.NewTicker(cleanupInterval)
		go sb.cleanupJob()
	}
	return sb
}

var _ Sandbox = &DockerSandbox{}
