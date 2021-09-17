package cluster

import (
	"regexp"
)

const (
	EventRestartApp     = "$restart"
	EventFetchNodeInfo  = "$nodeinfo"
	EventStartJobWorker = "$startjob"
)

type NodeID = uint64

type restartApp struct {
	Name string `json:"name"`
}

type publishEvent struct {
	Name string      `json:"name"`
	Data interface{} `json:"data"`
}

type functionInvoke struct {
	Data interface{} `json:"data"`
}

type functionResult struct {
	IsError bool        `json:"is_error,omitempty"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type startJobWorker struct {
	Name string `json:"name"`
}

type startFunctionWorker struct {
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

type logMessage struct {
	Message  string `json:"message"`
	Function string `json:"function"`
}

var safeSubjectRE = regexp.MustCompile("[^A-Za-z0-9_\\*\\.>]")

// SafeNATSSubject turns an event name into a safe to use NATS subject
// TODO: add safer handling of e.g. colons
func SafeNATSSubject(eventName string) string {
	return safeSubjectRE.ReplaceAllString(eventName, "_")
}
