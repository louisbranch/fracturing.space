// Package i18n provides locale helpers for the platform.
package i18n

import (
	"strings"

	"golang.org/x/text/message"
)

// Localizer is the minimal formatting contract needed to resolve keyed copy at
// render time without coupling platform helpers to one transport package.
type Localizer interface {
	Sprintf(key message.Reference, args ...any) string
}

// CopyRef stores user-facing copy as a catalog key plus positional string
// arguments so persisted and transported payloads stay locale-neutral.
type CopyRef struct {
	Key  string   `json:"key"`
	Args []string `json:"args,omitempty"`
}

// NewCopyRef constructs normalized copy references at authoring seams.
func NewCopyRef(key string, args ...string) CopyRef {
	ref, _ := NormalizeCopyRef(CopyRef{Key: key, Args: args})
	return ref
}

// NormalizeCopyRef trims and validates one copy reference.
func NormalizeCopyRef(ref CopyRef) (CopyRef, bool) {
	ref.Key = strings.TrimSpace(ref.Key)
	if ref.Key == "" {
		return CopyRef{}, false
	}
	if len(ref.Args) > 0 {
		args := make([]string, 0, len(ref.Args))
		for _, arg := range ref.Args {
			args = append(args, strings.TrimSpace(arg))
		}
		ref.Args = args
	} else {
		ref.Args = nil
	}
	return ref, true
}

// ResolveCopy renders one copy reference in the current locale.
func ResolveCopy(loc Localizer, ref CopyRef) string {
	normalized, ok := NormalizeCopyRef(ref)
	if !ok {
		return ""
	}
	if loc == nil {
		return normalized.Key
	}
	args := make([]any, 0, len(normalized.Args))
	for _, arg := range normalized.Args {
		args = append(args, arg)
	}
	return loc.Sprintf(normalized.Key, args...)
}
