package cluster

// New app deployments are signaled through the cluster data
// store through puts and deletes on the app keys

// In addition

const (
	EventRestartApp   = "restart"
	EventPublishEvent = "event"
	EventStartJob     = "job"
)

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
