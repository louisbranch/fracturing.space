package integrity

import "testing"

func TestKeyringFromEnvRequiresKey(t *testing.T) {
	t.Setenv(envHMACKey, "")
	t.Setenv(envHMACKeys, "")
	t.Setenv(envHMACKeyID, "")

	if _, err := KeyringFromEnv(); err == nil {
		t.Fatal("expected error when no key is configured")
	}
}

func TestKeyringFromEnvSingleKey(t *testing.T) {
	t.Setenv(envHMACKey, "secret")
	t.Setenv(envHMACKeys, "")
	t.Setenv(envHMACKeyID, "")

	ring, err := KeyringFromEnv()
	if err != nil {
		t.Fatalf("keyring from env: %v", err)
	}
	if ring.ActiveKeyID() != defaultKeyID {
		t.Fatalf("expected default key id %s, got %s", defaultKeyID, ring.ActiveKeyID())
	}
}

func TestKeyringFromEnvKeySpec(t *testing.T) {
	t.Setenv(envHMACKey, "")
	t.Setenv(envHMACKeys, "k1=one,k2=two")
	t.Setenv(envHMACKeyID, "k2")

	ring, err := KeyringFromEnv()
	if err != nil {
		t.Fatalf("keyring from env: %v", err)
	}
	if ring.ActiveKeyID() != "k2" {
		t.Fatalf("expected active key id k2, got %s", ring.ActiveKeyID())
	}
}

func TestKeyringFromEnvInvalidKeySpec(t *testing.T) {
	t.Setenv(envHMACKey, "")
	t.Setenv(envHMACKeys, "bad-entry")
	t.Setenv(envHMACKeyID, "k1")

	if _, err := KeyringFromEnv(); err == nil {
		t.Fatal("expected error for invalid key spec")
	}
}

func TestKeyringFromEnvEmptyKeySpecEntry(t *testing.T) {
	t.Setenv(envHMACKey, "")
	t.Setenv(envHMACKeys, "k1=one, ,k2=two")
	t.Setenv(envHMACKeyID, "k1")

	ring, err := KeyringFromEnv()
	if err != nil {
		t.Fatalf("keyring from env: %v", err)
	}
	if ring.ActiveKeyID() != "k1" {
		t.Fatalf("expected active key id k1, got %s", ring.ActiveKeyID())
	}
}

func TestKeyringFromEnvRejectsEmptyKeyValue(t *testing.T) {
	t.Setenv(envHMACKey, "")
	t.Setenv(envHMACKeys, "k1=one,k2=")
	t.Setenv(envHMACKeyID, "k1")

	if _, err := KeyringFromEnv(); err == nil {
		t.Fatal("expected error for empty key value")
	}
}
