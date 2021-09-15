package store_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/zefhemel/matterless/pkg/store"
	"testing"
)

type person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestStore(t *testing.T) {
	s, err := store.NewLevelDBStore("test")
	assert.NoError(t, err)

	s.Put("simpleKey", "simpleValue")
	v, err := s.Get("simpleKey")
	assert.NoError(t, err)
	assert.Equal(t, "simpleValue", v)

	s.Put("person:1", &person{
		Name: "John",
		Age:  20,
	})
	s.Put("person:2", &person{
		Name: "Jane",
		Age:  21,
	})

	people, err := s.QueryPrefix("person:")
	assert.NoError(t, err)
	assert.Len(t, people, 2)
	person0 := people[0].Value.(map[string]interface{})
	person1 := people[1].Value.(map[string]interface{})
	assert.Equal(t, "John", person0["name"])
	assert.Equal(t, "Jane", person1["name"])

	people, err = s.QueryRange("person:", "person:~")
	assert.NoError(t, err)
	assert.Len(t, people, 2)

	defer s.DeleteStore()
}
