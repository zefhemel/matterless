package cluster

import (
	"encoding/json"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/zefhemel/matterless/pkg/config"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/util"
)

// Checks if a NATS port is running at the given hostName and port, if not and if
// the hostname is localhost/127.0.0.1 will boot a new NATS server
func ConnectOrBoot(config *config.Config) (*nats.Conn, error) {
	nc, err := nats.Connect(config.ClusterNatsUrl)
	if err != nil {
		parsedUrl, err := url.Parse(config.ClusterNatsUrl)
		if err != nil {
			return nil, err
		}
		if parsedUrl.Hostname() == "localhost" || parsedUrl.Hostname() == "127.0.0.1" {
			// Attempt to boot it locally
			log.Debug("Booting NATS server")
			p, err := strconv.Atoi(parsedUrl.Port())
			if err != nil {
				return nil, errors.Wrap(err, "parsing port")
			}
			err = spawnNatsServer(path.Join(config.DataDir, ".nats"), p)
			if err != nil {
				return nil, errors.Wrap(err, "nats spawn")
			}
		}
		nc, err = nats.Connect(config.ClusterNatsUrl)
		if err != nil {
			return nil, errors.Wrap(err, "after nats server boot")
		}
	}
	return nc, nil
}

func spawnNatsServer(dataDir string, port int) error {
	opts := &server.Options{
		Host: "0.0.0.0",
		Port: port,

		// Jetstream
		JetStream: true,
		StoreDir:  dataDir,
	}

	s, err := server.NewServer(opts)
	if err != nil {
		return err
	}

	s.ConfigureLogger()
	go s.Start()

	if !s.ReadyForConnections(10 * time.Second) {
		return errors.New("server creation timeout")
	}

	return nil
}

type Subscription interface {
	Unsubscribe() error
}

type ClusterEventBus struct {
	conn   *nats.Conn
	prefix string
}

func NewClusterEventBus(conn *nats.Conn, prefix string) *ClusterEventBus {
	return &ClusterEventBus{
		conn:   conn,
		prefix: prefix,
	}
}

func (eb *ClusterEventBus) publish(name string, data []byte) error {
	return eb.conn.Publish(fmt.Sprintf("%s.%s", eb.prefix, name), data)
}

func (eb *ClusterEventBus) request(name string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	return eb.conn.Request(fmt.Sprintf("%s.%s", eb.prefix, name), data, timeout)
}

func (eb *ClusterEventBus) subscribe(name string, callback func(msg *nats.Msg)) (Subscription, error) {
	return eb.conn.Subscribe(fmt.Sprintf("%s.%s", eb.prefix, name), callback)
}

func (eb *ClusterEventBus) queueSubscribe(name string, queue string, callback func(msg *nats.Msg)) (Subscription, error) {
	return eb.conn.QueueSubscribe(fmt.Sprintf("%s.%s", eb.prefix, name), fmt.Sprintf("%s.%s", eb.prefix, queue), callback)
}

func (eb *ClusterEventBus) InvokeFunction(name string, event interface{}) (interface{}, error) {
	resp, err := eb.request(fmt.Sprintf("function.%s", SafeNATSSubject(name)), util.MustJsonByteSlice(functionInvoke{
		Data: event,
	}), 10*time.Second)
	if err != nil {
		return nil, err
	}
	var respMsg functionResult
	if err := json.Unmarshal(resp.Data, &respMsg); err != nil {
		return nil, err
	}
	if respMsg.IsError {
		return nil, errors.New(respMsg.Error)
	}
	return respMsg.Data, nil
}

