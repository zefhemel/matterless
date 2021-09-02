package sandbox

import (
	"bufio"
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/eventbus"
)

func pipeLogStreamToEventBus(functionName string, bufferedReader *bufio.Reader, eventBus *eventbus.LocalEventBus) {
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
		eventBus.Publish(fmt.Sprintf("logs:%s", functionName), LogEntry{
			FunctionName: functionName,
			Message:      line,
		})
	}
}
