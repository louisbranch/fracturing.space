package user

import (
	"errors"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

func TestCreateUserDefaults(t *testing.T) {
	input := CreateUserInput{Username: "Alice"}
	_, err := CreateUser(input, nil, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	created, err := CreateUser(input, nil, func() (string, error) { return "user-1", nil })
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if created.ID != "user-1" {
		t.Fatalf("expected id user-1, got %q", created.ID)
	}
	if created.Username != "alice" {
		t.Fatalf("expected canonical username, got %q", created.Username)
	}

	_, err = CreateUser(input, nil, func() (string, error) { return "", errors.New("id generator error") })
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateUserNormalizesInput(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 10, 0, 0, 0, time.UTC)
	input := CreateUserInput{
		Username: "  ALICE  ",
	}

	created, err := CreateUser(input, func() time.Time { return fixedTime }, func() (string, error) {
		return "user-123", nil
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	if created.ID != "user-123" {
		t.Fatalf("expected id user-123, got %q", created.ID)
	}
	if created.Username != "alice" {
		t.Fatalf("expected canonical username, got %q", created.Username)
	}
	if created.Locale != commonv1.Locale_LOCALE_EN_US {
		t.Fatalf("expected normalized default locale, got %v", created.Locale)
	}
	if !created.CreatedAt.Equal(fixedTime) || !created.UpdatedAt.Equal(fixedTime) {
		t.Fatalf("expected timestamps to match fixed time")
	}
	if !created.RecoveryCodeUpdatedAt.Equal(fixedTime) {
		t.Fatalf("expected recovery code timestamp to match fixed time")
	}
}

func TestNormalizeCreateUserInputNormalizesLocale(t *testing.T) {
	normalized, err := NormalizeCreateUserInput(CreateUserInput{
		Username: "alice",
		Locale:   commonv1.Locale_LOCALE_UNSPECIFIED,
	})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if normalized.Locale != commonv1.Locale_LOCALE_EN_US {
		t.Fatalf("locale = %v, want %v", normalized.Locale, commonv1.Locale_LOCALE_EN_US)
	}
}

func TestNormalizeCreateUserInputValidation(t *testing.T) {
	_, err := NormalizeCreateUserInput(CreateUserInput{Username: ""})
	if !errors.Is(err, ErrEmptyUsername) {
		t.Fatalf("expected error %v, got %v", ErrEmptyUsername, err)
	}
}

func TestValidateUsernameFormat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "valid username", input: "alice", wantErr: nil},
		{name: "valid mixed case username", input: "AlIcE_123", wantErr: nil},
		{name: "too short", input: "ab", wantErr: ErrInvalidUsername},
		{name: "starts with number", input: "1alice", wantErr: ErrInvalidUsername},
		{name: "space", input: "alice smith", wantErr: ErrInvalidUsername},
		{name: "empty", input: "", wantErr: ErrInvalidUsername},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUsername(tt.input)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestNormalizeCreateUserInputCanonicalizesUsername(t *testing.T) {
	normalized, err := NormalizeCreateUserInput(CreateUserInput{Username: "  ALICE.Example  "})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if normalized.Username != "alice.example" {
		t.Fatalf("expected canonical username, got %q", normalized.Username)
	}
}

func TestNormalizeCreateUserInputRejectsInvalidUsername(t *testing.T) {
	_, err := NormalizeCreateUserInput(CreateUserInput{Username: "not valid"})
	if !errors.Is(err, ErrInvalidUsername) {
		t.Fatalf("expected ErrInvalidUsername, got %v", err)
	}
}