func (eb *ClusterEventBus) SubscribeInvokeFunction(name string, callback func(interface{}) (interface{}, error)) (Subscription, error) {
	return eb.queueSubscribe(fmt.Sprintf("function.%s", SafeNATSSubject(name)), fmt.Sprintf("function.%s.workers", SafeNATSSubject(name)), func(msg *nats.Msg) {
		var requestMessage functionInvoke
		if err := json.Unmarshal(msg.Data, &requestMessage); err != nil {
			log.Errorf("Could not unmarshal event data: %s", err)
			err = msg.Respond([]byte(util.MustJsonByteSlice(functionResult{
				IsError: true,
				Error:   err.Error(),
			})))
			if err != nil {
				log.Errorf("Could not respond with error message: %s", err)
			}
			return
		}
		resp, err := callback(requestMessage.Data)
		if err != nil {
			if err := msg.Respond([]byte(util.MustJsonByteSlice(functionResult{
				IsError: true,
				Error:   err.Error(),
			}))); err != nil {
				log.Errorf("Could not respond with error message: %s", err)
			}
			return
		}
		if err := msg.Respond([]byte(util.MustJsonByteSlice(functionResult{
			Data: resp,
		}))); err != nil {
			log.Errorf("Could not respond with response: %s", err)
		}
	})
}

func (eb *ClusterEventBus) SubscribeLogs(funcName string, callback func(funcName string, message string)) (Subscription, error) {
	return eb.SubscribeEvent(fmt.Sprintf("%s.log", funcName), func(name string, data interface{}, msg *nats.Msg) {
		var lm logMessage
		if err := mapstructure.Decode(data, &lm); err != nil {
			log.Errorf("Error unmarshaling log message: %s", err)
			return
		}
		callback(lm.Function, lm.Message)
	})
}

func (eb *ClusterEventBus) SubscribeContainerLogs(s string, callback func(appName, funcName, message string)) (Subscription, error) {
	return eb.subscribe("*.*.log", func(msg *nats.Msg) {
		parts := strings.Split(msg.Subject, ".") // mls.myapp.MyFunction.log
		var pe publishEvent
		if err := json.Unmarshal(msg.Data, &pe); err != nil {
			log.Errorf("Could not unmarshal event data: %s", err)
			return
		}
		var lm logMessage
		if err := mapstructure.Decode(pe.Data, &lm); err != nil {
			log.Errorf("Error unmarshaling log message: %s", err)
			return
		}
		callback(parts[1], lm.Function, lm.Message)
	})
}

func (eb *ClusterEventBus) PublishLog(funcName, message string) error {
	return eb.PublishEvent(fmt.Sprintf("%s.log", SafeNATSSubject(funcName)), logMessage{
		Function: funcName,
		Message:  message,
	})
}

func (eb *ClusterEventBus) FetchClusterInfo(wait time.Duration) (*ClusterInfo, error) {
	// TODO: Generate unique ID some other way
	responseSubject := fmt.Sprintf("clusterinfo.%s", strings.ReplaceAll(uuid.NewString(), "-", ""))

	ci := &ClusterInfo{
		Nodes: map[uint64]*NodeInfo{},
	}
	var mutex sync.Mutex

	sub, err := eb.subscribe(responseSubject, func(msg *nats.Msg) {
		var ni NodeInfo
		if err := json.Unmarshal(msg.Data, &ni); err != nil {
			log.Errorf("Could not unmarshal node info: %s", err)
			return
		}
		mutex.Lock()
		ci.Nodes[ni.ID] = &ni
		defer mutex.Unlock()
	})
	if err != nil {
		return nil, err
	}
	defer sub.Unsubscribe()
	if err := eb.publish(EventFetchNodeInfo, util.MustJsonByteSlice(FetchNodeInfo{
		ReplyTo: responseSubject,
	})); err != nil {
		return nil, err
	}

	// Now give all nodes time to respond
	time.Sleep(wait)

	// Enough already!
	return ci, nil
}

func (eb *ClusterEventBus) SubscribeFetchClusterInfo(callback func() *NodeInfo) (Subscription, error) {
	return eb.subscribe(EventFetchNodeInfo, func(msg *nats.Msg) {
		var fni FetchNodeInfo
		if err := json.Unmarshal(msg.Data, &fni); err != nil {
			log.Errorf("Could not unmarshal fetch node info: %s", err)
			return
		}
		if err := eb.publish(fni.ReplyTo, util.MustJsonByteSlice(callback())); err != nil {
			log.Errorf("Error publishing fetch node info: %s", err)
		}
	})
}

