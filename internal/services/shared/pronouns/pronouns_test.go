package pronouns

import (
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

func TestToProtoKnownAndCustom(t *testing.T) {
	known := ToProto("  THEY/THEM ")
	if known == nil {
		t.Fatal("expected known pronoun proto value")
	}
	if known.GetKind() != commonv1.Pronoun_PRONOUN_THEY_THEM {
		t.Fatalf("expected they/them enum, got %v", known.GetKind())
	}

	custom := ToProto("xe/xem")
	if custom == nil {
		t.Fatal("expected custom pronoun proto value")
	}
	if custom.GetCustom() != "xe/xem" {
		t.Fatalf("expected custom value xe/xem, got %q", custom.GetCustom())
	}

	if got := ToProto("   "); got != nil {
		t.Fatalf("expected nil for empty value, got %#v", got)
	}
}

func TestFromProto(t *testing.T) {
	if got := FromProto(nil); got != "" {
		t.Fatalf("expected empty string for nil proto, got %q", got)
	}

	known := &commonv1.Pronouns{
		Value: &commonv1.Pronouns_Kind{
			Kind: commonv1.Pronoun_PRONOUN_HE_HIM,
		},
	}
	if got := FromProto(known); got != PronounHeHim {
		t.Fatalf("expected %q, got %q", PronounHeHim, got)
	}

	custom := &commonv1.Pronouns{
		Value: &commonv1.Pronouns_Custom{
			Custom: " xe/xem ",
		},
	}
	if got := FromProto(custom); got != "xe/xem" {
		t.Fatalf("expected trimmed custom value, got %q", got)
	}
}
