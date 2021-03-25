package sandbox

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/util"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type functionHash string

type DockerSandbox struct {
	runningInstances map[functionHash]*instance // key: functionHash
	cleanupInterval  time.Duration
	keepAlive        time.Duration
	ticker           *time.Ticker
	stop             chan struct{}
	bootLock         sync.Mutex
	logChannel       chan LogEntry
}

// NewDockerSandbox creates a new instance of the sandbox
// Note: It is essential to listen to the .Logs() event channel (probably in a for loop in go routine) as soon as possible
// after instantiation.
func NewDockerSandbox(cleanupInterval time.Duration, keepAlive time.Duration) *DockerSandbox {
	sb := &DockerSandbox{
		cleanupInterval:  cleanupInterval,
		keepAlive:        keepAlive,
		runningInstances: map[functionHash]*instance{},
		logChannel:       make(chan LogEntry),
		stop:             make(chan struct{}),
	}
	if cleanupInterval != 0 {
		sb.ticker = time.NewTicker(cleanupInterval)
		go sb.cleanupJob()
	}
	return sb
}

// Logs receives logs from all functions managed by this sandbox
func (s *DockerSandbox) Logs() chan LogEntry {
	return s.logChannel
}

// Function looks up a running function instance, or boots up a docker container if it doesn't have one yet
// It also performs initialization (cals the init()) function, errors out when no running server runs in time
func (s *DockerSandbox) Function(ctx context.Context, name string, env EnvMap, modules ModuleMap, code string) (FunctionInstance, error) {
	// Only one function can be booted at once for now
	// TODO: Remove this restriction
	s.bootLock.Lock()
	defer s.bootLock.Unlock()

	functionHash := newFunctionHash(modules, env, code)
	inst, ok := s.runningInstances[functionHash]
	if !ok {
		// Create new instance
		var err error
		inst, err = newInstance(ctx, name, s.logChannel, env, modules, code)
		if err != nil {
			return nil, err
		}
		s.runningInstances[functionHash] = inst
	}
	return inst, nil
}

type instance struct {
	containerName string
	controlURL    string
	serverURL     string
	lastInvoked   time.Time
	runLock       sync.Mutex
	name          string
}

var _ FunctionInstance = &instance{}

func (inst *instance) Name() string {
	return inst.name
}

func newInstance(ctx context.Context, name string, logChannel chan LogEntry, env EnvMap, modules ModuleMap, code string) (*instance, error) {
	inst := &instance{
		name:          name,
		containerName: fmt.Sprintf("matterless-%s", newFunctionHash(modules, env, code)),
	}

	// Run "docker run -i" as child process
	cmd := exec.Command("docker", "run", "--rm", "-P", "-i",
		fmt.Sprintf("--name=%s", inst.containerName),
		"zefhemel/matterless-runner-docker", "node")

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "stdout pipe")
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, errors.Wrap(err, "stderr pipe")
	}

	// Kick off the command
	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "docker run")
	}

	// Listen to the stderr and log pipes and ship everything to logChannel
	bufferedStdout := bufio.NewReader(stdoutPipe)
	bufferedStderr := bufio.NewReader(stderrPipe)
	if _, err := bufferedStderr.Peek(1); err != nil {
		log.Error("Could not peek stdout data", err)
	}

	// Send stdout and stderr to the log channel
	go inst.pipeStream(bufferedStdout, logChannel)
	go inst.pipeStream(bufferedStderr, logChannel)

	// Run "docker inspect" to fetch exposed ports
	inspectData, err := exec.Command("docker", "inspect", inst.containerName).CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "docker inspect")
	}
	var dockerInspectOutputs []struct {
		NetworkSettings struct {
			Ports map[string][]struct {
				HostPort string
			}
		}
	}
	if err := json.Unmarshal(inspectData, &dockerInspectOutputs); err != nil {
		return nil, errors.Wrap(err, "parse docker inspect")
	}

	inst.controlURL = fmt.Sprintf("http://localhost:%s", dockerInspectOutputs[0].NetworkSettings.Ports["8081/tcp"][0].HostPort)
	inst.serverURL = fmt.Sprintf("http://localhost:%s", dockerInspectOutputs[0].NetworkSettings.Ports["8080/tcp"][0].HostPort)

	// Initialize the container via the /init call that uploads the code
	if err := inst.init(env, modules, code); err != nil {
		return nil, err
	}

	return inst, nil
}

