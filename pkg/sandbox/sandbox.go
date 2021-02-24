package sandbox

type Sandbox interface {
	Invoke(event interface{}, code string, env map[string]string) (interface{}, []string, error)
}
