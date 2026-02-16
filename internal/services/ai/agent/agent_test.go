package agent

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
			input:   CreateInput{OwnerUserID: "", Name: "Narrator", Provider: ProviderOpenAI, Model: "gpt-4o-mini", CredentialID: "cred-1"},
			wantErr: ErrEmptyOwnerUserID,
		},
		{
			name:    "invalid provider",
			input:   CreateInput{OwnerUserID: "user-1", Name: "Narrator", Provider: "other", Model: "gpt-4o-mini", CredentialID: "cred-1"},
			wantErr: ErrInvalidProvider,
		},
		{
			name:    "missing model",
			input:   CreateInput{OwnerUserID: "user-1", Name: "Narrator", Provider: ProviderOpenAI, Model: "", CredentialID: "cred-1"},
			wantErr: ErrEmptyModel,
		},
		{
			name:    "missing auth reference",
			input:   CreateInput{OwnerUserID: "user-1", Name: "Narrator", Provider: ProviderOpenAI, Model: "gpt-4o-mini"},
			wantErr: ErrMissingAuthReference,
		},
		{
			name:    "multiple auth references",
			input:   CreateInput{OwnerUserID: "user-1", Name: "Narrator", Provider: ProviderOpenAI, Model: "gpt-4o-mini", CredentialID: "cred-1", ProviderGrantID: "grant-1"},
			wantErr: ErrMultipleAuthReferences,
		},
		{
			name:  "normalizes fields",
			input: CreateInput{OwnerUserID: " user-1 ", Name: "  Narrator  ", Provider: ProviderOpenAI, Model: "  gpt-4o-mini  ", CredentialID: " cred-1 "},
			want:  CreateInput{OwnerUserID: "user-1", Name: "Narrator", Provider: ProviderOpenAI, Model: "gpt-4o-mini", CredentialID: "cred-1"},
		},
		{
			name:  "normalizes provider grant auth reference",
			input: CreateInput{OwnerUserID: " user-1 ", Name: "  Narrator  ", Provider: ProviderOpenAI, Model: "  gpt-4o-mini  ", ProviderGrantID: " grant-1 "},
			want:  CreateInput{OwnerUserID: "user-1", Name: "Narrator", Provider: ProviderOpenAI, Model: "gpt-4o-mini", ProviderGrantID: "grant-1"},
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

func TestCreateAgent(t *testing.T) {
	fixedTime := time.Date(2026, 2, 15, 22, 40, 0, 0, time.UTC)
	input := CreateInput{
		OwnerUserID:     "user-1",
		Name:            "Narrator",
		Provider:        ProviderOpenAI,
		Model:           "gpt-4o-mini",
		ProviderGrantID: "grant-1",
	}

	_, err := Create(input, nil, func() (string, error) { return "", errors.New("id fail") })
	if err == nil {
		t.Fatal("expected id generation error")
	}

	created, err := Create(input, func() time.Time { return fixedTime }, func() (string, error) { return "agent-1", nil })
	if err != nil {
		t.Fatalf("Create error = %v", err)
	}

	if created.ID != "agent-1" {
		t.Fatalf("ID = %q, want %q", created.ID, "agent-1")
	}
	if created.Status != StatusActive {
		t.Fatalf("Status = %q, want %q", created.Status, StatusActive)
	}
	if created.CreatedAt != fixedTime || created.UpdatedAt != fixedTime {
		t.Fatalf("timestamps = (%s,%s), want %s", created.CreatedAt, created.UpdatedAt, fixedTime)
	}
	if created.ProviderGrantID != "grant-1" {
		t.Fatalf("ProviderGrantID = %q, want %q", created.ProviderGrantID, "grant-1")
	}
	if created.CredentialID != "" {
		t.Fatalf("CredentialID = %q, want empty", created.CredentialID)
	}
}

func TestNormalizeUpdateInput(t *testing.T) {
	tests := []struct {
		name    string
		input   UpdateInput
		wantErr error
		want    UpdateInput
	}{
		{
			name:    "missing agent id",
			input:   UpdateInput{ID: "", OwnerUserID: "user-1", Name: "Narrator", Model: "gpt-4o-mini", CredentialID: "cred-2"},
			wantErr: ErrEmptyID,
		},
		{
			name:    "missing owner user id",
			input:   UpdateInput{ID: "agent-1", OwnerUserID: "", Name: "Narrator", Model: "gpt-4o-mini", CredentialID: "cred-2"},
			wantErr: ErrEmptyOwnerUserID,
		},
		{
			name:    "multiple auth references",
			input:   UpdateInput{ID: "agent-1", OwnerUserID: "user-1", CredentialID: "cred-2", ProviderGrantID: "grant-1"},
			wantErr: ErrMultipleAuthReferences,
		},
		{
			name:  "normalizes optional fields",
			input: UpdateInput{ID: " agent-1 ", OwnerUserID: " user-1 ", Name: "  Narrator  ", Model: " gpt-4o ", ProviderGrantID: " grant-2 "},
			want:  UpdateInput{ID: "agent-1", OwnerUserID: "user-1", Name: "Narrator", Model: "gpt-4o", ProviderGrantID: "grant-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeUpdateInput(tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("NormalizeUpdateInput error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeUpdateInput error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeUpdateInput = %+v, want %+v", got, tt.want)
			}
		})
	}
}
