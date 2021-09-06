package util

import "encoding/json"

func MustJsonString(v interface{}) string {
	return string(MustJsonByteSlice(v))
}

func MustJsonByteSlice(v interface{}) []byte {
	buf, _ := json.Marshal(v)
	return buf
}
