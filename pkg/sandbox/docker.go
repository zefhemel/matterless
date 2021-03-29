package sandbox

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/eventbus"
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
	runningFunctions map[functionHash]*functionInstance
	runningJobs      map[string]*jobInstance // key: job name
	cleanupInterval  time.Duration
	keepAlive        time.Duration
	ticker           *time.Ticker
	stop             chan struct{}
	bootLock         sync.Mutex
	eventBus         eventbus.EventBus
}

// NewDockerSandbox creates a new functionInstance of the sandbox
// Note: It is essential to listen to the .Logs() event channel (probably in a for loop in go routine) as soon as possible
// after instantiation.
func NewDockerSandbox(eventBus eventbus.EventBus, cleanupInterval time.Duration, keepAlive time.Duration) *DockerSandbox {
	sb := &DockerSandbox{
		cleanupInterval:  cleanupInterval,
		keepAlive:        keepAlive,
		runningFunctions: map[functionHash]*functionInstance{},
		runningJobs:      map[string]*jobInstance{},
		eventBus:         eventBus,
		stop:             make(chan struct{}),
	}
	if cleanupInterval != 0 {
		sb.ticker = time.NewTicker(cleanupInterval)
		go sb.cleanupJob()
	}
	return sb
}

// Function looks up a running function functionInstance, or boots up a docker container if it doesn't have one yet
// It also performs initialization (cals the init()) function, errors out when no running server runs in time
func (s *DockerSandbox) Function(ctx context.Context, name string, env EnvMap, modules ModuleMap, functionConfig definition.FunctionConfig, code string) (FunctionInstance, error) {
	// Only one function can be booted at once for now
	// TODO: Remove this restriction
	s.bootLock.Lock()
	defer s.bootLock.Unlock()

	functionHash := newFunctionHash(modules, env, functionConfig, code)
	inst, ok := s.runningFunctions[functionHash]
	if !ok {
		// Create new functionInstance
		var err error
		inst, err = newFunctionInstance(ctx, "node-function", name, s.eventBus, env, modules, functionConfig, code)
		if err != nil {
			return nil, err
		}
		s.runningFunctions[functionHash] = inst
	}
	return inst, nil
}

func (s *DockerSandbox) Job(ctx context.Context, name string, env EnvMap, modules ModuleMap, functionConfig definition.FunctionConfig, code string) (JobInstance, error) {
	// Only one function can be booted at once for now
	// TODO: Remove this restriction
	s.bootLock.Lock()
	defer s.bootLock.Unlock()

	inst, ok := s.runningJobs[name]
	if !ok {
		// Create new functionInstance
		var err error
		inst, err = newJobInstance(ctx, name, s.eventBus, env, modules, functionConfig, code)
		if err != nil {
			return nil, err
		}
		s.runningJobs[name] = inst
	}
	return inst, nil
}

type functionInstance struct {
	containerName string
	controlURL    string
	serverURL     string
	lastInvoked   time.Time
	runLock       sync.Mutex
	name          string
}

var _ FunctionInstance = &functionInstance{}

func (inst *functionInstance) Name() string {
	return inst.name
}

func newFunctionInstance(ctx context.Context, runnerType string, name string, eventBus eventbus.EventBus, env EnvMap, modules ModuleMap, functionConfig definition.FunctionConfig, code string) (*functionInstance, error) {
	inst := &functionInstance{
		name:          name,
		containerName: fmt.Sprintf("mls-%s-%s", runnerType, newFunctionHash(modules, env, functionConfig, code)),
	}

	// Run "docker run -i" as child process
	cmd := exec.Command("docker", "run", "--rm", "-P", "-i",
		fmt.Sprintf("--name=%s", inst.containerName),
		"zefhemel/matterless-runner-docker", runnerType)

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
	go inst.pipeStream(bufferedStdout, eventBus)
	go inst.pipeStream(bufferedStderr, eventBus)

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

	if len(dockerInspectOutputs) == 0 {
		return nil, errors.New("invalid docker inspect output")
	}

	if len(dockerInspectOutputs[0].NetworkSettings.Ports["8081/tcp"]) == 0 {
		return nil, errors.New("invalid docker inspect output")
	}

	inst.controlURL = fmt.Sprintf("http://localhost:%s", dockerInspectOutputs[0].NetworkSettings.Ports["8081/tcp"][0].HostPort)
	inst.serverURL = fmt.Sprintf("http://localhost:%s", dockerInspectOutputs[0].NetworkSettings.Ports["8080/tcp"][0].HostPort)

	// Initialize the container via the /init call that uploads the code
	if err := inst.init(env, modules, functionConfig, code); err != nil {
		return nil, err
	}

	return inst, nil
}

