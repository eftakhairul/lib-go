package config

type options struct {
	envPrefix string
}

// Option configures Load.
type Option func(*options)

// WithEnvPrefix prepends prefix + "_" to every auto-derived (non-explicit
// `env` tag) environment variable name Load looks up.
func WithEnvPrefix(prefix string) Option {
	return func(o *options) {
		o.envPrefix = prefix
	}
}
