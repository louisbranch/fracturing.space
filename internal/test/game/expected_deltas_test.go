//go:build scenario

package game

import "testing"

func TestReadExpectedDeltas(t *testing.T) {
	tests := []struct {
		name   string
		args   map[string]any
		wantOK bool
		want   expectedDeltaInput
	}{
		{
			name:   "no expected deltas",
			args:   map[string]any{},
			wantOK: false,
		},
		{
			name: "character deltas",
			args: map[string]any{
				"expect_hope_delta":   1,
				"expect_stress_delta": -2,
				"expect_hp_delta":     -3,
				"expect_armor_delta":  -1,
			},
			wantOK: true,
			want: expectedDeltaInput{
				hopeDelta:   intPtr(1),
				stressDelta: intPtr(-2),
				hpDelta:     intPtr(-3),
				armorDelta:  intPtr(-1),
			},
		},
		{
			name: "gm fear delta",
			args: map[string]any{
				"expect_gm_fear_delta": 2,
			},
			wantOK: true,
			want: expectedDeltaInput{
				gmFearDelta: intPtr(2),
			},
		},
	}

	for _, tt := range tests {
		input, ok := readExpectedDeltas(tt.args)
		if ok != tt.wantOK {
			t.Fatalf("%s: ok = %v, want %v", tt.name, ok, tt.wantOK)
		}
		if !tt.wantOK {
			continue
		}
		assertExpectedDeltaInput(t, tt.name, input, tt.want)
	}
}

func TestReadExpectedAdversaryDeltas(t *testing.T) {
	tests := []struct {
		name   string
		args   map[string]any
		want   map[string]expectedAdversaryDelta
		wantOK bool
	}{
		{
			name:   "no adversary deltas",
			args:   map[string]any{},
			want:   nil,
			wantOK: false,
		},
		{
			name: "single adversary mitigated",
			args: map[string]any{
				"expect_adversary":                  "Nazgul",
				"expect_adversary_damage_mitigated": true,
			},
			wantOK: true,
			want: map[string]expectedAdversaryDelta{
				"Nazgul": {
					name:      "Nazgul",
					mitigated: boolPtr(true),
				},
			},
		},
		{
			name: "adversary delta list",
			args: map[string]any{
				"expect_adversary_deltas": []any{
					map[string]any{
						"target":           "Nazgul",
						"hp_delta":         -1,
						"damage_mitigated": true,
					},
				},
			},
			wantOK: true,
			want: map[string]expectedAdversaryDelta{
				"Nazgul": {
					name:      "Nazgul",
					hpDelta:   intPtr(-1),
					mitigated: boolPtr(true),
				},
			},
		},
	}

	for _, tt := range tests {
		got := readExpectedAdversaryDeltas(t, tt.args, "")
		if tt.wantOK != (got != nil) {
			t.Fatalf("%s: ok = %v, want %v", tt.name, got != nil, tt.wantOK)
		}
		if !tt.wantOK {
			continue
		}
		assertExpectedAdversaryDeltaMap(t, tt.name, got, tt.want)
	}
}

func assertExpectedDeltaInput(t *testing.T, name string, got expectedDeltaInput, want expectedDeltaInput) {
	t.Helper()
	if !intPtrEqual(got.hopeDelta, want.hopeDelta) {
		t.Fatalf("%s: hope_delta = %v, want %v", name, got.hopeDelta, want.hopeDelta)
	}
	if !intPtrEqual(got.stressDelta, want.stressDelta) {
		t.Fatalf("%s: stress_delta = %v, want %v", name, got.stressDelta, want.stressDelta)
	}
	if !intPtrEqual(got.hpDelta, want.hpDelta) {
		t.Fatalf("%s: hp_delta = %v, want %v", name, got.hpDelta, want.hpDelta)
	}
	if !intPtrEqual(got.armorDelta, want.armorDelta) {
		t.Fatalf("%s: armor_delta = %v, want %v", name, got.armorDelta, want.armorDelta)
	}
	if !intPtrEqual(got.gmFearDelta, want.gmFearDelta) {
		t.Fatalf("%s: gm_fear_delta = %v, want %v", name, got.gmFearDelta, want.gmFearDelta)
	}
}

func intPtr(value int) *int {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}

func intPtrEqual(left *int, right *int) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	return *left == *right
}

func boolPtrEqual(left *bool, right *bool) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	return *left == *right
}

func assertExpectedAdversaryDeltaMap(t *testing.T, name string, got map[string]expectedAdversaryDelta, want map[string]expectedAdversaryDelta) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: adversary deltas len = %d, want %d", name, len(got), len(want))
	}
	for key, wantDelta := range want {
		gotDelta, ok := got[key]
		if !ok {
			t.Fatalf("%s: missing adversary %s", name, key)
		}
		if !intPtrEqual(gotDelta.hpDelta, wantDelta.hpDelta) {
			t.Fatalf("%s: %s hp_delta = %v, want %v", name, key, gotDelta.hpDelta, wantDelta.hpDelta)
		}
		if !intPtrEqual(gotDelta.armorDelta, wantDelta.armorDelta) {
			t.Fatalf("%s: %s armor_delta = %v, want %v", name, key, gotDelta.armorDelta, wantDelta.armorDelta)
		}
		if !boolPtrEqual(gotDelta.mitigated, wantDelta.mitigated) {
			t.Fatalf("%s: %s damage_mitigated = %v, want %v", name, key, gotDelta.mitigated, wantDelta.mitigated)
		}
	}
}
