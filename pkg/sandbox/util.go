package sandbox

import (
	"bufio"
	"io"

	log "github.com/sirupsen/logrus"
)

func pipeLogStreamToCallback(functionName string, bufferedReader *bufio.Reader, callback func(funcName string, message string)) {
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
		callback(functionName, line)
	}
}
