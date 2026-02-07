// Package random provides cryptographic seed generation helpers.
//
// It uses crypto/rand to generate high-entropy seeds suitable for
// initializing pseudo-random number generators in deterministic systems.
package random

import (
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

const (
	// RngAlgoMathRandV1 identifies the math/rand RNG algorithm version.
	RngAlgoMathRandV1 = "math-rand-v1"
	// SeedSourceClient indicates a client-supplied seed was used.
	SeedSourceClient = "CLIENT"
	// SeedSourceServer indicates a server-generated seed was used.
	SeedSourceServer = "SERVER"
)

const maxSeedInt64 = int64(^uint64(0) >> 1)

var errSeedOutOfRange = errors.New("seed must fit in int64")

// ErrSeedOutOfRange reports when a seed does not fit in int64.
func ErrSeedOutOfRange() error {
	return errSeedOutOfRange
}

// NewSeed generates a random, non-negative seed using crypto/rand.
func NewSeed() (int64, error) {
	var b [8]byte
	if _, err := crand.Read(b[:]); err != nil {
		return 0, fmt.Errorf("read random seed: %w", err)
	}

	seed := binary.LittleEndian.Uint64(b[:]) & uint64(^uint64(0)>>1)
	return int64(seed), nil
}

// ResolveSeed determines the seed, seed source, and roll mode for a request.
func ResolveSeed(rng *commonv1.RngRequest, seedFunc func() (int64, error), allowClientSeed func(commonv1.RollMode) bool) (int64, string, commonv1.RollMode, error) {
	rollMode := commonv1.RollMode_LIVE
	if rng != nil && rng.GetRollMode() != commonv1.RollMode_ROLL_MODE_UNSPECIFIED {
		rollMode = rng.GetRollMode()
	}

	seedSource := SeedSourceServer
	seed := int64(0)
	seedProvided := rng != nil && rng.Seed != nil
	if seedProvided {
		seedValue := rng.GetSeed()
		if seedValue > uint64(maxSeedInt64) {
			return 0, "", rollMode, errSeedOutOfRange
		}
		if allowClientSeed != nil && allowClientSeed(rollMode) {
			seed = int64(seedValue)
			seedSource = SeedSourceClient
		}
	}
	if seedSource == SeedSourceServer {
		generated, err := seedFunc()
		if err != nil {
			return 0, "", rollMode, err
		}
		seed = generated
	}
	return seed, seedSource, rollMode, nil
}
