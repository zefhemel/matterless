package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/sandbox"
	"github.com/zefhemel/matterless/pkg/util"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"text/template"
	"time"
)

// https://docs.openfaas.com/reference/workloads/

//go:embed templates/template.mjs
var jsTemplate string

type runnerConfig struct {
	cmd            []string
	template       string
	scriptFilename string
}

var runnerTypes = map[string]runnerConfig{
	"node-function": {
		cmd:            []string{"node", "function_server.mjs"},
		scriptFilename: "function.mjs",
		template:       jsTemplate,
	},
	"node-job": {
		cmd:            []string{"node", "job_server.mjs"},
		scriptFilename: "function.mjs",
		template:       jsTemplate,
	},
}

func wrapScript(runnerConfig runnerConfig, configMap map[string]interface{}, code string) string {
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
	tmpl, err := template.New("sourceTemplate").Parse(runnerConfig.template)
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

var cmd *exec.Cmd

func run(runnerConfig runnerConfig, processStdout io.WriteCloser, processStderr io.WriteCloser) error {
	cmd = exec.Command(runnerConfig.cmd[0], runnerConfig.cmd[1:]...)
	cmd.Env = os.Environ()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var allOutputBuffer bytes.Buffer

	multiplexedProcessStdout := io.MultiWriter(processStdout, &allOutputBuffer)
	multiplexedProcessStderr := io.MultiWriter(processStderr, &allOutputBuffer)

	go func() {
		if _, err := io.Copy(multiplexedProcessStdout, stdout); err != nil {
			log.Fatalf("stdout pipe: %s", err)
		}
	}()
	go func() {
		if _, err := io.Copy(multiplexedProcessStderr, stderr); err != nil {
			log.Fatalf("stderr pipe: %s", err)
		}
	}()

	if err := cmd.Wait(); err != nil {
		return errors.New(allOutputBuffer.String())
	}

	return nil
}

func main() {
	log.SetLevel(log.DebugLevel)
	if len(os.Args) != 2 {
		log.Fatal("Expected argument: [runnerType]")
	}

	runnerConfig := runnerTypes[os.Args[1]]

	http.HandleFunc("/init", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if cmd != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Already inited")
			return
		}
		var initMessage sandbox.DockerInitMessage
		defer r.Body.Close()
		reader := json.NewDecoder(r.Body)
		if err := reader.Decode(&initMessage); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "JSON parse error: %s", err)
			scheduleShutdown()
			return
		}
		// Write script file
		if err := os.WriteFile(runnerConfig.scriptFilename, []byte(wrapScript(runnerConfig, initMessage.Config, initMessage.Script)), 0600); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error writing script: %s", err)
			scheduleShutdown()
			return
		}

		errorChan := make(chan error, 1)
		go func() {
			if err := run(runnerConfig, os.Stdout, os.Stderr); err != nil {
				errorChan <- err
			}
		}()

		// Wait for server to go up
		// Bootup shouldn't take longer than 15s
		// TODO: Remove magic value
	waitLoop:
		for {
			select {
			case <-ctx.Done():
				if ctx.Err() == context.DeadlineExceeded {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprint(w, "Deadline exceeded")
					scheduleShutdown()
					return
				}
				break waitLoop
			case err := <-errorChan:
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, err.Error())
				scheduleShutdown()
				return
			default:
			}
			_, err := net.Dial("tcp", "localhost:8080")
			if err == nil {
				break waitLoop
			}
			time.Sleep(100 * time.Millisecond)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})
	http.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		http.Get("http://localhost:8080/stop")
		if cmd != nil && cmd.Process != nil {
			cmd.Process.Kill()
		}
		log.Info("Function runtime shut down.")
		os.Exit(0)
	})

	log.Info("Function runtime booted.")
	log.Fatal(http.ListenAndServe(":8081", nil))

}

func scheduleShutdown() {
	go func() {
		time.Sleep(500 * time.Millisecond)
		os.Exit(1)
	}()
}
