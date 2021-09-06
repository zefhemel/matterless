package cluster

import (
	"net/url"
	"strconv"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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
