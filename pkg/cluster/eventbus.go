package cluster

import (
	"encoding/json"
	"fmt"
	"net/url"
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
func ConnectOrBoot(natsUrl string, options ...nats.Option) (*nats.Conn, error) {
	nc, err := nats.Connect(natsUrl, options...)
	if err != nil {
		parsedUrl, err := url.Parse(natsUrl)
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
			err = spawnNatsServer(p)
			if err != nil {
				return nil, errors.Wrap(err, "nats spawn")
			}
		}
		nc, err = nats.Connect(natsUrl, options...)
		if err != nil {
			return nil, errors.Wrap(err, "after nats server boot")
		}
	}
	return nc, nil
}

func spawnNatsServer(port int) error {
	opts := &server.Options{
		Host:                  "0.0.0.0",
		Port:                  port,
		MaxControlLine:        4096,
		DisableShortFirstPing: true,

		// Jetstream
		JetStream: true,
		StoreDir:  "js-data",
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

func (eb *ClusterEventBus) Publish(name string, data []byte) error {
	return eb.conn.Publish(fmt.Sprintf("%s.%s", eb.prefix, name), data)
}

func (eb *ClusterEventBus) Request(name string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	return eb.conn.Request(fmt.Sprintf("%s.%s", eb.prefix, name), data, timeout)
}

func (eb *ClusterEventBus) Subscribe(name string, callback func(msg *nats.Msg)) (Subscription, error) {
	return eb.conn.Subscribe(fmt.Sprintf("%s.%s", eb.prefix, name), callback)
}

func (eb *ClusterEventBus) QueueSubscribe(name string, queue string, callback func(msg *nats.Msg)) (Subscription, error) {
	return eb.conn.QueueSubscribe(fmt.Sprintf("%s.%s", eb.prefix, name), fmt.Sprintf("%s.%s", eb.prefix, queue), callback)
}

func (eb *ClusterEventBus) InvokeFunction(name string, event interface{}) (interface{}, error) {
	resp, err := eb.Request(fmt.Sprintf("function.%s", name), util.MustJsonByteSlice(FunctionInvoke{
		Data: event,
	}), 10*time.Second)
	if err != nil {
		return nil, err
	}
	var respMsg FunctionResult
	if err := json.Unmarshal(resp.Data, &respMsg); err != nil {
		return nil, err
	}
	if respMsg.IsError {
		return nil, errors.New(respMsg.Error)
	}
	return respMsg.Data, nil
}

func (eb *ClusterEventBus) SubscribeInvokeFunction(name string, callback func(interface{}) (interface{}, error)) (Subscription, error) {
	return eb.QueueSubscribe(fmt.Sprintf("function.%s", name), fmt.Sprintf("function.%s.workers", name), func(msg *nats.Msg) {
		var requestMessage FunctionInvoke
		if err := json.Unmarshal(msg.Data, &requestMessage); err != nil {
			log.Errorf("Could not unmarshal event data: %s", err)
			err = msg.Respond([]byte(util.MustJsonByteSlice(FunctionResult{
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
			if err := msg.Respond([]byte(util.MustJsonByteSlice(FunctionResult{
				IsError: true,
				Error:   err.Error(),
			}))); err != nil {
				log.Errorf("Could not respond with error message: %s", err)
			}
			return
		}
		if err := msg.Respond([]byte(util.MustJsonByteSlice(FunctionResult{
			Data: resp,
		}))); err != nil {
			log.Errorf("Could not respond with response: %s", err)
		}
	})
}

func (eb *ClusterEventBus) FetchClusterInfo(wait time.Duration) (*ClusterInfo, error) {
	// TODO: Generate unique ID some other way
	responseSubject := fmt.Sprintf("clusterinfo.%s", strings.ReplaceAll(uuid.NewString(), "-", ""))

	ci := &ClusterInfo{
		Nodes: map[uint64]*NodeInfo{},
	}
	var mutex sync.Mutex

	sub, err := eb.Subscribe(responseSubject, func(msg *nats.Msg) {
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
	if err := eb.Publish(EventFetchNodeInfo, util.MustJsonByteSlice(FetchNodeInfo{
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
	return eb.Subscribe(EventFetchNodeInfo, func(msg *nats.Msg) {
		var fni FetchNodeInfo
		if err := json.Unmarshal(msg.Data, &fni); err != nil {
			log.Errorf("Could not unmarshal fetch node info: %s", err)
			return
		}
		if err := eb.Publish(fni.ReplyTo, util.MustJsonByteSlice(callback())); err != nil {
			log.Errorf("Error publishing fetch node info: %s", err)
		}
	})
}

func (eb *ClusterEventBus) SubscribeRequestJobWorker(callback func(jobName string)) (Subscription, error) {
	return eb.QueueSubscribe(EventStartJobWorker, fmt.Sprintf("%s.workers", EventStartJobWorker), func(msg *nats.Msg) {
		var sjw StartJobWorker
		if err := json.Unmarshal(msg.Data, &sjw); err != nil {
			log.Errorf("Could not unmarshal start job worker: %s", err)
			return
		}
		callback(sjw.Name)
	})
}

func (eb *ClusterEventBus) RequestJobWorkers(name string, n int) error {
	for i := 0; i < n; i++ {
		if err := eb.Publish(EventStartJobWorker, util.MustJsonByteSlice(StartJobWorker{name})); err != nil {
			return err
		}
	}
	return nil
}
