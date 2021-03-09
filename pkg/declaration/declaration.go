package declaration

import "strings"

type Declarations struct {
	Sources       map[string]*SourceDef
	Functions     map[string]*FunctionDef
	Subscriptions map[string]*SubscriptionDef
	Environment   map[string]string
	// For now we just support the empty library name
	Libraries map[string]*FunctionDef
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

// CompileFunctionCode appends all library code to a function to be eval'ed
func (decls *Declarations) CompileFunctionCode(code string) string {
	codeParts := []string{code}
	for _, libDef := range decls.Libraries {
		codeParts = append(codeParts, libDef.Code)
	}
	return strings.Join(codeParts, "\n\n")
}
