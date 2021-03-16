package definition

type APIGatewayRequestEvent struct {
	Path          string            `json:"path"`
	Method        string            `json:"method"`
	Headers       map[string]string `json:"headers"`
	FormValues    map[string]string `json:"form_values"`
	RequestParams map[string]string `json:"request_params"`
	JSONBody      interface{}       `json:"json_body"`
}

type APIGatewayResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    interface{}       `json:"body"`
}
