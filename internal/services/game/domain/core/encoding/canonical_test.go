package encoding

import (
	"testing"
)

func TestCanonicalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    string
		wantErr bool
	}{
		{
			name:  "simple object sorted keys",
			input: map[string]any{"z": 1, "a": 2, "m": 3},
			want:  `{"a":2,"m":3,"z":1}`,
		},
		{
			name:  "nested object sorted keys",
			input: map[string]any{"b": map[string]any{"d": 1, "c": 2}, "a": 3},
			want:  `{"a":3,"b":{"c":2,"d":1}}`,
		},
		{
			name:  "array preserved order",
			input: []any{3, 1, 2},
			want:  `[3,1,2]`,
		},
		{
			name:  "mixed types",
			input: map[string]any{"str": "hello", "num": 42, "bool": true, "null": nil},
			want:  `{"bool":true,"null":null,"num":42,"str":"hello"}`,
		},
		{
			name:  "empty object",
			input: map[string]any{},
			want:  `{}`,
		},
		{
			name:  "empty array",
			input: []any{},
			want:  `[]`,
		},
		{
			name: "event envelope structure",
			input: map[string]any{
				"campaign_id": "camp_123",
				"event_type":  "campaign.created",
				"timestamp":   "2024-01-15T10:30:00Z",
				"actor_type":  "system",
				"payload": map[string]any{
					"name":        "Test Campaign",
					"game_system": "daggerheart",
				},
			},
			want: `{"actor_type":"system","campaign_id":"camp_123","event_type":"campaign.created","payload":{"game_system":"daggerheart","name":"Test Campaign"},"timestamp":"2024-01-15T10:30:00Z"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CanonicalJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanonicalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(got) != tt.want {
				t.Errorf("CanonicalJSON() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestContentHash(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		wantLen int
		wantErr bool
	}{
		{
			name:    "simple object produces 32 char hash",
			input:   map[string]any{"key": "value"},
			wantLen: 32, // 128 bits = 16 bytes = 32 hex chars
		},
		{
			name:    "empty object produces hash",
			input:   map[string]any{},
			wantLen: 32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ContentHash(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ContentHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.wantLen {
				t.Errorf("ContentHash() length = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestContentHash_Deterministic(t *testing.T) {
	// Same input in different key order should produce same hash
	input1 := map[string]any{"z": 1, "a": 2, "m": 3}
	input2 := map[string]any{"a": 2, "m": 3, "z": 1}
	input3 := map[string]any{"m": 3, "z": 1, "a": 2}

	hash1, err := ContentHash(input1)
	if err != nil {
		t.Fatalf("ContentHash(input1) error = %v", err)
	}

	hash2, err := ContentHash(input2)
	if err != nil {
		t.Fatalf("ContentHash(input2) error = %v", err)
	}

	hash3, err := ContentHash(input3)
	if err != nil {
		t.Fatalf("ContentHash(input3) error = %v", err)
	}

	if hash1 != hash2 || hash2 != hash3 {
		t.Errorf("ContentHash not deterministic: %s, %s, %s", hash1, hash2, hash3)
	}
}

func TestContentHash_DifferentInputsDifferentHashes(t *testing.T) {
	input1 := map[string]any{"key": "value1"}
	input2 := map[string]any{"key": "value2"}

	hash1, _ := ContentHash(input1)
	hash2, _ := ContentHash(input2)

	if hash1 == hash2 {
		t.Error("Different inputs should produce different hashes")
	}
}
