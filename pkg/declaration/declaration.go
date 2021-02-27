package declaration

type Declarations struct {
	Sources       map[string]SourceDef
	Functions     map[string]FunctionDef
	Subscriptions map[string]SubscriptionDef
}

type FunctionDef struct {
	Name     string
	Language string
	Code     string
	Debug    bool
}

type SubscriptionDef struct {
	Source     string
	Function   string
	EventTypes []string
	Channel    string
}

type SourceDef struct {
	URL   string
	Token string
}
