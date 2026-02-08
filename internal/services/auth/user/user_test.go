package user

import (
	"errors"
	"testing"
	"time"
)

func TestCreateUserDefaults(t *testing.T) {
	input := CreateUserInput{DisplayName: "Alice"}
	_, err := CreateUser(input, nil, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	_, err = CreateUser(input, nil, func() (string, error) { return "", errors.New("id generator error") })
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateUserNormalizesInput(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 10, 0, 0, 0, time.UTC)
	input := CreateUserInput{DisplayName: "  Alice  "}

	created, err := CreateUser(input, func() time.Time { return fixedTime }, func() (string, error) {
		return "user-123", nil
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	if created.ID != "user-123" {
		t.Fatalf("expected id user-123, got %q", created.ID)
	}
	if created.DisplayName != "Alice" {
		t.Fatalf("expected trimmed display name, got %q", created.DisplayName)
	}
	if !created.CreatedAt.Equal(fixedTime) || !created.UpdatedAt.Equal(fixedTime) {
		t.Fatalf("expected timestamps to match fixed time")
	}
}

func TestNormalizeCreateUserInputValidation(t *testing.T) {
	_, err := NormalizeCreateUserInput(CreateUserInput{DisplayName: "   "})
	if !errors.Is(err, ErrEmptyDisplayName) {
		t.Fatalf("expected error %v, got %v", ErrEmptyDisplayName, err)
	}
}
