package sandbox

import (
	"bufio"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/eventbus"
	"io"
)

func pipeLogStreamToEventBus(functionName string, bufferedReader *bufio.Reader, eventBus eventbus.EventBus) {
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
