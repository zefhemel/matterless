package cluster_test

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/cluster"
)

func TestNatsCluster(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	a := assert.New(t)
	eb, err := cluster.ConnectOrBoot("nats://localhost:4222")
	a.Nil(err)
	defer eb.Close()

}
