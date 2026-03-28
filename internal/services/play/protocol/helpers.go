package protocol

import (
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ProtoEnumToLower maps a proto enum value to a lowercase string by stripping
// the given prefix. Returns "" for unspecified or empty values. Exported for
// use by game-system protocol sub-packages.
func ProtoEnumToLower[T interface{ String() string }](value T, unspecified T, prefix string) string {
	name := strings.TrimSpace(value.String())
	if name == "" || name == unspecified.String() {
		return ""
	}
	return strings.ToLower(strings.TrimPrefix(name, prefix))
}

// FormatTimestamp formats a proto Timestamp as RFC3339, returning "" for nil or
// zero timestamps. Exported for use by game-system protocol sub-packages.
func FormatTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil || (ts.GetSeconds() == 0 && ts.GetNanos() == 0) {
		return ""
	}
	return ts.AsTime().Format(time.RFC3339)
}

// TrimStringSlice returns a new slice containing only non-empty trimmed
// strings. Exported for use by game-system protocol sub-packages.
func TrimStringSlice(values []string) []string {
	result := make([]string, 0, len(values))
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			result = append(result, s)
		}
	}
	return result
}

// PronounsString formats a proto Pronouns value as a lowercase display string.
// Exported for use by game-system protocol sub-packages.
func PronounsString(p *commonv1.Pronouns) string {
	if p == nil {
		return ""
	}
	switch v := p.GetValue().(type) {
	case *commonv1.Pronouns_Kind:
		name := strings.TrimSpace(v.Kind.String())
		if name == "" || name == commonv1.Pronoun_PRONOUN_UNSPECIFIED.String() {
			return ""
		}
		name = strings.TrimPrefix(name, "PRONOUN_")
		return strings.ToLower(strings.ReplaceAll(name, "_", "/"))
	case *commonv1.Pronouns_Custom:
		return strings.TrimSpace(v.Custom)
	default:
		return ""
	}
}

func interactionRoleString(value gamev1.ParticipantRole) string {
	return ProtoEnumToLower(value, gamev1.ParticipantRole_ROLE_UNSPECIFIED, "PARTICIPANT_ROLE_")
}

func localeString(value commonv1.Locale) string {
	name := strings.TrimSpace(value.String())
	if name == "" || name == commonv1.Locale_LOCALE_UNSPECIFIED.String() {
		return ""
	}
	name = strings.TrimPrefix(name, "LOCALE_")
	return strings.ToLower(strings.ReplaceAll(name, "_", "-"))
}
