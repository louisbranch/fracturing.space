package oauth

import "testing"

func TestComputeS256Challenge(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	want := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	if got := ComputeS256Challenge(verifier); got != want {
		t.Fatalf("ComputeS256Challenge() = %v, want %v", got, want)
	}
}

func TestValidatePKCE(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	if !ValidatePKCE(verifier, challenge, "S256") {
		t.Fatal("expected PKCE validation to pass")
	}
	if ValidatePKCE(verifier, challenge, "plain") {
		t.Fatal("expected PKCE validation to fail for non-S256 method")
	}
	if ValidatePKCE("short", challenge, "S256") {
		t.Fatal("expected PKCE validation to fail for invalid verifier")
	}
	if ValidatePKCE(verifier, "invalid", "S256") {
		t.Fatal("expected PKCE validation to fail for mismatched challenge")
	}
}

func TestValidateCodeChallenge(t *testing.T) {
	valid := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	if !ValidateCodeChallenge(valid) {
		t.Fatal("expected valid code challenge")
	}
	if ValidateCodeChallenge("short") {
		t.Fatal("expected invalid length to fail")
	}
	if ValidateCodeChallenge("E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw+M") {
		t.Fatal("expected invalid characters to fail")
	}
}
