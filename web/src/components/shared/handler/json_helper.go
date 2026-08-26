package handler

import "encoding/json"

// jsonMarshal is a tiny helper so we don't import encoding/json at the top
// of every file that calls auditLog. Returns nil on error so the caller
// can fall back to "{}".
func jsonMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}
