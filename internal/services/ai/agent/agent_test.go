package agent

import (
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
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
			input:   CreateInput{OwnerUserID: "", Label: "narrator", Provider: provider.OpenAI, Model: "gpt-4o-mini", AuthReference: CredentialAuthReference("cred-1")},
			wantErr: ErrEmptyOwnerUserID,
		},
		{
			name:    "invalid provider",
			input:   CreateInput{OwnerUserID: "user-1", Label: "narrator", Provider: "other", Model: "gpt-4o-mini", AuthReference: CredentialAuthReference("cred-1")},
			wantErr: provider.ErrInvalid,
		},
		{
			name:    "missing model",
			input:   CreateInput{OwnerUserID: "user-1", Label: "narrator", Provider: provider.OpenAI, Model: "", AuthReference: CredentialAuthReference("cred-1")},
			wantErr: ErrEmptyModel,
		},
		{
			name:    "missing auth reference",
			input:   CreateInput{OwnerUserID: "user-1", Label: "narrator", Provider: provider.OpenAI, Model: "gpt-4o-mini"},
			wantErr: ErrMissingAuthReference,
		},
		{
			name:    "invalid auth reference kind",
			input:   CreateInput{OwnerUserID: "user-1", Label: "narrator", Provider: provider.OpenAI, Model: "gpt-4o-mini", AuthReference: AuthReference{Kind: "other", ID: "cred-1"}},
			wantErr: ErrInvalidAuthReference,
		},
		{
			name:    "invalid label format",
			input:   CreateInput{OwnerUserID: "user-1", Label: "Narrator Prime", Provider: provider.OpenAI, Model: "gpt-4o-mini", AuthReference: CredentialAuthReference("cred-1")},
			wantErr: ErrInvalidLabel,
		},
		{
			name:  "normalizes fields",
			input: CreateInput{OwnerUserID: " user-1 ", Label: " narrator ", Provider: provider.OpenAI, Model: "  gpt-4o-mini  ", AuthReference: CredentialAuthReference(" cred-1 ")},
			want:  CreateInput{OwnerUserID: "user-1", Label: "narrator", Provider: provider.OpenAI, Model: "gpt-4o-mini", AuthReference: CredentialAuthReference("cred-1")},
		},
		{
			name:  "normalizes provider grant auth reference",
			input: CreateInput{OwnerUserID: " user-1 ", Label: " narrator ", Provider: provider.OpenAI, Model: "  gpt-4o-mini  ", AuthReference: ProviderGrantAuthReference(" grant-1 ")},
			want:  CreateInput{OwnerUserID: "user-1", Label: "narrator", Provider: provider.OpenAI, Model: "gpt-4o-mini", AuthReference: ProviderGrantAuthReference("grant-1")},
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

func TestAuthReferenceFromIDs(t *testing.T) {
	ref, err := AuthReferenceFromIDs(" cred-1 ", "", true)
	if err != nil {
		t.Fatalf("AuthReferenceFromIDs credential: %v", err)
	}
	if ref != CredentialAuthReference("cred-1") {
		t.Fatalf("credential ref = %+v, want %+v", ref, CredentialAuthReference("cred-1"))
	}

	ref, err = AuthReferenceFromIDs("", " grant-1 ", true)
	if err != nil {
		t.Fatalf("AuthReferenceFromIDs provider grant: %v", err)
	}
	if ref != ProviderGrantAuthReference("grant-1") {
		t.Fatalf("provider grant ref = %+v, want %+v", ref, ProviderGrantAuthReference("grant-1"))
	}

	if _, err := AuthReferenceFromIDs("cred-1", "grant-1", true); !errors.Is(err, ErrMultipleAuthReferences) {
		t.Fatalf("AuthReferenceFromIDs mixed error = %v, want %v", err, ErrMultipleAuthReferences)
	}
}

func TestCreateAgent(t *testing.T) {
	fixedTime := time.Date(2026, 2, 15, 22, 40, 0, 0, time.UTC)
	input := CreateInput{
		OwnerUserID:   "user-1",
		Label:         "narrator",
		Provider:      provider.OpenAI,
		Model:         "gpt-4o-mini",
		AuthReference: ProviderGrantAuthReference("grant-1"),
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
	if created.AuthReference.ProviderGrantID() != "grant-1" {
		t.Fatalf("ProviderGrantID = %q, want %q", created.AuthReference.ProviderGrantID(), "grant-1")
	}
	if created.Label != "narrator" {
		t.Fatalf("Label = %q, want %q", created.Label, "narrator")
	}
	if created.AuthReference.CredentialID() != "" {
		t.Fatalf("CredentialID = %q, want empty", created.AuthReference.CredentialID())
	}
}

func TestStatusAndAuthReferenceHelpers(t *testing.T) {
	if !Status(" active ").IsActive() {
		t.Fatal("expected active status helper to normalize whitespace")
	}
	if ParseStatus("unknown") != "" {
		t.Fatal("expected unknown status to normalize to empty")
	}
	if got := (Agent{AuthReference: CredentialAuthReference("cred-1")}).AuthRefType(); got != "credential" {
		t.Fatalf("AuthRefType() = %q, want credential", got)
	}
	if got := (Agent{AuthReference: ProviderGrantAuthReference("grant-1")}).AuthRefType(); got != "provider_grant" {
		t.Fatalf("AuthRefType() = %q, want provider_grant", got)
	}
	if got := (Agent{}).AuthRefType(); got != "" {
		t.Fatalf("AuthRefType() = %q, want empty for zero auth ref", got)
	}
}

func TestAuthReferenceIsZero(t *testing.T) {
	if !(AuthReference{}).IsZero() {
		t.Fatal("expected IsZero for empty auth reference")
	}
	if CredentialAuthReference("cred-1").IsZero() {
		t.Fatal("expected not IsZero for populated auth reference")
	}
}

func TestAuthReferenceFromRecord(t *testing.T) {
	ref, err := AuthReferenceFromRecord(storage.AgentRecord{CredentialID: "cred-1"})
	if err != nil {
		t.Fatalf("AuthReferenceFromRecord credential: %v", err)
	}
	if ref != CredentialAuthReference("cred-1") {
		t.Fatalf("ref = %+v, want credential cred-1", ref)
	}

	ref, err = AuthReferenceFromRecord(storage.AgentRecord{ProviderGrantID: "grant-1"})
	if err != nil {
		t.Fatalf("AuthReferenceFromRecord provider grant: %v", err)
	}
	if ref != ProviderGrantAuthReference("grant-1") {
		t.Fatalf("ref = %+v, want provider_grant grant-1", ref)
	}

	_, err = AuthReferenceFromRecord(storage.AgentRecord{CredentialID: "c", ProviderGrantID: "g"})
	if !errors.Is(err, ErrMultipleAuthReferences) {
		t.Fatalf("expected ErrMultipleAuthReferences, got %v", err)
	}
}

func TestApplyAuthReference(t *testing.T) {
	var record storage.AgentRecord

	ApplyAuthReference(&record, CredentialAuthReference("cred-1"))
	if record.CredentialID != "cred-1" {
		t.Fatalf("CredentialID = %q, want %q", record.CredentialID, "cred-1")
	}
	if record.ProviderGrantID != "" {
		t.Fatalf("ProviderGrantID = %q, want empty", record.ProviderGrantID)
	}

	ApplyAuthReference(&record, ProviderGrantAuthReference("grant-1"))
	if record.ProviderGrantID != "grant-1" {
		t.Fatalf("ProviderGrantID = %q, want %q", record.ProviderGrantID, "grant-1")
	}
	if record.CredentialID != "" {
		t.Fatalf("CredentialID = %q, want empty", record.CredentialID)
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
			input:   UpdateInput{ID: "", OwnerUserID: "user-1", Label: "narrator", Model: "gpt-4o-mini", AuthReference: CredentialAuthReference("cred-2")},
			wantErr: ErrEmptyID,
		},
		{
			name:    "missing owner user id",
			input:   UpdateInput{ID: "agent-1", OwnerUserID: "", Label: "narrator", Model: "gpt-4o-mini", AuthReference: CredentialAuthReference("cred-2")},
			wantErr: ErrEmptyOwnerUserID,
		},
		{
			name:    "invalid auth reference kind",
			input:   UpdateInput{ID: "agent-1", OwnerUserID: "user-1", AuthReference: AuthReference{Kind: "other", ID: "cred-2"}},
			wantErr: ErrInvalidAuthReference,
		},
		{
			name:    "rejects invalid label",
			input:   UpdateInput{ID: "agent-1", OwnerUserID: "user-1", Label: "Narrator Prime"},
			wantErr: ErrInvalidLabel,
		},
		{
			name:  "normalizes optional fields",
			input: UpdateInput{ID: " agent-1 ", OwnerUserID: " user-1 ", Label: " narrator ", Model: " gpt-4o ", AuthReference: ProviderGrantAuthReference(" grant-2 ")},
			want:  UpdateInput{ID: "agent-1", OwnerUserID: "user-1", Label: "narrator", Model: "gpt-4o", AuthReference: ProviderGrantAuthReference("grant-2")},
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
