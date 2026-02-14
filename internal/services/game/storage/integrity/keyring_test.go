package integrity

import "testing"

func TestNewKeyringValidation(t *testing.T) {
	if _, err := NewKeyring(nil, "v1"); err == nil {
		t.Fatal("expected error for missing keys")
	}

	if _, err := NewKeyring(map[string][]byte{"v1": []byte("secret")}, ""); err == nil {
		t.Fatal("expected error for missing active key id")
	}

	if _, err := NewKeyring(map[string][]byte{"v1": []byte("secret")}, "v2"); err == nil {
		t.Fatal("expected error for unknown active key id")
	}
}

func TestKeyringSignAndVerify(t *testing.T) {
	ring, err := NewKeyring(map[string][]byte{"v1": []byte("secret")}, "v1")
	if err != nil {
		t.Fatalf("new keyring: %v", err)
	}

	sig, keyID, err := ring.SignChainHash("c1", "chainhash")
	if err != nil {
		t.Fatalf("sign chain hash: %v", err)
	}
	if keyID != "v1" {
		t.Fatalf("expected key id v1, got %s", keyID)
	}

	if err := ring.VerifyChainHash("c1", "chainhash", sig, keyID); err != nil {
		t.Fatalf("verify chain hash: %v", err)
	}
}

func TestKeyringVerifyFailures(t *testing.T) {
	ring, err := NewKeyring(map[string][]byte{"v1": []byte("secret")}, "v1")
	if err != nil {
		t.Fatalf("new keyring: %v", err)
	}

	sig, _, err := ring.SignChainHash("c1", "chainhash")
	if err != nil {
		t.Fatalf("sign chain hash: %v", err)
	}

	if err := ring.VerifyChainHash("c1", "chainhash", sig, ""); err == nil {
		t.Fatal("expected error for missing key id")
	}
	if err := ring.VerifyChainHash("c1", "chainhash", sig, "unknown"); err == nil {
		t.Fatal("expected error for unknown key id")
	}
	if err := ring.VerifyChainHash("c1", "chainhash", "bad", "v1"); err == nil {
		t.Fatal("expected error for signature mismatch")
	}
}

func TestKeyringActiveKeyID(t *testing.T) {
	var ring *Keyring
	if ring.ActiveKeyID() != "" {
		t.Fatal("expected empty active key id for nil keyring")
	}

	ring, err := NewKeyring(map[string][]byte{"v1": []byte("secret")}, "v1")
	if err != nil {
		t.Fatalf("new keyring: %v", err)
	}
	if ring.ActiveKeyID() != "v1" {
		t.Fatalf("expected active key id v1, got %s", ring.ActiveKeyID())
	}
}

func TestKeyringSignRequiresKeyring(t *testing.T) {
	var ring *Keyring
	if _, _, err := ring.SignChainHash("c1", "hash"); err == nil {
		t.Fatal("expected error for nil keyring")
	}
}

func TestKeyringVerifyRequiresCampaignID(t *testing.T) {
	ring, err := NewKeyring(map[string][]byte{"v1": []byte("secret")}, "v1")
	if err != nil {
		t.Fatalf("new keyring: %v", err)
	}

	sig, keyID, err := ring.SignChainHash("c1", "hash")
	if err != nil {
		t.Fatalf("sign chain hash: %v", err)
	}

	if err := ring.VerifyChainHash("", "hash", sig, keyID); err == nil {
		t.Fatal("expected error for missing campaign id")
	}
}