func (eb *ClusterEventBus) SubscribeRequestJobWorker(callback func(jobName string)) (Subscription, error) {
	return eb.queueSubscribe(EventStartJobWorker, fmt.Sprintf("%s.workers", EventStartJobWorker), func(msg *nats.Msg) {
		var sjw startJobWorker
		if err := json.Unmarshal(msg.Data, &sjw); err != nil {
			log.Errorf("Could not unmarshal start job worker: %s", err)
			return
		}
		callback(sjw.Name)
		// Respond with empty reply
		msg.Respond([]byte{})
	})
}

func (eb *ClusterEventBus) RequestJobWorkers(name string, n int, timeout time.Duration) error {
	for i := 0; i < n; i++ {
		if _, err := eb.request(EventStartJobWorker, util.MustJsonByteSlice(startJobWorker{name}), timeout); err != nil {
			return err
		}
	}
	return nil
}

func (eb *ClusterEventBus) RestartApp(appName string) error {
	return eb.publish(EventRestartApp, util.MustJsonByteSlice(restartApp{appName}))
}

func (eb *ClusterEventBus) SubscribeRestartApp(callback func(appName string)) (Subscription, error) {
	return eb.subscribe(EventRestartApp, func(msg *nats.Msg) {
		var restartEvent restartApp
		if err := json.Unmarshal(msg.Data, &restartEvent); err != nil {
			log.Errorf("Could not decode restart app message: %s", err)
			return
		}
		callback(restartEvent.Name)
	})
}

func (eb *ClusterEventBus) PublishEvent(name string, event interface{}) error {
	return eb.publish(SafeNATSSubject(name), util.MustJsonByteSlice(publishEvent{
		Name: name,
		Data: event,
	}))
}

func (eb *ClusterEventBus) SubscribeEvent(pattern string, callback func(name string, data interface{}, msg *nats.Msg)) (Subscription, error) {
	return eb.subscribe(SafeNATSSubject(pattern), func(msg *nats.Msg) {
		var eventData publishEvent
		if err := json.Unmarshal(msg.Data, &eventData); err != nil {
			log.Errorf("Could not unmarshal event: %s - %s", err, string(msg.Data))
			return
		}
		callback(eventData.Name, eventData.Data, msg)
	})
}

func (eb *ClusterEventBus) QueueSubscribeEvent(pattern string, callback func(name string, data interface{}, msg *nats.Msg)) (Subscription, error) {
	return eb.queueSubscribe(SafeNATSSubject(pattern), fmt.Sprintf("%s.workers", SafeNATSSubject(pattern)), func(msg *nats.Msg) {
		var eventData publishEvent
		if err := json.Unmarshal(msg.Data, &eventData); err != nil {
			log.Errorf("Could not unmarshal event: %s - %s", err, string(msg.Data))
			return
		}
		callback(eventData.Name, eventData.Data, msg)
	})
}

//func (eb *ClusterEventBus) SubscribeEvent(pattern string, callback func(name string, data interface{}, msg *nats.Msg)) (Subscription, error) {
//	return eb.subscribe(SafeNATSSubject(pattern), func(msg *nats.Msg) {
//		var eventData publishEvent
//		if err := json.Unmarshal(msg.Data, &eventData); err != nil {
//			log.Errorf("Could not unmarshal event: %s - %s", err, string(msg.Data))
//			return
//		}
//		callback(eventData.Name, eventData.Data, msg)
//	})
//}

func (eb *ClusterEventBus) RequestEvent(name string, event interface{}, timeout time.Duration) (*nats.Msg, error) {
	return eb.request(SafeNATSSubject(name), util.MustJsonByteSlice(publishEvent{
		Name: name,
		Data: event,
	}), timeout)
}
