package domain

import (
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
)

// pronounsUpdateField converts optional MCP update input into proto update
// semantics. Nil means "field omitted"; blank means explicit clear.
func pronounsUpdateField(value *string) (*commonv1.Pronouns, bool) {
	if value == nil {
		return nil, false
	}

	if strings.TrimSpace(*value) == "" {
		return &commonv1.Pronouns{
			Value: &commonv1.Pronouns_Kind{
				Kind: commonv1.Pronoun_PRONOUN_UNSPECIFIED,
			},
		}, true
	}

	return sharedpronouns.ToProto(*value), true
}
