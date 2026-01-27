// Package random provides cryptographic seed generation helpers.
//
// It uses crypto/rand to generate high-entropy seeds suitable for
// initializing pseudo-random number generators in deterministic systems.
package random

import (
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
)

// NewSeed generates a random seed using crypto/rand.
func NewSeed() (int64, error) {
	var b [8]byte
	if _, err := crand.Read(b[:]); err != nil {
		return 0, fmt.Errorf("read random seed: %w", err)
	}

	return int64(binary.LittleEndian.Uint64(b[:])), nil
}
