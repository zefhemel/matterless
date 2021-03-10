package main

import (
	"bytes"
	_ "embed"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"text/template"
)

//go:embed templates/template.mjs
var jsTemplate string

type runnerConfig struct {
	bin            string
	template       string
	scriptFilename string
}

var runnerTypes = map[string]runnerConfig{
	"node": {
		bin:            "node",
		scriptFilename: "function.mjs",
		template:       jsTemplate,
	},
}

func wrapScript(runnerConfig runnerConfig, code string) string {
	data := struct {
		Code string
	}{
		Code: code,
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

func run(runnerConfig runnerConfig, code string, env []string, processStdin io.ReadCloser, processStdout io.WriteCloser, processStderr io.WriteCloser) error {
	if err := os.WriteFile(runnerConfig.scriptFilename, []byte(wrapScript(runnerConfig, code)), 0600); err != nil {
		return err
	}
	cmd := exec.Command(runnerConfig.bin, runnerConfig.scriptFilename)

	cmd.Env = env
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return errors.Wrap(err, "stdin pipe")
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "stdout pipe")
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errors.Wrap(err, "stderr pipe")
	}

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "start")
	}

	go func() {
		if _, err := io.Copy(processStdout, stdout); err != nil {
			log.Error("Piping stdout", err)
		}
	}()
	go func() {
		if _, err := io.Copy(processStderr, stderr); err != nil {
			log.Error("Piping stderr", err)
		}
	}()
	go func() {
		_, err = io.Copy(stdin, processStdin)
		stdin.Close()
	}()
	if err := cmd.Wait(); err != nil {
		os.Exit(cmd.ProcessState.ExitCode())
	}
	return nil
}

func main() {
	if len(os.Args) != 3 {
		log.Fatal("Expected two arguments: [runnerType] [script]")
	}
	if err := run(runnerTypes[os.Args[1]], os.Args[2], os.Environ(), os.Stdin, os.Stdout, os.Stderr); err != nil {
		log.Fatal(err)
	}
}
