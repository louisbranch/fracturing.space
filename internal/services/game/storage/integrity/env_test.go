package integrity

import "testing"

func TestKeyringFromEnvRequiresKey(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEYS", "")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY_ID", "")

	if _, err := KeyringFromEnv(); err == nil {
		t.Fatal("expected error when no key is configured")
	}
}

func TestKeyringFromEnvSingleKey(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "secret")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEYS", "")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY_ID", "")

	ring, err := KeyringFromEnv()
	if err != nil {
		t.Fatalf("keyring from env: %v", err)
	}
	if ring.ActiveKeyID() != "v1" {
		t.Fatalf("expected default key id v1, got %s", ring.ActiveKeyID())
	}
}

func TestKeyringFromEnvWhitespaceKeySpecFallsBack(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "secret")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEYS", "   ")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY_ID", "")

	ring, err := KeyringFromEnv()
	if err != nil {
		t.Fatalf("keyring from env: %v", err)
	}
	if ring.ActiveKeyID() != "v1" {
		t.Fatalf("expected default key id v1, got %s", ring.ActiveKeyID())
	}
}

func TestKeyringFromEnvWhitespaceKeyIDUsesDefault(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "secret")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEYS", "")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY_ID", "   ")

	ring, err := KeyringFromEnv()
	if err != nil {
		t.Fatalf("keyring from env: %v", err)
	}
	if ring.ActiveKeyID() != "v1" {
		t.Fatalf("expected default key id v1, got %s", ring.ActiveKeyID())
	}
}

func TestKeyringFromEnvKeySpec(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEYS", "k1=one,k2=two")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY_ID", "k2")

	ring, err := KeyringFromEnv()
	if err != nil {
		t.Fatalf("keyring from env: %v", err)
	}
	if ring.ActiveKeyID() != "k2" {
		t.Fatalf("expected active key id k2, got %s", ring.ActiveKeyID())
	}
}

func TestKeyringFromEnvInvalidKeySpec(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEYS", "bad-entry")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY_ID", "k1")

	if _, err := KeyringFromEnv(); err == nil {
		t.Fatal("expected error for invalid key spec")
	}
}

func TestKeyringFromEnvEmptyKeySpecEntry(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEYS", "k1=one, ,k2=two")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY_ID", "k1")

	ring, err := KeyringFromEnv()
	if err != nil {
		t.Fatalf("keyring from env: %v", err)
	}
	if ring.ActiveKeyID() != "k1" {
		t.Fatalf("expected active key id k1, got %s", ring.ActiveKeyID())
	}
}

func TestKeyringFromEnvRejectsEmptyKeyValue(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEYS", "k1=one,k2=")
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY_ID", "k1")

	if _, err := KeyringFromEnv(); err == nil {
		t.Fatal("expected error for empty key value")
	}
}
