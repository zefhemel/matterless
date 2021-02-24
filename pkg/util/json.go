package util

import "encoding/json"

func MustJsonString(v interface{}) string {
	buf, _ := json.Marshal(v)
	return string(buf)
}
