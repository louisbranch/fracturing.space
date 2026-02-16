package credential

import (
	"errors"
	"testing"
	"time"
)

func TestNormalizeCreateInput(t *testing.T) {
	tests := []struct {
		name    string
		input   CreateInput
		wantErr error
		want    CreateInput
	}{
		{
			name:    "missing owner user id",
			input:   CreateInput{OwnerUserID: "", Provider: ProviderOpenAI, Label: "main", Secret: "sk-123"},
			wantErr: ErrEmptyOwnerUserID,
		},
		{
			name:    "unsupported provider",
			input:   CreateInput{OwnerUserID: "user-1", Provider: "anthropic", Label: "main", Secret: "sk-123"},
			wantErr: ErrInvalidProvider,
		},
		{
			name:    "empty secret",
			input:   CreateInput{OwnerUserID: "user-1", Provider: ProviderOpenAI, Label: "main", Secret: "   "},
			wantErr: ErrEmptySecret,
		},
		{
			name:  "normalizes trim",
			input: CreateInput{OwnerUserID: " user-1 ", Provider: ProviderOpenAI, Label: "  Main OpenAI  ", Secret: "  sk-123  "},
			want:  CreateInput{OwnerUserID: "user-1", Provider: ProviderOpenAI, Label: "Main OpenAI", Secret: "sk-123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeCreateInput(tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("NormalizeCreateInput error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeCreateInput error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeCreateInput = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestCreateCredential(t *testing.T) {
	fixedTime := time.Date(2026, 2, 15, 22, 36, 0, 0, time.UTC)
	input := CreateInput{
		OwnerUserID: "user-1",
		Provider:    ProviderOpenAI,
		Label:       "Main",
		Secret:      "sk-123",
	}

	_, err := Create(input, nil, func() (string, error) { return "", errors.New("id fail") })
	if err == nil {
		t.Fatal("expected id generation error")
	}

	created, err := Create(input, func() time.Time { return fixedTime }, func() (string, error) { return "cred-1", nil })
	if err != nil {
		t.Fatalf("Create error = %v", err)
	}

	if created.ID != "cred-1" {
		t.Fatalf("ID = %q, want %q", created.ID, "cred-1")
	}
	if created.OwnerUserID != "user-1" {
		t.Fatalf("OwnerUserID = %q, want %q", created.OwnerUserID, "user-1")
	}
	if created.Provider != ProviderOpenAI {
		t.Fatalf("Provider = %q, want %q", created.Provider, ProviderOpenAI)
	}
	if created.Status != StatusActive {
		t.Fatalf("Status = %q, want %q", created.Status, StatusActive)
	}
	if created.CreatedAt != fixedTime || created.UpdatedAt != fixedTime {
		t.Fatalf("timestamps = (%s,%s), want %s", created.CreatedAt, created.UpdatedAt, fixedTime)
	}
	if created.Secret != "sk-123" {
		t.Fatalf("Secret = %q, want %q", created.Secret, "sk-123")
	}
}
