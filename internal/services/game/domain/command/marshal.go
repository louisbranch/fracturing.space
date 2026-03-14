package command

import (
	"encoding/json"
	"fmt"
)

// MustMarshalJSON marshals v and panics on error. Use for struct types
// where marshal cannot fail (no channel/func fields, no cyclic references).
//
// Convention: across all deciders, `_, _ = json.Marshal(...)` is used for
// known-safe payload structs. This helper makes the safety guarantee explicit
// for callers that prefer a panic over a silent discard.
func MustMarshalJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("command: marshal should not fail for %T: %v", v, err))
	}
	return data
}
