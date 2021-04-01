package application

import (
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/store"
	"testing"
)

func TestInterpolate(t *testing.T) {
	log.Info(interPolateStoreValues(store.MockStore{}, "Hello there ${config.bla}", func(s string) {

	}))
}
