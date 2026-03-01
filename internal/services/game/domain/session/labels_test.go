package session

import "testing"

func TestNormalizeStatus(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      Status
		wantValid bool
	}{
		{name: "empty", input: " ", want: StatusUnspecified, wantValid: false},
		{name: "active short label", input: "ACTIVE", want: StatusActive, wantValid: true},
		{name: "active enum label", input: "session_status_active", want: StatusActive, wantValid: true},
		{name: "ended short label", input: " ended ", want: StatusEnded, wantValid: true},
		{name: "ended enum label", input: "SESSION_STATUS_ENDED", want: StatusEnded, wantValid: true},
		{name: "invalid", input: "paused", want: StatusUnspecified, wantValid: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := NormalizeStatus(tc.input)
			if got != tc.want || ok != tc.wantValid {
				t.Fatalf("NormalizeStatus(%q) = (%q, %v), want (%q, %v)", tc.input, got, ok, tc.want, tc.wantValid)
			}
		})
	}
}

func TestNormalizeGateType(t *testing.T) {
	got, err := NormalizeGateType("  GM_Consequence ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "gm_consequence" {
		t.Fatalf("NormalizeGateType() = %q, want %q", got, "gm_consequence")
	}

	_, err = NormalizeGateType("   ")
	if err == nil {
		t.Fatal("expected error for empty gate type")
	}
}

func TestNormalizeGateReason(t *testing.T) {
	if got := NormalizeGateReason("  reason  "); got != "reason" {
		t.Fatalf("NormalizeGateReason() = %q, want %q", got, "reason")
	}
}

func TestNormalizeSpotlightType(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      SpotlightType
		shouldErr bool
	}{
		{name: "gm", input: " GM ", want: SpotlightTypeGM},
		{name: "character", input: "character", want: SpotlightTypeCharacter},
		{name: "invalid", input: "narrator", shouldErr: true},
		{name: "missing", input: " ", shouldErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeSpotlightType(tc.input)
			if tc.shouldErr {
				if err == nil {
					t.Fatalf("expected error for input %q", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("NormalizeSpotlightType(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestValidateSpotlightTarget(t *testing.T) {
	if err := ValidateSpotlightTarget(SpotlightTypeCharacter, " "); err == nil {
		t.Fatal("expected missing character id error for character spotlight")
	}
	if err := ValidateSpotlightTarget(SpotlightTypeGM, "char-1"); err == nil {
		t.Fatal("expected invalid gm target error")
	}
	if err := ValidateSpotlightTarget(SpotlightTypeCharacter, "char-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ValidateSpotlightTarget(SpotlightTypeGM, " "); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
