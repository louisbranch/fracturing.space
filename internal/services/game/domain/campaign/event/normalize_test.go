package event

import "testing"

func TestNormalizeForAppend(t *testing.T) {
	tests := []struct {
		name      string
		input     Event
		wantErr   bool
		assertion func(t *testing.T, evt Event)
	}{
		{
			name: "defaults actor type and payload",
			input: Event{
				CampaignID:  "camp-1",
				Type:        TypeCampaignCreated,
				PayloadJSON: nil,
			},
			wantErr: false,
			assertion: func(t *testing.T, evt Event) {
				if evt.ActorType != ActorTypeSystem {
					t.Fatalf("ActorType = %s, want %s", evt.ActorType, ActorTypeSystem)
				}
				if string(evt.PayloadJSON) != "{}" {
					t.Fatalf("PayloadJSON = %s, want {}", string(evt.PayloadJSON))
				}
			},
		},
		{
			name: "rejects invalid actor type",
			input: Event{
				CampaignID:  "camp-1",
				Type:        TypeCampaignCreated,
				ActorType:   ActorType("alien"),
				PayloadJSON: []byte("{}"),
			},
			wantErr: true,
		},
		{
			name: "rejects missing actor id for participant",
			input: Event{
				CampaignID:  "camp-1",
				Type:        TypeCampaignCreated,
				ActorType:   ActorTypeParticipant,
				PayloadJSON: []byte("{}"),
			},
			wantErr: true,
		},
		{
			name: "rejects missing actor id for gm",
			input: Event{
				CampaignID:  "camp-1",
				Type:        TypeCampaignCreated,
				ActorType:   ActorTypeGM,
				PayloadJSON: []byte("{}"),
			},
			wantErr: true,
		},
		{
			name: "rejects invalid payload json",
			input: Event{
				CampaignID:  "camp-1",
				Type:        TypeCampaignCreated,
				PayloadJSON: []byte("{"),
			},
			wantErr: true,
		},
		{
			name: "rejects preset sequence",
			input: Event{
				CampaignID:  "camp-1",
				Type:        TypeCampaignCreated,
				Seq:         7,
				PayloadJSON: []byte("{}"),
			},
			wantErr: true,
		},
		{
			name: "rejects preset hash",
			input: Event{
				CampaignID:  "camp-1",
				Type:        TypeCampaignCreated,
				Hash:        "hash",
				PayloadJSON: []byte("{}"),
			},
			wantErr: true,
		},
		{
			name:    "rejects empty campaign id",
			input:   Event{CampaignID: "  ", Type: TypeCampaignCreated, PayloadJSON: []byte("{}")},
			wantErr: true,
		},
		{
			name:    "rejects empty event type",
			input:   Event{CampaignID: "camp-1", Type: Type("  "), PayloadJSON: []byte("{}")},
			wantErr: true,
		},
		{
			name: "rejects preset chain hashes",
			input: Event{
				CampaignID:  "camp-1",
				Type:        TypeCampaignCreated,
				PrevHash:    "prev",
				PayloadJSON: []byte("{}"),
			},
			wantErr: true,
		},
		{
			name: "rejects preset signatures",
			input: Event{
				CampaignID:     "camp-1",
				Type:           TypeCampaignCreated,
				SignatureKeyID: "key-1",
				PayloadJSON:    []byte("{}"),
			},
			wantErr: true,
		},
		{
			name: "accepts gm actor with actor id",
			input: Event{
				CampaignID:  "camp-1",
				Type:        TypeCampaignCreated,
				ActorType:   ActorTypeGM,
				ActorID:     "gm-1",
				PayloadJSON: []byte("{}"),
			},
			wantErr: false,
			assertion: func(t *testing.T, evt Event) {
				if evt.ActorType != ActorTypeGM {
					t.Fatalf("ActorType = %s, want %s", evt.ActorType, ActorTypeGM)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized, err := NormalizeForAppend(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.assertion != nil {
				tt.assertion(t, normalized)
			}
		})
	}
}
