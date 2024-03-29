package sandbox

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/util"
)

// ======= Functions ============
type dockerFunctionInstance struct {
	containerName string
	serverURL     string
	lastInvoked   time.Time
	runLock       sync.Mutex
	name          string
	apiURL        string
	procExit      chan error
}

var _ FunctionInstance = &dockerFunctionInstance{}

func (inst *dockerFunctionInstance) Name() string {
	return inst.name
}

func (inst *dockerFunctionInstance) LastInvoked() time.Time {
	return inst.lastInvoked
}

func (inst *dockerFunctionInstance) DidExit() chan error {
	return inst.procExit
}

func newDockerFunctionInstance(ctx context.Context, cfg *config.Config, apiURL string, apiToken string, runMode RunMode, name string, logCallback func(funcName, message string), functionConfig *definition.FunctionConfig, code string, libs definition.LibraryMap) (FunctionInstance, error) {
	//funcHash := newFunctionHash(name, code)
	inst := &dockerFunctionInstance{
		name:          name,
		apiURL:        apiURL,
		containerName: fmt.Sprintf("mls-%s", uuid.NewString()),
		procExit:      make(chan error, 1),
	}

	apiHost := "172.17.0.1"
	if runtime.GOOS != "linux" {
		apiHost = "host.docker.internal"
	}

	// Run "docker run -i" as child process
	cmd := exec.Command("docker", "run", "--rm", "-P", "-i",
		fmt.Sprintf("--name=%s", inst.containerName),
		fmt.Sprintf("-eAPI_URL=%s", fmt.Sprintf(inst.apiURL, apiHost)),
		fmt.Sprintf("-eAPI_TOKEN=%s", apiToken),
		functionConfig.DockerImage)

	stdInPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, errors.Wrap(err, "stdin pipe")
	}
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

	// Send stdout and stderr to the log channel
	go pipeLogStreamToCallback(name, bufferedStdout, logCallback)
	go pipeLogStreamToCallback(name, bufferedStderr, logCallback)

	if code != "" {
		if _, err := stdInPipe.Write([]byte(code)); err != nil {
			log.Errorf("Could not write: %s", err)
		}
		if err := stdInPipe.Close(); err != nil {
			log.Errorf("Could not close: %s", err)
		}
	}

	go func() {
		cmd.Wait()
		log.Info("Docker process exited")
		close(inst.procExit)
	}()

	// Wait for something to come out of stdout or stderr
	if _, err := bufferedStderr.Peek(1); err != nil {
		log.Error("Could not peek stdout data", err)
	}
	// Run "docker inspect" to fetch exposed ports
	inspectData, err := exec.Command("docker", "inspect", inst.containerName).CombinedOutput()
	if err != nil {
		log.Infof("Result: %s", inspectData)
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

	if len(dockerInspectOutputs[0].NetworkSettings.Ports["8080/tcp"]) == 0 {
		return nil, errors.New("invalid docker inspect output")
	}

	inst.serverURL = fmt.Sprintf("http://localhost:%s", dockerInspectOutputs[0].NetworkSettings.Ports["8080/tcp"][0].HostPort)
	log.Info("Server url", inst.serverURL)

	return inst, nil
}

func (inst *dockerFunctionInstance) Kill() {
	// Don't Kill until current run is over, if any
	inst.runLock.Lock()
	inst.runLock.Unlock()

	// Now hard Kill the docker container, if it's still running
	exec.Command("docker", "stop", "-t5", inst.containerName).Run()

	log.Debug("Killed function instance.")
}

type jsError struct {
	Message string `json:"message"`
	Stack   string `json:"stack"`
}

func (inst *dockerFunctionInstance) Invoke(ctx context.Context, event interface{}) (interface{}, error) {
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

// ======= Jobs ============

type dockerJobInstance struct {
	config        *config.Config
	timeStarted   time.Time
	name          string
	cmd           *exec.Cmd
	code          string
	logCallback   func(funcName string, message string)
	apiURL        string
	containerName string
	procExit      chan error
}

var _ JobInstance = &dockerJobInstance{}

func (inst *dockerJobInstance) Name() string {
	return inst.name
}

func newDockerJobInstance(ctx context.Context, cfg *config.Config, apiURL string, apiToken string, name string, logCallback func(funcName, message string), jobConfig *definition.JobConfig, code string, libs definition.LibraryMap) (JobInstance, error) {
	//funcHash := newFunctionHash(name, code)
	inst := &dockerJobInstance{
		apiURL:        apiURL,
		containerName: fmt.Sprintf("mls-%s", uuid.NewString()),
		procExit:      make(chan error, 1),
		config:        cfg,
		timeStarted:   time.Time{},
		name:          name,
		code:          code,
		logCallback:   logCallback,
	}

	apiHost := "172.17.0.1"
	if runtime.GOOS != "linux" {
		apiHost = "host.docker.internal"
	}

	// Run "docker run -i" as child process
	inst.cmd = exec.Command("docker", "run", "--rm", "-i",
		fmt.Sprintf("--name=%s", inst.containerName),
		fmt.Sprintf("-eAPI_URL=%s", fmt.Sprintf(inst.apiURL, apiHost)),
		fmt.Sprintf("-eAPI_TOKEN=%s", apiToken),
		jobConfig.DockerImage)
	inst.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	return inst, nil
}

func (inst *dockerJobInstance) Start(ctx context.Context) error {
	stdoutPipe, err := inst.cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "stdout pipe")
	}
	stderrPipe, err := inst.cmd.StderrPipe()
	if err != nil {
		return errors.Wrap(err, "stderr pipe")
	}
	// If code was supplied, let's pipe that into the docker container's stdin
	stdInPipe, err := inst.cmd.StdinPipe()
	if err != nil {
		return errors.Wrap(err, "stdin pipe")
	}

	// Kick off the command
	if err := inst.cmd.Start(); err != nil {
		return errors.Wrap(err, "docker run")
	}

	// Listen to the stderr and log pipes and ship everything to logChannel
	bufferedStdout := bufio.NewReader(stdoutPipe)
	bufferedStderr := bufio.NewReader(stderrPipe)

	// Send stdout and stderr to the log channel
	go pipeLogStreamToCallback(inst.name, bufferedStdout, inst.logCallback)
	go pipeLogStreamToCallback(inst.name, bufferedStderr, inst.logCallback)

	if inst.code != "" {
		if _, err := stdInPipe.Write([]byte(inst.code)); err != nil {
			log.Errorf("Could not write: %s", err)
		}
		if err := stdInPipe.Close(); err != nil {
			log.Errorf("Could not close: %s", err)
		}
	}

	go func() {
		inst.cmd.Wait()
		close(inst.procExit)
	}()

	return nil
}

func (inst *dockerJobInstance) Stop(ctx context.Context) error {
	if inst.cmd.Process != nil {
		log.Info("Stopping container")
		if err := exec.Command("docker", "stop", "-t", "5", inst.containerName).Run(); err != nil {
			log.Error("docker stop failed", err)
		}
		log.Info("Stopping completed")
	} else {
		log.Error("Container not running (no process)")
	}
	return nil
}

func (inst *dockerJobInstance) DidExit() chan error {
	return inst.procExit
}
