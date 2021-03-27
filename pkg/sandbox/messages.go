package sandbox

type InitMessage struct {
	Env     map[string]string      `json:"env"`
	Script  string                 `json:"script"`
	Modules map[string]string      `json:"modules"`
	Config  map[string]interface{} `json:"config"`
}