func (inst *functionInstance) kill() error {
	// Don't kill until current run is over, if any
	inst.runLock.Lock()
	inst.runLock.Unlock()

	// Call /stop on control server, but ignore if this fails for whatever reason
	http.Get(fmt.Sprintf("%s/stop", inst.controlURL))

	// Now hard kill the docker container, if it's still running
	exec.Command("docker", "kill", inst.containerName).Run()

	log.Info("Killed functionInstance.")
	return nil
}

func (inst *functionInstance) pipeStream(bufferedReader *bufio.Reader, eventBus eventbus.EventBus) {
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
		eventBus.Publish(fmt.Sprintf("logs:%s", inst.name), LogEntry{
			Instance: inst,
			Message:  line,
		})
	}
}

func (s *DockerSandbox) cleanup() {
	now := time.Now()
	if len(s.runningFunctions) == 0 {
		return
	}
	log.Debugf("Cleaning up %d running functions...", len(s.runningFunctions))
	for id, inst := range s.runningFunctions {
		if inst.lastInvoked.Add(s.keepAlive).Before(now) {
			log.Debugf("Killing functionInstance '%s'.", inst.name)
			if err := inst.kill(); err != nil {
				log.Error("Error killing functionInstance", err)
			}
			// Is it ok to delete entries from a map while iterating over it?
			delete(s.runningFunctions, id)
		}
	}
}

type jsError struct {
	Message string `json:"message"`
	Stack   string `json:"stack"`
}

func (inst *functionInstance) Invoke(ctx context.Context, event interface{}) (interface{}, error) {
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

func (inst *functionInstance) init(env EnvMap, modules ModuleMap, functionConfig definition.FunctionConfig, code string) error {
	httpClient := http.DefaultClient
	// TODO: Remove magic 15s value
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	initMessage := InitMessage{
		Env:     env,
		Config:  functionConfig.Config,
		Script:  code,
		Modules: modules,
	}
	req, err := http.NewRequestWithContext(timeoutCtx, "POST", fmt.Sprintf("%s/init", inst.controlURL), strings.NewReader(util.MustJsonString(initMessage)))
	if err != nil {
		return errors.Wrap(err, "init call")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "init http call: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Init Error: %s", body)
	}
	return nil
}

// Jobs

type jobInstance struct {
	functionInstance
	timeStarted time.Time
	name        string
}

var _ JobInstance = &jobInstance{}

func (inst *jobInstance) Name() string {
	return inst.name
}

func newJobInstance(ctx context.Context, name string, eventBus eventbus.EventBus, env EnvMap, modules ModuleMap, functionConfig definition.FunctionConfig, code string) (*jobInstance, error) {
	inst := &jobInstance{
		name: name,
	}

	functionInstance, err := newFunctionInstance(ctx, "node-job", name, eventBus, env, modules, functionConfig, code)
	if err != nil {
		return nil, err
	}

	inst.functionInstance = *functionInstance

	return inst, nil
}

func (inst *jobInstance) Start(ctx context.Context) error {
	inst.timeStarted = time.Now()

	httpClient := http.DefaultClient
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/start", inst.serverURL), nil)
	if err != nil {
		return errors.Wrap(err, "invoke call")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not make HTTP invocation: %s", err.Error()))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP Error: %s", body)
	}

	return nil
}

func (inst *jobInstance) Stop() {
	http.Get(fmt.Sprintf("%s/stop", inst.serverURL))
}

// End

func (s *DockerSandbox) Close() {
	// Close the cleanup ticker
	if s.ticker != nil {
		s.ticker.Stop()
		s.stop <- struct{}{}
	}
	s.Flush()
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
	log.Infof("Stopping %d running functions...", len(s.runningFunctions))
	for _, inst := range s.runningFunctions {
		if err := inst.kill(); err != nil {
			log.Error("Error killing functionInstance", err)
		}
	}
	s.runningFunctions = map[functionHash]*functionInstance{}
	// Close all running jobs
	log.Infof("Stopping %d running jobs...", len(s.runningJobs))
	for _, inst := range s.runningJobs {
		if err := inst.kill(); err != nil {
			log.Error("Error killing jobInstance", err)
		}
	}
	s.runningJobs = map[string]*jobInstance{}
}

func newFunctionHash(modules map[string]string, env map[string]string, functionConfig definition.FunctionConfig, code string) functionHash {
	// This can probably be optimized, the goal is to generate a unique string representing a mix of the code, modules and environment
	h := sha1.New()
	h.Write([]byte(util.MustJsonString(modules)))
	h.Write([]byte(util.MustJsonString(env)))
	h.Write([]byte(util.MustJsonString(functionConfig)))
	h.Write([]byte(code))
	bs := h.Sum(nil)
	return functionHash(fmt.Sprintf("%x", bs))
}

var _ Sandbox = &DockerSandbox{}
