package generator

import (
	"fmt"
	"math/rand"
	"os"
	"time"
)

// NewSeededRNG creates a seeded random number generator.
// If seed is 0, uses current time and prints the seed for reproducibility.
func NewSeededRNG(seed int64, verbose bool) *rand.Rand {
	if seed == 0 {
		seed = time.Now().UnixNano()
		if verbose {
			fmt.Fprintf(os.Stderr, "Using seed: %d\n", seed)
		}
	}
	return rand.New(rand.NewSource(seed))
}
