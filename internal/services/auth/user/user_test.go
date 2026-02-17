package user

import (
	"errors"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
)

func TestCreateUserDefaults(t *testing.T) {
	input := CreateUserInput{PrimaryEmail: "alice@example.com"}
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
	input := CreateUserInput{
		PrimaryEmail: "  ALICE@example.com  ",
		Locale:       commonv1.Locale_LOCALE_PT_BR,
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
	if created.PrimaryEmail != "alice@example.com" {
		t.Fatalf("expected lowercased trimmed primary email, got %q", created.PrimaryEmail)
	}
	if created.Locale != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("expected locale %v, got %v", commonv1.Locale_LOCALE_PT_BR, created.Locale)
	}
	if !created.CreatedAt.Equal(fixedTime) || !created.UpdatedAt.Equal(fixedTime) {
		t.Fatalf("expected timestamps to match fixed time")
	}
}

func TestNormalizeCreateUserInputValidation(t *testing.T) {
	_, err := NormalizeCreateUserInput(CreateUserInput{PrimaryEmail: "   "})
	if !errors.Is(err, ErrEmptyPrimaryEmail) {
		t.Fatalf("expected error %v, got %v", ErrEmptyPrimaryEmail, err)
	}
}

func TestValidatePrimaryEmailFormat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "valid email", input: "alice@example.com", wantErr: nil},
		{name: "valid mixed case email", input: "AlIcE@Example.Com", wantErr: nil},
		{name: "missing domain", input: "alice@", wantErr: ErrInvalidPrimaryEmail},
		{name: "display name email", input: "Alice <alice@example.com>", wantErr: ErrInvalidPrimaryEmail},
		{name: "spaces", input: "alice @example.com", wantErr: ErrInvalidPrimaryEmail},
		{name: "no at sign", input: "alice.example.com", wantErr: ErrInvalidPrimaryEmail},
		{name: "empty", input: "", wantErr: ErrInvalidPrimaryEmail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePrimaryEmail(tt.input)
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
	normalized, err := NormalizeCreateUserInput(CreateUserInput{PrimaryEmail: "  ALICE@Example.COM  "})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if normalized.PrimaryEmail != "alice@example.com" {
		t.Fatalf("expected lowercased primary email, got %q", normalized.PrimaryEmail)
	}
}

func TestNormalizeCreateUserInputRejectsInvalid(t *testing.T) {
	_, err := NormalizeCreateUserInput(CreateUserInput{PrimaryEmail: "not-an-email"})
	if !errors.Is(err, ErrInvalidPrimaryEmail) {
		t.Fatalf("expected ErrInvalidPrimaryEmail, got %v", err)
	}
}

func TestNormalizeCreateUserInputAcceptsEmailAddress(t *testing.T) {
	_, err := NormalizeCreateUserInput(CreateUserInput{PrimaryEmail: "ALICE@EXAMPLE.COM"})
	if err != nil {
		t.Fatalf("expected email address to be accepted, got %v", err)
	}
}
