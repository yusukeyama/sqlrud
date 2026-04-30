package sqlrud

// InsertOption configures an Insert call.
type InsertOption func(*insertConfig)

type insertConfig struct {
	ignore bool
}

// WithIgnore adds INSERT IGNORE semantics. This is MySQL / MariaDB-specific
// syntax and will not work with PostgreSQL. For PostgreSQL, use a raw query
// with ON CONFLICT DO NOTHING instead.
func WithIgnore() InsertOption {
	return func(c *insertConfig) { c.ignore = true }
}
