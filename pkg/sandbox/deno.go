package sandbox

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/util"
)

// ======= Function ============
type denoFunctionInstance struct {
	config      *config.Config
	name        string
	cmd         *exec.Cmd
	lastInvoked time.Time
	runLock     sync.Mutex
	serverURL   string
	tempDir     string
	denoExited  chan error
}

var _ FunctionInstance = &denoFunctionInstance{}

func (inst *denoFunctionInstance) Name() string {
	return inst.name
}

func (inst *denoFunctionInstance) LastInvoked() time.Time {
	return inst.lastInvoked
}

// All these files will be copied into a temporary function directory that deno will be invoked on
//go:embed deno/*.ts
var denoFiles embed.FS

func copyDenoFiles(destDir string) error {
	dirEntries, _ := denoFiles.ReadDir("deno")
	for _, file := range dirEntries {
		buf, err := denoFiles.ReadFile(fmt.Sprintf("deno/%s", file.Name()))
		if err != nil {
			return errors.Wrap(err, "read file")
		}
		if err := os.WriteFile(fmt.Sprintf("%s/%s", destDir, file.Name()), buf, 0600); err != nil {
			return errors.Wrap(err, "write file")
		}
	}
	return nil
}

//go:embed deno/template.js
var denoFunctionTemplate string

func wrapScript(initData interface{}, code string) string {
	data := struct {
		Code     string
		InitData string
	}{
		Code:     code,
		InitData: util.MustJsonString(initData),
	}
	tmpl, err := template.New("sourceTemplate").Parse(denoFunctionTemplate)
	if err != nil {
		log.Fatal("Could not render javascript:", err)
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		log.Fatal("Could not render javascript:", err)
	}
	return out.String()
}

type functionHash string

// Generates a content-based hash to be used as unique identifier for this function
func newFunctionHash(name string, code string) functionHash {
	h := sha1.New()
	h.Write([]byte(name))
	h.Write([]byte(code))
	bs := h.Sum(nil)
	return functionHash(fmt.Sprintf("%x", bs))
}

func newDenoFunctionInstance(ctx context.Context, config *config.Config, apiURL string, apiToken string, runMode RunMode, name string, logCallback func(funcName, message string), functionConfig *definition.FunctionConfig, code string) (FunctionInstance, error) {
	inst := &denoFunctionInstance{
		name:   name,
		config: config,
	}

	runModeString := "function"
	if runMode == RunModeJob {
		runModeString = "job"
	}

	// Create deno project for function
	denoDir := fmt.Sprintf("%s/.deno/%s-%s", config.DataDir, runModeString, newFunctionHash(name, code))
	if err := os.MkdirAll(denoDir, 0700); err != nil {
		return nil, errors.Wrap(err, "create deno dir")
	}
	inst.tempDir = denoDir

	if err := copyDenoFiles(denoDir); err != nil {
		return nil, errors.Wrap(err, "copy deno files")
	}

	if err := os.WriteFile(fmt.Sprintf("%s/function.js", denoDir), []byte(wrapScript(functionConfig.Init, code)), 0600); err != nil {
		return nil, errors.Wrap(err, "write JS function file")
	}

	// Find an available TCP port to bind the function server to
	listenPort := util.FindFreePort(8000)

	// Run deno as child process with only network and environment variable access
	inst.cmd = exec.Command(denoBinPath(config), "run", "--allow-net", "--allow-env", fmt.Sprintf("%s/%s_server.ts", denoDir, runModeString), fmt.Sprintf("%d", listenPort))

	// Don't propagate Ctrl-c to children
	inst.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	inst.cmd.Env = append(inst.cmd.Env,
		"NO_COLOR=1",
		fmt.Sprintf("DENO_DIR=%s/.deno/cache", config.DataDir),
		fmt.Sprintf("API_URL=%s", fmt.Sprintf(apiURL, "localhost")),
		fmt.Sprintf("API_TOKEN=%s", apiToken))

	stdoutPipe, err := inst.cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "stdout pipe")
	}
	stderrPipe, err := inst.cmd.StderrPipe()
	if err != nil {
		return nil, errors.Wrap(err, "stderr pipe")
	}

	// Kick off the command in the background
	// Making it buffered to prevent go-routine leak (we don't care for the result after initial start-up)
	inst.denoExited = make(chan error, 1)
	if err := inst.cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "deno run")
	}
	go func() {
		inst.denoExited <- inst.cmd.Wait()
	}()

	// Listen to the stderr and log pipes and ship everything to logChannel
	bufferedStdout := bufio.NewReader(stdoutPipe)
	bufferedStderr := bufio.NewReader(stderrPipe)

	// Send stdout and stderr to the log channel
	go pipeLogStreamToCallback(name, bufferedStdout, logCallback)
	go pipeLogStreamToCallback(name, bufferedStderr, logCallback)

	inst.serverURL = fmt.Sprintf("http://localhost:%d", listenPort)

	// Wait for server to come up
