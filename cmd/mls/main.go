package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/application"
	"os"
	"time"
)

func main() {
	log.SetLevel(log.DebugLevel)
	filename := "matterless.md"
	if len(os.Args) > 0 {
		filename = os.Args[1]
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Could not open file %s: %s", filename, err)
	}
	app := application.NewApplication(func(kind, message string) {
		log.Infof("%s: %s", kind, message)
	})
	err = app.Eval(string(data))
	if err != nil {
		log.Fatal(err)
	}
	log.Info("HEre")
	for {
		time.Sleep(30 * time.Second)
		app.FlushSandbox()
		log.Info("Flushed sandbox")
	}
}
