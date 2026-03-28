package httpx

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
)

// DecodeJSONStrict decodes one JSON object with size and trailing-token guards.
func DecodeJSONStrict(r *http.Request, target any, maxBytes int64) error {
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

// DecodeJSONStrictInvalidInput decodes one JSON object with strict field/size
// constraints and maps malformed input to the standard web invalid-input error.
func DecodeJSONStrictInvalidInput(r *http.Request, target any, maxBytes int64) error {
	if err := DecodeJSONStrict(r, target, maxBytes); err != nil {
		return apperrors.E(apperrors.KindInvalidInput, "Invalid JSON body.")
	}
	return nil
}

// ParseFormInvalidInput parses form values and maps malformed bodies to the
// standard invalid-input transport error used by web modules.
func ParseFormInvalidInput(r *http.Request, localizationKey, message string) error {
	if err := parseForm(r); err != nil {
		return apperrors.EK(apperrors.KindInvalidInput, localizationKey, message)
	}
	return nil
}

// ParseFormOrRedirectErrorNotice parses form values and redirects back to the
// current module page with one flash error notice when parsing fails.
func ParseFormOrRedirectErrorNotice(w http.ResponseWriter, r *http.Request, localizationKey, redirectURL string) bool {
	if err := parseForm(r); err != nil {
		flash.Write(w, r, flash.Notice{Kind: flash.KindError, Key: localizationKey})
		WriteRedirect(w, r, redirectURL)
		return false
	}
	return true
}

// ReadRouteParam returns a trimmed route parameter and whether it is present.
func ReadRouteParam(r *http.Request, name string) (string, bool) {
	if r == nil {
		return "", false
	}
	value := strings.TrimSpace(r.PathValue(strings.TrimSpace(name)))
	if value == "" {
		return "", false
	}
	return value, true
}

// WithRequiredRouteParam extracts one required route parameter and delegates to
// fn. When the parameter is missing, onMissing handles the response instead.
func WithRequiredRouteParam(
	name string,
	onMissing func(http.ResponseWriter, *http.Request),
	fn func(http.ResponseWriter, *http.Request, string),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		value, ok := ReadRouteParam(r, name)
		if !ok {
			if onMissing != nil {
				onMissing(w, r)
			}
			return
		}
		if fn != nil {
			fn(w, r, value)
		}
	}
}

// parseForm centralizes nil-safe form parsing for the package transport
// policies.
func parseForm(r *http.Request) error {
	if r == nil {
		return http.ErrMissingFile
	}
	return r.ParseForm()
}
