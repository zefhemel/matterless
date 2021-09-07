package cluster

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

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
			log.Debug("Booting nats server")
			p, err := strconv.Atoi(parsedUrl.Port())
			if err != nil {
				return nil, errors.Wrap(err, "parsing port")
			}
			err = spawnNatsServer(p)
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
