// Package main provides a one-shot utility for join-grant key generation.
//
// It emits the asymmetric keypair used by auth invitation flow checks.
package main

import (
	"os"

	"github.com/louisbranch/fracturing.space/internal/platform/config"
	"github.com/louisbranch/fracturing.space/internal/tools/joingrant"
)

func main() {
	if err := joingrant.Run(os.Stdout, nil); err != nil {
		config.Exitf("generate join grant key: %v", err)
	}
}
