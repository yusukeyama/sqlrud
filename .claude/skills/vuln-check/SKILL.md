---
name: vuln-check
description: Security review of sqlrud focused on SQL injection through identifiers, unsafe mutations, and dependency CVEs. Use when asked to check this repository for vulnerabilities, before a release, or when reviewing changes to query building (query.go, client.go, metadata.go).
---

# sqlrud vulnerability check

sqlrud binds values with placeholders, but builds identifiers (table names,
column names, operators, ORDER BY direction, upsert clauses) with
`fmt.Sprintf` and string concatenation. Most real risk lives there.

## Scope

- No argument: review the whole package.
- A branch, PR, or "diff": review only `git diff origin/main...HEAD`, but
  follow any changed function into its callers.

## Steps

1. Run the automated tools and keep their output for the report:
   - `go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...`
   - `go run github.com/securego/gosec/v2/cmd/gosec@v2.29.0 ./...`
     (gosec needs Go 1.26 or earlier; with a newer local Go, prefix
     `GOTOOLCHAIN=go1.26.9`)
   - `go vet ./...` and `go test -race ./...`

   If a tool cannot run (for example the vulnerability DB is unreachable),
   say so in the report instead of skipping silently.
2. List every place SQL text is assembled:
   `grep -n 'Sprintf\|+ "\|strings.Join' *.go`. For each one, write down
   which parts are identifiers and where each identifier comes from.
3. Walk the checklist below. For every item, trace from a public API
   (`Find`, `First`, `Create`, `Update`, `CreateOrUpdate`, `Delete`,
   `Query`, `Where`, `OrderBy`, `Limit`, `Offset`, `TableName()`) to the
   SQL string. Do not report an issue without that trace.
4. If you suspect a bug, prove it with a test using `go-sqlmock` (see
   `client_test.go` for the existing style) and run it. Do not commit the
   test unless asked.

## Checklist

### Identifier injection
- `Where(field, ...)` and `OrderBy(field, ...)`: is `field` resolved only
  against the model's known columns, or can an arbitrary string reach the
  query? Check the error path returns `ErrInvalidIdentifier`.
- `OrderBy` direction: limited to `Asc` / `Desc`?
- `TableName()` and `db` tags: these come from the library user's code,
  not end users, but check whether they are validated or quoted and note
  the behavior if a caller passes untrusted input into a dynamic
  `TableName()`.
- Default table name from `toSnakeCase`: can a struct name produce
  unexpected characters?

### Operators and predicates
- `Predicate.operator` is unexported; confirm no path lets a caller set
  an arbitrary operator (including a zero-value `Predicate{}`).
- `In` / `NotIn` with zero values: does it produce invalid SQL like
  `IN ()`, or worse, a condition that matches every row?
- `expandValues`: are nested slices, `[]byte`, and `nil` handled safely?

### Mutations
- `Update` / `Delete` with no options and a zero primary key: can they
  produce an unbounded `UPDATE ... ` / `DELETE FROM ...` that affects every
  row? Check `validateMutation`.
- Composite primary keys: are all key columns included in the WHERE?
- `CreateOrUpdate` upserts (`ON CONFLICT`, `ON DUPLICATE KEY UPDATE`): are
  the column lists derived only from model metadata?

### Pagination
- `Limit` / `Offset`: negative or huge values rejected? Bound as
  placeholders, not formatted into the string?

### Raw queries
- `Query(ctx, query, args...)`: the API itself is a pass-through. Check
  that the README and doc comment tell callers to use placeholders and
  never concatenate input.

### Placeholder rebinding
- `rebind` / `dialectForDriver`: for unknown drivers, is the fallback
  bindvar style correct, so values are never mis-bound?

### Transactions
- `TransactionOptions`: rollback on panic and on error, no swallowed
  commit errors.

### Dependencies
- Rely on `govulncheck`; only report vulnerabilities whose symbols are
  actually reachable.

## Report format

Reply in the user's language. Start with a one-line verdict, then a table:

| Severity | Location | Issue | Trace | Fix |
|---|---|---|---|---|

Severity: Critical (attacker-controlled SQL or mass update/delete),
High, Medium, Low, Info. Use `file:line` for locations. List items you
checked and found safe in a short "Checked, no issue" section so the
reader can see coverage. Finish with the automated tool results.
