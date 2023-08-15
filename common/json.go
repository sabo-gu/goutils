package common

import "encoding/json"

func ToJsonString(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
