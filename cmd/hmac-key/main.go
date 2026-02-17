// Package main provides a one-off utility to generate HMAC secrets.
//
// The command emits generated keys to stdout so operational bootstrap tooling can
// pipe them directly into environment or secret stores.
package main

import (
	"flag"
	"os"

	"github.com/louisbranch/fracturing.space/internal/platform/config"
	"github.com/louisbranch/fracturing.space/internal/tools/hmackey"
)

func main() {
	cfg, err := hmackey.ParseConfig(flag.CommandLine, os.Args[1:])
	if err != nil {
		config.Exitf("parse flags: %v", err)
	}
	if err := hmackey.Run(cfg, os.Stdout, nil); err != nil {
		config.Exitf("generate key: %v", err)
	}
}
