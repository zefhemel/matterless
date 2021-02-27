package eventsource

type EventSource interface {
	Start() error
	Events() chan interface{}
	Stop()
}
