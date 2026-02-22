package projection

import (
	"encoding/json"
	"fmt"
)

// decodePayload is a guarded bridge between event envelopes and in-memory domain
// payload types, preserving a clear failure message when replay/apply input is
// malformed.
func decodePayload(payload []byte, target any, name string) error {
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("decode %s payload: %w", name, err)
	}
	return nil
}
