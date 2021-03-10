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

//go:embed templates/template.js
var jsTemplate string

func wrapScript(code string) string {
	data := struct {
		Code string
	}{
		Code: code,
	}
	tmpl, err := template.New("sourceTemplate").Parse(jsTemplate)
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

func run(runnerType string, code string, env []string, processStdin io.ReadCloser, processStdout io.WriteCloser, processStderr io.WriteCloser) error {
	if err := os.WriteFile("function.mjs", []byte(wrapScript(code)), 0600); err != nil {
		return err
	}
	cmd := exec.Command("node", "function.mjs")

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
	if err := run("node", os.Args[1], os.Environ(), os.Stdin, os.Stdout, os.Stderr); err != nil {
		log.Fatal(err)
	}
}