waitLoop:
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				return nil, ctx.Err()
			}
			break waitLoop
		case <-inst.denoExited:
			//log.Info("Exited ", err)
			return nil, errors.New("deno exited on boot")
		default:
		}
		_, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", listenPort))
		if err == nil {
			break waitLoop
		}
		time.Sleep(100 * time.Millisecond)
	}

	return inst, nil
}

func (inst *denoFunctionInstance) Kill() {
	if err := inst.cmd.Process.Kill(); err != nil {
		log.Errorf("Error killing deno instance: %s", err)
	}

	if err := os.RemoveAll(inst.tempDir); err != nil {
		log.Errorf("Could not delete directory %s: %s", inst.tempDir, err)
	}
}

type InvocationError struct {
	err error
}

var ProcessExitedError = errors.New("process exited")

func (inst *denoFunctionInstance) Invoke(ctx context.Context, event interface{}) (interface{}, error) {
	// Instance can only be used sequentially for now
	inst.runLock.Lock()
	defer inst.runLock.Unlock()

	inst.lastInvoked = time.Now()

	if inst.cmd.ProcessState != nil && inst.cmd.ProcessState.Exited() {
		return nil, ProcessExitedError
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, inst.serverURL, strings.NewReader(util.MustJsonString(event)))
	if err != nil {
		return nil, errors.Wrap(err, "invoke call")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "function http request")
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

type denoJobInstance struct {
	// Jobs are mostly functions with only a few differences
	denoFunctionInstance
}

var _ JobInstance = &denoJobInstance{}

func (inst *denoJobInstance) Name() string {
	return inst.name
}

func newDenoJobInstance(ctx context.Context, config *config.Config, apiURL string, apiToken string, name string, logCallback func(funcName, message string), jobConfig *definition.JobConfig, code string) (JobInstance, error) {
	inst := &denoJobInstance{}

	functionInstance, err := newDenoFunctionInstance(ctx, config, apiURL, apiToken, RunModeJob, name, logCallback, &definition.FunctionConfig{
		Init:        jobConfig.Init,
		Runtime:     jobConfig.Runtime,
		Prewarm:     false,
		Instances:   jobConfig.Instances,
		DockerImage: jobConfig.DockerImage,
	}, code)
	if err != nil {
		return nil, err
	}

	inst.denoFunctionInstance = *(functionInstance.(*denoFunctionInstance))

	return inst, nil
}

func (inst *denoJobInstance) Start(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/start", inst.serverURL), nil)
	if err != nil {
		return errors.Wrap(err, "invoke call")
	}
	resp, err := http.DefaultClient.Do(req)
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

func (inst *denoJobInstance) Stop(ctx context.Context) error {
	defer inst.Kill()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/stop", inst.serverURL), nil)
	if err != nil {
		return errors.Wrap(err, "stop call")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not make HTTP invocation: %s", err.Error()))
	}
	resp.Body.Close()
	return nil
}

func (inst *denoJobInstance) DidExit() chan error {
	return inst.denoExited
}
