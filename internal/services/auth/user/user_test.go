package user

import (
	"errors"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
)

func TestCreateUserDefaults(t *testing.T) {
	input := CreateUserInput{Username: "alice"}
	_, err := CreateUser(input, nil, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	created, err := CreateUser(input, nil, func() (string, error) { return "user-1", nil })
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if created.Locale != platformi18n.DefaultLocale() {
		t.Fatalf("expected default locale %v, got %v", platformi18n.DefaultLocale(), created.Locale)
	}

	_, err = CreateUser(input, nil, func() (string, error) { return "", errors.New("id generator error") })
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateUserNormalizesInput(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 10, 0, 0, 0, time.UTC)
	input := CreateUserInput{Username: "  Alice  ", Locale: commonv1.Locale_LOCALE_PT_BR}

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
		t.Fatalf("expected lowercased trimmed username, got %q", created.Username)
	}
	if created.Locale != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("expected locale %v, got %v", commonv1.Locale_LOCALE_PT_BR, created.Locale)
	}
	if !created.CreatedAt.Equal(fixedTime) || !created.UpdatedAt.Equal(fixedTime) {
		t.Fatalf("expected timestamps to match fixed time")
	}
}

func TestNormalizeCreateUserInputValidation(t *testing.T) {
	_, err := NormalizeCreateUserInput(CreateUserInput{Username: "   "})
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
		{name: "valid lowercase", input: "alice", wantErr: nil},
		{name: "valid with dots", input: "alice.b", wantErr: nil},
		{name: "valid with dashes", input: "alice-b", wantErr: nil},
		{name: "valid with underscores", input: "alice_b", wantErr: nil},
		{name: "valid with numbers", input: "alice123", wantErr: nil},
		{name: "valid min length", input: "abc", wantErr: nil},
		{name: "valid max length", input: "abcdefghijklmnopqrstuvwxyz012345", wantErr: nil},
		{name: "too short", input: "ab", wantErr: ErrInvalidUsername},
		{name: "too long", input: "abcdefghijklmnopqrstuvwxyz0123456", wantErr: ErrInvalidUsername},
		{name: "uppercase", input: "Alice", wantErr: ErrInvalidUsername},
		{name: "spaces", input: "ali ce", wantErr: ErrInvalidUsername},
		{name: "special chars", input: "ali@ce", wantErr: ErrInvalidUsername},
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

func TestNormalizeCreateUserInputLowercases(t *testing.T) {
	normalized, err := NormalizeCreateUserInput(CreateUserInput{Username: "  Alice  "})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if normalized.Username != "alice" {
		t.Fatalf("expected lowercased username, got %q", normalized.Username)
	}
}

func TestNormalizeCreateUserInputRejectsInvalid(t *testing.T) {
	_, err := NormalizeCreateUserInput(CreateUserInput{Username: "ab"})
	if !errors.Is(err, ErrInvalidUsername) {
		t.Fatalf("expected ErrInvalidUsername, got %v", err)
	}
}