func newFunctionHash(modules map[string]string, env map[string]string, code string) functionHash {
	// This can probably be optimized, the goal is to generate a unique string representing a mix of the code, modules and environment
	h := sha1.New()
	h.Write([]byte(util.MustJsonString(modules)))
	h.Write([]byte(util.MustJsonString(env)))
	h.Write([]byte(code))
	bs := h.Sum(nil)
	return functionHash(fmt.Sprintf("%x", bs))
}

func (inst *instance) kill() error {
	// Don't kill until current run is over, if any
	inst.runLock.Lock()
	inst.runLock.Unlock()

	// Call /stop on control server, but ignore if this fails for whatever reason
	http.Get(fmt.Sprintf("%s/stop", inst.controlURL))

	// Now hard kill the docker container, if it's still running
	exec.Command("docker", "kill", inst.containerName).Run()

	log.Info("Killed instance.")
	return nil
}

func (inst *instance) pipeStream(bufferedReader *bufio.Reader, logChannel chan LogEntry) {
readLoop:
	for {
		line, err := bufferedReader.ReadString('\n')
		if err == io.EOF {
			break readLoop
		}
		if err != nil {
			log.Error("log read error", err)
			break readLoop
		}
		logChannel <- LogEntry{
			Instance: inst,
			Message:  line,
		}
	}
}

func (s *DockerSandbox) cleanup() {
	now := time.Now()
	if len(s.runningInstances) == 0 {
		return
	}
	log.Debugf("Cleaning up %d running functions...", len(s.runningInstances))
	for id, inst := range s.runningInstances {
		if inst.lastInvoked.Add(s.keepAlive).Before(now) {
			log.Debugf("Killing instance '%s'.", inst.name)
			if err := inst.kill(); err != nil {
				log.Error("Error killing instance", err)
			}
			// Is it ok to delete entries from a map while iterating over it?
			delete(s.runningInstances, id)
		}
	}
}

type jsError struct {
	Message string `json:"message"`
	Stack   string `json:"stack"`
}

func (inst *instance) Invoke(ctx context.Context, event interface{}) (interface{}, error) {
	inst.runLock.Lock()
	defer inst.runLock.Unlock()

	inst.lastInvoked = time.Now()

	httpClient := http.DefaultClient
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, inst.serverURL, strings.NewReader(util.MustJsonString(event)))
	if err != nil {
		return nil, errors.Wrap(err, "invoke call")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("could not make HTTP invocation: %s", err.Error()))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP Error: %s", body)
	}

	var result interface{}
	jsonDecoder := json.NewDecoder(resp.Body)
	if err := jsonDecoder.Decode(&result); err != nil {
		return nil, errors.Wrap(err, "unmarshall response")
	}
	if errorMap, ok := result.(map[string]interface{}); ok {
		if errorObj, ok := errorMap["error"]; ok {
			var jsError jsError
			err = json.Unmarshal([]byte(util.MustJsonString(errorObj)), &jsError)
			if err != nil {
				return nil, fmt.Errorf("Runtime error: %s", util.MustJsonString(errorObj))
			}
			return nil, fmt.Errorf("Runtime error: %s\n%s", jsError.Message, jsError.Stack)

		}
	}

	return result, nil
}

func (inst *instance) init(env EnvMap, modules ModuleMap, code string) error {
	httpClient := http.DefaultClient
	// TODO: Remove magic 15s value
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	initMessage := InitMessage{
		Env:     env,
		Script:  code,
		Modules: modules,
	}
	req, err := http.NewRequestWithContext(timeoutCtx, "POST", fmt.Sprintf("%s/init", inst.controlURL), strings.NewReader(util.MustJsonString(initMessage)))
	if err != nil {
		return errors.Wrap(err, "init call")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "init http call")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Init Error: %s", body)
	}
	return nil
}

func (s *DockerSandbox) Close() {
	// Close the cleanup ticker
	if s.ticker != nil {
		s.ticker.Stop()
		s.stop <- struct{}{}
	}
	s.Flush()

	// Close the log channel
	close(s.logChannel)
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

func (s *DockerSandbox) Flush() {
	// Close all running instances
	log.Infof("Stopping %d running functions...", len(s.runningInstances))
	for _, inst := range s.runningInstances {
		if err := inst.kill(); err != nil {
			log.Error("Error killing instance", err)
		}
	}
	s.runningInstances = map[functionHash]*instance{}
}

var _ Sandbox = &DockerSandbox{}
