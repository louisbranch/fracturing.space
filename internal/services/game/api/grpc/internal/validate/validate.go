package validate

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Transport-layer string length limits. These prevent oversized payloads from
// reaching domain logic and inflating storage. Values are in bytes.
const (
	MaxNameLen        = 200
	MaxDescriptionLen = 5000
	MaxNotesLen       = 5000
	MaxReasonLen      = 1000
	MaxPromptLen      = 5000
)

// RequiredID trims whitespace from raw and returns an InvalidArgument error
// if the result is empty. The name parameter is used in the error message
// (e.g. "campaign id").
func RequiredID(raw, name string) (string, error) {
	id := strings.TrimSpace(raw)
	if id == "" {
		return "", status.Error(codes.InvalidArgument, name+" is required")
	}
	return id, nil
}

// MaxLength returns an InvalidArgument error if value exceeds maxLen bytes.
// Empty values pass — combine with RequiredID for mandatory bounded fields.
func MaxLength(value, name string, maxLen int) error {
	if len(value) > maxLen {
		return status.Errorf(codes.InvalidArgument, "%s exceeds maximum length of %d", name, maxLen)
	}
	return nil
}
