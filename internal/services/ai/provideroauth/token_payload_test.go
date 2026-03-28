package provideroauth

import (
	"strings"
	"testing"
)

func TestEncodeDecodeTokenPayload(t *testing.T) {
	raw, err := EncodeTokenPayload(TokenPayload{
		AccessToken:  " at ",
		RefreshToken: " rt ",
		TokenType:    " Bearer ",
		Scope:        " responses.read ",
	})
	if err != nil {
		t.Fatalf("EncodeTokenPayload: %v", err)
	}
	if strings.Contains(raw, " at ") || strings.Contains(raw, " rt ") {
		t.Fatalf("encoded payload was not normalized: %q", raw)
	}

	got, err := DecodeTokenPayload(raw)
	if err != nil {
		t.Fatalf("DecodeTokenPayload: %v", err)
	}
	if got.AccessToken != "at" || got.RefreshToken != "rt" || got.TokenType != "Bearer" || got.Scope != "responses.read" {
		t.Fatalf("decoded payload = %#v", got)
	}
}

func TestDecodeTokenPayloadRejectsEmptyOrInvalid(t *testing.T) {
	if _, err := DecodeTokenPayload(""); err == nil {
		t.Fatal("expected error for empty payload")
	}
	if _, err := DecodeTokenPayload("not-json"); err == nil {
		t.Fatal("expected error for invalid payload")
	}
}

func TestRefreshTokenFromPayload(t *testing.T) {
	raw, err := EncodeTokenPayload(TokenPayload{AccessToken: "at", RefreshToken: "rt"})
	if err != nil {
		t.Fatalf("EncodeTokenPayload: %v", err)
	}
	token, err := RefreshTokenFromPayload(raw)
	if err != nil {
		t.Fatalf("RefreshTokenFromPayload: %v", err)
	}
	if token != "rt" {
		t.Fatalf("refresh token = %q, want %q", token, "rt")
	}
}

func TestAccessTokenFromPayload(t *testing.T) {
	raw, err := EncodeTokenPayload(TokenPayload{AccessToken: "at", RefreshToken: "rt"})
	if err != nil {
		t.Fatalf("EncodeTokenPayload: %v", err)
	}
	token, err := AccessTokenFromPayload(raw)
	if err != nil {
		t.Fatalf("AccessTokenFromPayload: %v", err)
	}
	if token != "at" {
		t.Fatalf("access token = %q, want %q", token, "at")
	}
}

func TestRevokeTokenFromPayloadPrefersRefreshThenAccess(t *testing.T) {
	withRefresh, err := EncodeTokenPayload(TokenPayload{AccessToken: "at", RefreshToken: "rt"})
	if err != nil {
		t.Fatalf("EncodeTokenPayload: %v", err)
	}
	token, err := RevokeTokenFromPayload(withRefresh)
	if err != nil {
		t.Fatalf("RevokeTokenFromPayload(with refresh): %v", err)
	}
	if token != "rt" {
		t.Fatalf("revoke token = %q, want %q", token, "rt")
	}

	withoutRefresh, err := EncodeTokenPayload(TokenPayload{AccessToken: "at"})
	if err != nil {
		t.Fatalf("EncodeTokenPayload: %v", err)
	}
	token, err = RevokeTokenFromPayload(withoutRefresh)
	if err != nil {
		t.Fatalf("RevokeTokenFromPayload(with access only): %v", err)
	}
	if token != "at" {
		t.Fatalf("revoke token = %q, want %q", token, "at")
	}
}
