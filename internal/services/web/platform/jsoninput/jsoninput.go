// Package jsoninput provides strict JSON body decoding for web handlers.
package jsoninput

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// DecodeStrict decodes one JSON object with size and trailing-token guards.
func DecodeStrict(r *http.Request, target any, maxBytes int64) error {
	if r == nil || r.Body == nil {
		return io.ErrUnexpectedEOF
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBytes+1))
	if err != nil {
		return err
	}
	if len(body) == 0 || int64(len(body)) > maxBytes {
		return io.ErrUnexpectedEOF
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return io.ErrUnexpectedEOF
	}
	return nil
}
