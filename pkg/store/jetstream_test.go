package store_test

import (
	"fmt"
	"github.com/zefhemel/matterless/pkg/config"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/cluster"
	"github.com/zefhemel/matterless/pkg/store"

	"github.com/stretchr/testify/assert"
)

func TestJetstreamStore(t *testing.T) {
	jetStreamStores := []*store.JetstreamStore{}
	var timeout = 5 * time.Second

	// Create a bunch of JetStream stores
	for i := 0; i < 5; i++ {
		conn, err := cluster.ConnectOrBoot(&config.Config{
			DataDir:        "nats-data",
			ClusterNatsUrl: "nats://localhost:4222",
		})
		assert.Nil(t, err)
		levelDBStore, err := store.NewLevelDBStore(fmt.Sprintf("lvldb-%d", i))
		assert.Nil(t, err)
		defer levelDBStore.DeleteStore()
		jss, err := store.NewJetstreamStore(conn, "test", levelDBStore)
		assert.Nil(t, err)
		assert.NotNil(t, jss)
		jetStreamStores = append(jetStreamStores, jss)
	}

	// Connect a bunch of them
	assert.Nil(t, jetStreamStores[0].Connect(timeout))
	defer jetStreamStores[0].Disconnect()
	assert.Nil(t, jetStreamStores[1].Connect(timeout))
	defer jetStreamStores[1].Disconnect()
	assert.Nil(t, jetStreamStores[2].Connect(timeout))
	defer jetStreamStores[2].Disconnect()
	assert.Nil(t, jetStreamStores[3].Connect(timeout))
	defer jetStreamStores[3].Disconnect()

	// Clean up stream later
	defer jetStreamStores[0].DeleteStore()

	// Basic put-get
	assert.Nil(t, jetStreamStores[0].Put("name", "Pete"))

	v, err := jetStreamStores[0].Get("name")
	assert.Nil(t, err)
	assert.Equal(t, "Pete", v)

	assert.Nil(t, jetStreamStores[2].Sync(timeout))

	v, err = jetStreamStores[2].Get("name")
	assert.Nil(t, err)
	assert.Equal(t, "Pete", v)

	for i := 0; i < 100; i++ {
		assert.Nil(t, jetStreamStores[3].Put(fmt.Sprintf("key%d", i), i))
	}

	// Spot check the result
	assert.Nil(t, jetStreamStores[0].Sync(timeout))
	r, err := jetStreamStores[0].Get("key22")
	assert.Nil(t, err)
	assert.Equal(t, float64(22), r)

	// Late connecter
	log.Info("Now connecting late")
	assert.Nil(t, jetStreamStores[4].Connect(timeout))
	defer jetStreamStores[4].Disconnect()

	v, err = jetStreamStores[4].Get("name")
	assert.Nil(t, err)
	assert.Equal(t, "Pete", v)

	// Disconnect one store and keep updating
	jetStreamStores[0].Disconnect()
	assert.Nil(t, jetStreamStores[2].Put("name", "John"))

	// Affects connected store
	assert.Nil(t, jetStreamStores[1].Sync(timeout))
	v, err = jetStreamStores[1].Get("name")
	assert.Nil(t, err)
	assert.Equal(t, "John", v)

	// Doesn't affect disconnected
	v, err = jetStreamStores[0].Get("name")
	assert.Nil(t, err)
	assert.Equal(t, "Pete", v)

	// Reconnect
	log.Info("Connecting again")
	assert.Nil(t, jetStreamStores[0].Connect(timeout))
	v, err = jetStreamStores[0].Get("name")
	assert.Nil(t, err)
	assert.Equal(t, "John", v)

	// Test delete
	assert.Nil(t, jetStreamStores[2].Put("deleteme", "sup"))
	v, err = jetStreamStores[2].Get("deleteme")
	assert.Nil(t, err)
	assert.Equal(t, "sup", v)

	assert.Nil(t, jetStreamStores[2].Delete("deleteme"))
	assert.Nil(t, jetStreamStores[0].Sync(timeout))
	v, err = jetStreamStores[0].Get("deleteme")
	assert.Nil(t, err)
	assert.Nil(t, v)

	// Test query stuff
	s := jetStreamStores[0]
	s2 := jetStreamStores[3]

	s.Put("person:1", &person{
		Name: "John",
		Age:  20,
	})
	s.Put("person:2", &person{
		Name: "Jane",
		Age:  21,
	})

	assert.Nil(t, s2.Sync(timeout))

	people, err := s2.QueryPrefix("person:")
	assert.NoError(t, err)
	assert.Len(t, people, 2)
	person0 := people[0].Value.(map[string]interface{})
	person1 := people[1].Value.(map[string]interface{})
	assert.Equal(t, "John", person0["name"])
	assert.Equal(t, "Jane", person1["name"])

	people, err = s2.QueryRange("person:", "person:~")
	assert.NoError(t, err)
	assert.Len(t, people, 2)

	// assert.Fail(t, "fail")
}
