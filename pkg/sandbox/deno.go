package sandbox

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/config"
	"github.com/zefhemel/matterless/pkg/definition"
	"github.com/zefhemel/matterless/pkg/eventbus"
	"github.com/zefhemel/matterless/pkg/util"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"text/template"
	"time"
)

// ======= Functions ============
type denoFunctionInstance struct {
	serverURL   string
	lastInvoked time.Time
	runLock     sync.Mutex
	name        string
	cmd         *exec.Cmd
	tempDir     string
	config      *config.Config
}

var _ FunctionInstance = &denoFunctionInstance{}

func (inst *denoFunctionInstance) Name() string {
	return inst.name
}

func (inst *denoFunctionInstance) LastInvoked() time.Time {
	return inst.lastInvoked
}

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

func wrapScript(configMap map[string]interface{}, code string) string {
	if configMap == nil {
		configMap = map[string]interface{}{}
	}
	data := struct {
		Code       string
		ConfigJSON string
	}{
		Code:       code,
		ConfigJSON: util.MustJsonString(configMap),
	}
	tmpl, err := template.New("sourceTemplate").Parse(denoFunctionTemplate)
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

func newDenoFunctionInstance(ctx context.Context, apiURL string, runMode string, name string, eventBus eventbus.EventBus, env EnvMap, modules ModuleMap, functionConfig definition.FunctionConfig, code string) (*denoFunctionInstance, error) {
	inst := &denoFunctionInstance{
		name: name,
	}

	denoDir, err := os.MkdirTemp(os.TempDir(), "mls-deno")
	inst.tempDir = denoDir
	if err != nil {
		return nil, errors.Wrap(err, "create temp dir")
	}

	if err := copyDenoFiles(denoDir); err != nil {
		return nil, errors.Wrap(err, "copy deno files")
	}

	if err := os.WriteFile(fmt.Sprintf("%s/function.js", denoDir), []byte(wrapScript(functionConfig.Config, code)), 0600); err != nil {
		return nil, errors.Wrap(err, "write JS function file")
	}

	listenPort := util.FindFreePort(8000)

	// Run "docker run -i" as child process
	inst.cmd = exec.Command("deno", "run", "--allow-net", "--allow-env", fmt.Sprintf("%s/%s_server.ts", denoDir, runMode), fmt.Sprintf("%d", listenPort))
	inst.cmd.Env = append(inst.cmd.Env, "NO_COLOR=1", fmt.Sprintf("API_URL=%s", fmt.Sprintf(apiURL, "localhost")))

	for k, v := range env {
		inst.cmd.Env = append(inst.cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	stdoutPipe, err := inst.cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "stdout pipe")
	}
	stderrPipe, err := inst.cmd.StderrPipe()
	if err != nil {
		return nil, errors.Wrap(err, "stderr pipe")
	}

	// Kick off the command
	if err := inst.cmd.Start(); err != nil {
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

func (inst *denoFunctionInstance) Kill() error {
	// Don't Kill until current run is over, if any
	inst.runLock.Lock()
	inst.runLock.Unlock()

	if err := inst.cmd.Process.Kill(); err != nil {
		log.Errorf("Error killing deno instance: %s", err)
	}

	log.Info("Killed deno process, cleaning files from disk.")
	if err := os.RemoveAll(inst.tempDir); err != nil {
		log.Errorf("Could not delete directory %s: %s", inst.tempDir, err)
	}

	return nil
}

func (inst *denoFunctionInstance) pipeStream(bufferedReader *bufio.Reader, eventBus eventbus.EventBus) {
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

func (inst *denoFunctionInstance) Invoke(ctx context.Context, event interface{}) (interface{}, error) {
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

type denoJobInstance struct {
	denoFunctionInstance
	timeStarted time.Time
	name        string
}

var _ JobInstance = &denoJobInstance{}

func (inst *denoJobInstance) Name() string {
	return inst.name
}

func newDenoJobInstance(ctx context.Context, apiURL string, name string, eventBus eventbus.EventBus, env EnvMap, modules ModuleMap, functionConfig definition.FunctionConfig, code string) (*denoJobInstance, error) {
	inst := &denoJobInstance{
		name: name,
	}

	functionInstance, err := newDenoFunctionInstance(ctx, apiURL, "job", name, eventBus, env, modules, functionConfig, code)
	if err != nil {
		return nil, err
	}

	inst.denoFunctionInstance = *functionInstance

	return inst, nil
}

func (inst *denoJobInstance) Start(ctx context.Context) error {
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

func (inst *denoJobInstance) Stop() {
	http.Get(fmt.Sprintf("%s/stop", inst.serverURL))
}
