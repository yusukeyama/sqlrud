package sqlrud

// queryOptions holds per-call options for Insert, Update, and Delete.
type queryOptions struct {
	whereColumns   []string
	excludeColumns []string
}

// QueryOption is a functional option applied to a single Insert/Update/Delete
// call.
type QueryOption func(*queryOptions)

// WhereColumns specifies which struct fields (by their db tag column names) are
// used in the WHERE clause of an UPDATE or DELETE statement.
func WhereColumns(columns ...string) QueryOption {
	return func(o *queryOptions) {
		o.whereColumns = columns
	}
}

// ExcludeColumns specifies struct fields that should be omitted from the
// generated INSERT or UPDATE statement.
func ExcludeColumns(columns ...string) QueryOption {
	return func(o *queryOptions) {
		o.excludeColumns = columns
	}
}

func applyQueryOptions(opts []QueryOption) queryOptions {
	o := queryOptions{}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// Option is a functional option applied when constructing a CRUD instance via
// New.
type Option func(*CRUD)

// WithDialect sets the Dialect used for identifier quoting and error detection.
func WithDialect(d Dialect) Option {
	return func(c *CRUD) {
		c.dialect = d
	}
}

// WithMySQLDialect is a convenience option that sets MySQLDialect.
func WithMySQLDialect() Option {
	return WithDialect(MySQLDialect{})
}

// WithDefaultInsertExcludeColumns sets columns that are excluded from every
// INSERT statement unless overridden per-call.
func WithDefaultInsertExcludeColumns(columns ...string) Option {
	return func(c *CRUD) {
		c.config.DefaultInsertExcludeColumns = columns
	}
}

// WithDefaultUpdateExcludeColumns sets columns that are excluded from every
// UPDATE statement unless overridden per-call.
func WithDefaultUpdateExcludeColumns(columns ...string) Option {
	return func(c *CRUD) {
		c.config.DefaultUpdateExcludeColumns = columns
	}
}

// WithAutoIncrementTag overrides the struct tag option name used to mark
// auto-increment primary key fields (default: "auto_increment").
func WithAutoIncrementTag(tag string) Option {
	return func(c *CRUD) {
		c.config.AutoIncrementTag = tag
	}
}
