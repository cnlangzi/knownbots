package knownbots

import "time"

// Config holds the options for creating a Validator.
type Config struct {
	Root       string
	Interval   time.Duration
	FailLimit  int
	ClassifyUA bool
}

// Option is a functional option for configuring a Validator.
type Option func(*Config)

// WithRoot sets the bots root directory (containing conf.d and data subdirs).
func WithRoot(dir string) Option {
	return func(c *Config) {
		c.Root = dir
	}
}

// WithSchedulerInterval sets the background scheduler interval.
func WithSchedulerInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.Interval = interval
	}
}

// WithFailLimit sets the limit for failed lookup cache.
func WithFailLimit(limit int) Option {
	return func(c *Config) {
		c.FailLimit = limit
	}
}

// WithClassifyUA enables UA classification for non-bot UAs.
// By default, classifyUA is disabled for performance.
// Enable it to distinguish legitimate browsers from suspicious UAs.
func WithClassifyUA() Option {
	return func(c *Config) {
		c.ClassifyUA = true
	}
}
