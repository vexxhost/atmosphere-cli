package atmosphere

import (
	"context"
	"errors"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// contextKey is an unexported type for context keys to prevent collisions
type contextKey int

// Context key constants using iota for better performance and type safety
const (
	configFlagsKey contextKey = iota
)

var (
	// ErrNoConfigFlags is returned when config flags are not found in context
	ErrNoConfigFlags = errors.New("kubernetes config flags not found in context")
)

// Context wraps the standard context with atmosphere-specific values
type Context struct {
	context.Context
}

// New creates a new atmosphere context with the given configuration
func New(parent context.Context, configFlags *genericclioptions.ConfigFlags) Context {
	ctx := context.WithValue(parent, configFlagsKey, configFlags)
	return Context{ctx}
}

// WithConfigFlags returns a new context with the config flags set
func WithConfigFlags(parent context.Context, configFlags *genericclioptions.ConfigFlags) context.Context {
	return context.WithValue(parent, configFlagsKey, configFlags)
}

// ConfigFlags returns the Kubernetes config flags from the context
func ConfigFlags(ctx context.Context) (*genericclioptions.ConfigFlags, error) {
	v := ctx.Value(configFlagsKey)
	if v == nil {
		return nil, ErrNoConfigFlags
	}
	flags, ok := v.(*genericclioptions.ConfigFlags)
	if !ok {
		return nil, ErrNoConfigFlags
	}
	return flags, nil
}

// MustConfigFlags returns the config flags or panics if not found
func MustConfigFlags(ctx context.Context) *genericclioptions.ConfigFlags {
	flags, err := ConfigFlags(ctx)
	if err != nil {
		panic(err)
	}
	return flags
}

// Convenience method on the wrapped context
func (c Context) ConfigFlags() (*genericclioptions.ConfigFlags, error) {
	return ConfigFlags(c.Context)
}