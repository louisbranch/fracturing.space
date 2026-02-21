package user

import (
	"errors"
	"testing"
	"time"
)

func TestCreateUserDefaults(t *testing.T) {
	input := CreateUserInput{Email: "alice@example.com"}
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

	_, err = CreateUser(input, nil, func() (string, error) { return "", errors.New("id generator error") })
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateUserNormalizesInput(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 10, 0, 0, 0, time.UTC)
	input := CreateUserInput{
		Email: "  ALICE@example.com  ",
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
	if created.Email != "alice@example.com" {
		t.Fatalf("expected lowercased trimmed email, got %q", created.Email)
	}
	if !created.CreatedAt.Equal(fixedTime) || !created.UpdatedAt.Equal(fixedTime) {
		t.Fatalf("expected timestamps to match fixed time")
	}
}

func TestNormalizeCreateUserInputValidation(t *testing.T) {
	_, err := NormalizeCreateUserInput(CreateUserInput{Email: "   "})
	if !errors.Is(err, ErrEmptyEmail) {
		t.Fatalf("expected error %v, got %v", ErrEmptyEmail, err)
	}
}

func TestValidateEmailFormat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "valid email", input: "alice@example.com", wantErr: nil},
		{name: "valid mixed case email", input: "AlIcE@Example.Com", wantErr: nil},
		{name: "missing domain", input: "alice@", wantErr: ErrInvalidEmail},
		{name: "display name email", input: "Alice <alice@example.com>", wantErr: ErrInvalidEmail},
		{name: "spaces", input: "alice @example.com", wantErr: ErrInvalidEmail},
		{name: "no at sign", input: "alice.example.com", wantErr: ErrInvalidEmail},
		{name: "empty", input: "", wantErr: ErrInvalidEmail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.input)
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
	normalized, err := NormalizeCreateUserInput(CreateUserInput{Email: "  ALICE@Example.COM  "})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if normalized.Email != "alice@example.com" {
		t.Fatalf("expected lowercased email, got %q", normalized.Email)
	}
}

func TestNormalizeCreateUserInputRejectsInvalid(t *testing.T) {
	_, err := NormalizeCreateUserInput(CreateUserInput{Email: "not-an-email"})
	if !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestNormalizeCreateUserInputAcceptsEmailAddress(t *testing.T) {
	_, err := NormalizeCreateUserInput(CreateUserInput{Email: "ALICE@EXAMPLE.COM"})
	if err != nil {
		t.Fatalf("expected email address to be accepted, got %v", err)
	}
}
