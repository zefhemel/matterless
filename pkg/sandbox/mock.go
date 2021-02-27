package sandbox

type MockSandbox struct {
	result interface{}
	logs   string
	err    error
}

func NewMockSandbox(result interface{}, logs string, err error) *MockSandbox {
	return &MockSandbox{
		result: result,
		logs:   logs,
		err:    err,
	}
}

func (sandbox *MockSandbox) Invoke(event interface{}, code string, env map[string]string) (interface{}, string, error) {
	return sandbox.result, sandbox.logs, sandbox.err
}

var _ Sandbox = &MockSandbox{}
