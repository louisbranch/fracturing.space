package pronouns

import (
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

// Canonical pronoun values used as persisted and transport-agnostic defaults.
const (
	PronounSheHer   = "she/her"
	PronounHeHim    = "he/him"
	PronounTheyThem = "they/them"
	PronounItIts    = "it/its"
)

var canon = map[string]commonv1.Pronoun{
	PronounSheHer:   commonv1.Pronoun_PRONOUN_SHE_HER,
	PronounHeHim:    commonv1.Pronoun_PRONOUN_HE_HIM,
	PronounTheyThem: commonv1.Pronoun_PRONOUN_THEY_THEM,
	PronounItIts:    commonv1.Pronoun_PRONOUN_IT_ITS,
}

var protoToCanonical = map[commonv1.Pronoun]string{
	commonv1.Pronoun_PRONOUN_SHE_HER:   PronounSheHer,
	commonv1.Pronoun_PRONOUN_HE_HIM:    PronounHeHim,
	commonv1.Pronoun_PRONOUN_THEY_THEM: PronounTheyThem,
	commonv1.Pronoun_PRONOUN_IT_ITS:    PronounItIts,
}

// FromProto converts a pronouns message to its canonical string form.
func FromProto(value *commonv1.Pronouns) string {
	if value == nil {
		return ""
	}

	switch data := value.Value.(type) {
	case *commonv1.Pronouns_Kind:
		return protoToCanonical[data.Kind]
	case *commonv1.Pronouns_Custom:
		return strings.TrimSpace(data.Custom)
	default:
		return ""
	}
}

// ToProto converts a canonical or custom pronouns string to the transport type.
func ToProto(value string) *commonv1.Pronouns {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if kind, ok := canon[strings.ToLower(value)]; ok {
		return &commonv1.Pronouns{
			Value: &commonv1.Pronouns_Kind{
				Kind: kind,
			},
		}
	}
	return &commonv1.Pronouns{
		Value: &commonv1.Pronouns_Custom{
			Custom: value,
		},
	}
}
