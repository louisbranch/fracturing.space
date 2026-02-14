package random

import (
	"errors"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

func TestResolveSeedDefaultsToServerSeed(t *testing.T) {
	seed, source, mode, err := ResolveSeed(nil, func() (int64, error) {
		return 123, nil
	}, nil)
	if err != nil {
		t.Fatalf("ResolveSeed returned error: %v", err)
	}
	if seed != 123 {
		t.Fatalf("seed = %d, want 123", seed)
	}
	if source != SeedSourceServer {
		t.Fatalf("seed source = %q, want %q", source, SeedSourceServer)
	}
	if mode != commonv1.RollMode_LIVE {
		t.Fatalf("roll mode = %v, want %v", mode, commonv1.RollMode_LIVE)
	}
}

func TestResolveSeedUsesClientSeedWhenAllowed(t *testing.T) {
	seedValue := uint64(77)
	seed, source, mode, err := ResolveSeed(&commonv1.RngRequest{
		Seed:     &seedValue,
		RollMode: commonv1.RollMode_REPLAY,
	}, func() (int64, error) {
		return 123, nil
	}, func(mode commonv1.RollMode) bool {
		return mode == commonv1.RollMode_REPLAY
	})
	if err != nil {
		t.Fatalf("ResolveSeed returned error: %v", err)
	}
	if seed != int64(seedValue) {
		t.Fatalf("seed = %d, want %d", seed, seedValue)
	}
	if source != SeedSourceClient {
		t.Fatalf("seed source = %q, want %q", source, SeedSourceClient)
	}
	if mode != commonv1.RollMode_REPLAY {
		t.Fatalf("roll mode = %v, want %v", mode, commonv1.RollMode_REPLAY)
	}
}

func TestResolveSeedIgnoresClientSeedWhenDisallowed(t *testing.T) {
	seedValue := uint64(77)
	seed, source, mode, err := ResolveSeed(&commonv1.RngRequest{
		Seed:     &seedValue,
		RollMode: commonv1.RollMode_LIVE,
	}, func() (int64, error) {
		return 555, nil
	}, func(mode commonv1.RollMode) bool {
		return mode == commonv1.RollMode_REPLAY
	})
	if err != nil {
		t.Fatalf("ResolveSeed returned error: %v", err)
	}
	if seed != 555 {
		t.Fatalf("seed = %d, want 555", seed)
	}
	if source != SeedSourceServer {
		t.Fatalf("seed source = %q, want %q", source, SeedSourceServer)
	}
	if mode != commonv1.RollMode_LIVE {
		t.Fatalf("roll mode = %v, want %v", mode, commonv1.RollMode_LIVE)
	}
}

func TestResolveSeedRejectsOutOfRangeSeed(t *testing.T) {
	seedValue := uint64(maxSeedInt64) + 1
	_, _, _, err := ResolveSeed(&commonv1.RngRequest{
		Seed:     &seedValue,
		RollMode: commonv1.RollMode_REPLAY,
	}, func() (int64, error) {
		return 123, nil
	}, func(mode commonv1.RollMode) bool {
		return mode == commonv1.RollMode_REPLAY
	})
	if !errors.Is(err, ErrSeedOutOfRange()) {
		t.Fatalf("ResolveSeed error = %v, want %v", err, ErrSeedOutOfRange())
	}
}
