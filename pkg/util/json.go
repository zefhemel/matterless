package util

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
)

func MustJsonString(v interface{}) string {
	return string(MustJsonByteSlice(v))
}

func MustJsonByteSlice(v interface{}) []byte {
	buf, err := json.Marshal(v)
	if err != nil {
		log.Errorf("JSON serialization error: %s", err)
	}
	return buf
}
