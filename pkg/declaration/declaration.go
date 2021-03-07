package declaration

type Declarations struct {
	Sources       map[string]*SourceDef
	Functions     map[string]*FunctionDef
	Subscriptions map[string]*SubscriptionDef
	Environment   map[string]string
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
}

type SourceDef struct {
	Type  string
	URL   string
	Token string
	// TODO: Add Username and Password
}
