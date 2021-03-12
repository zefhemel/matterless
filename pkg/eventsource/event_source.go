package eventsource

type EventSource interface {
	Start() error
	Stop()
}
