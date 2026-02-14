package joingrant

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// Run generates a join grant key pair and writes exports.
func Run(out io.Writer, reader io.Reader) error {
	if out == nil {
		return errors.New("output is required")
	}
	if reader == nil {
		reader = rand.Reader
	}
	publicKey, privateKey, err := ed25519.GenerateKey(reader)
	if err != nil {
		return fmt.Errorf("generate join grant key: %w", err)
	}
	if _, err := fmt.Fprintf(out, "export FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY=%s\n", base64.RawStdEncoding.EncodeToString(privateKey)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "export FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY=%s\n", base64.RawStdEncoding.EncodeToString(publicKey)); err != nil {
		return err
	}
	return nil
}
