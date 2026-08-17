package truenas

// JSON-RPC helper that accepts a typed params shape so per-feature files
// don't need to spell out []any{} everywhere.

// AsParams converts a typed object to the positional []any slice a JSON-RPC
// method expects. Filters must be [][]any (already in TRUENAS format).
func AsParams(args ...any) []any { return args }

// Row is one element of a list-style result. Caller casts to the right type.
type Row = map[string]any
