package sqlrud

// InsertOption configures an Insert call.
type InsertOption func(*insertConfig)

type insertConfig struct {
	ignore bool
}

// WithIgnore adds INSERT IGNORE (MySQL) / INSERT OR IGNORE (SQLite) semantics.
func WithIgnore() InsertOption {
	return func(c *insertConfig) { c.ignore = true }
}
