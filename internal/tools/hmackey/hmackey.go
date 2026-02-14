package hmackey

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
)

// Config holds configuration for HMAC key generation.
type Config struct {
	Bytes int
}

// ParseConfig parses flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	cfg := Config{Bytes: 32}
	fs.IntVar(&cfg.Bytes, "bytes", cfg.Bytes, "number of random bytes (default: 32)")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run generates the key and writes it to out.
func Run(cfg Config, out io.Writer, reader io.Reader) error {
	if cfg.Bytes <= 0 {
		return errors.New("bytes must be greater than zero")
	}
	if out == nil {
		return errors.New("output is required")
	}
	if reader == nil {
		reader = rand.Reader
	}

	buf := make([]byte, cfg.Bytes)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return fmt.Errorf("generate random bytes: %w", err)
	}
	_, err := fmt.Fprintf(out, "FRACTURING_SPACE_GAME_EVENT_HMAC_KEY=%s\n", hex.EncodeToString(buf))
	return err
}
