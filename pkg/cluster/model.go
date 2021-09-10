package cluster

const (
	EventRestartApp          = "restart"
	EventPublishEvent        = "event"
	EventFetchNodeInfo       = "nodeinfo"
	EventStartJobWorker      = "startjob"
	EventStartFunctionWorker = "startfunction"
)

type NodeID = uint64

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

type StartJobWorker struct {
	Name string `json:"name"`
}

type StartFunctionWorker struct {
	Name string `yaml:"name"`
}

type FetchNodeInfo struct {
	ReplyTo string `json:"reply_to"`
}

type ClusterInfo struct {
	Nodes map[NodeID]*NodeInfo
}

type NodeInfo struct {
	ID   NodeID
	Apps map[string]*AppInfo
}

type AppInfo struct {
	FunctionWorkers map[string]int
	JobWorkers      map[string]int
}
