package cluster

const (
	EventPublishApp     = "app.publish"
	EventDeleteApp      = "app.delete"
	EventRestartApp     = "app.restart"
	EventPublishEvent   = "event"
	EventInvokeFunction = "invoke"
)

type PublishApp struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type DeleteApp struct {
	Name string `json:"name"`
}

type RestartApp struct {
	Name string `json:"name"`
}

type PublishEvent struct {
	Name string      `json:"name"`
	Data interface{} `json:"data"`
}

type FunctionInvoke struct {
	Data interface{} `json:"data"`
}

type FunctionResult struct {
	IsError bool        `json:"is_error,omitempty"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
