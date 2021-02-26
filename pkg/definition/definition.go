package definition

type Definitions struct {
	Identities    map[string]IdentityDef
	Functions     map[string]FunctionDef
	Subscriptions []SubscriptionDef
}

type FunctionDef struct {
	Name     string
	Language string
	Code     string
	Debug    bool
}

type SubscriptionDef struct {
	EventTypes      []string
	Channel         string
	TriggerFunction string
	Identity        string
}

type IdentityDef struct {
	Token string
	URL   string // TODO: at some point
	WSURL string // TODO: at some point
}
