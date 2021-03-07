package declaration

import (
	"github.com/buildkite/interpolate"
)

// Normalize replaces environment variables with their values
func (decls *Declarations) Normalize() {
	mapEnv := interpolate.NewMapEnv(decls.Environment)
	for _, sourceDecl := range decls.Sources {
		sourceDecl.Token, _ = interpolate.Interpolate(mapEnv, sourceDecl.Token)
		sourceDecl.URL, _ = interpolate.Interpolate(mapEnv, sourceDecl.URL)
	}
}
