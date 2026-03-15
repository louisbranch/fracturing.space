package credential

import (
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
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
			input:   CreateInput{OwnerUserID: "", Provider: provider.OpenAI, Label: "main", Secret: "sk-123"},
			wantErr: ErrEmptyOwnerUserID,
		},
		{
			name:    "unsupported provider",
			input:   CreateInput{OwnerUserID: "user-1", Provider: "anthropic", Label: "main", Secret: "sk-123"},
			wantErr: provider.ErrInvalid,
		},
		{
			name:    "empty secret",
			input:   CreateInput{OwnerUserID: "user-1", Provider: provider.OpenAI, Label: "main", Secret: "   "},
			wantErr: ErrEmptySecret,
		},
		{
			name:  "normalizes trim",
			input: CreateInput{OwnerUserID: " user-1 ", Provider: provider.OpenAI, Label: "  Main OpenAI  ", Secret: "  sk-123  "},
			want:  CreateInput{OwnerUserID: "user-1", Provider: provider.OpenAI, Label: "Main OpenAI", Secret: "sk-123"},
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
		Provider:    provider.OpenAI,
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
	if created.Provider != provider.OpenAI {
		t.Fatalf("Provider = %q, want %q", created.Provider, provider.OpenAI)
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

func TestStatusAndUsabilityHelpers(t *testing.T) {
	record := Credential{
		OwnerUserID: "user-1",
		Provider:    provider.OpenAI,
		Status:      Status(" active "),
	}
	if !record.Status.IsActive() {
		t.Fatal("expected active status helper to normalize whitespace")
	}
	if Status(" revoked ").IsActive() {
		t.Fatal("expected revoked status to be inactive")
	}
	if !Status(" revoked ").IsRevoked() {
		t.Fatal("expected revoked status helper to normalize whitespace")
	}
	if !record.IsUsableBy("user-1", provider.OpenAI) {
		t.Fatal("expected credential to be usable by matching owner/provider")
	}
	if record.IsUsableBy("user-2", provider.OpenAI) {
		t.Fatal("expected credential usability to reject wrong owner")
	}
}
