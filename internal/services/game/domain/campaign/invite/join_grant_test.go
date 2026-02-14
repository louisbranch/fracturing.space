package invite

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestLoadJoinGrantConfigFromEnv(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", "")
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", "")
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY", "")

	if _, err := LoadJoinGrantConfigFromEnv(nil); err == nil {
		t.Fatal("expected error when env vars are missing")
	}

	pubKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", "issuer")
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", "audience")
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY", base64.RawStdEncoding.EncodeToString(pubKey))

	cfg, err := LoadJoinGrantConfigFromEnv(nil)
	if err != nil {
		t.Fatalf("load join grant config: %v", err)
	}
	if cfg.Issuer != "issuer" || cfg.Audience != "audience" {
		t.Fatal("expected issuer and audience to be loaded")
	}
	if len(cfg.Key) != ed25519.PublicKeySize {
		t.Fatalf("expected public key size %d", ed25519.PublicKeySize)
	}
}

func TestLoadJoinGrantConfigFromEnvTrimsIssuer(t *testing.T) {
	pubKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", "   ")
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", "audience")
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY", base64.RawStdEncoding.EncodeToString(pubKey))

	_, err = LoadJoinGrantConfigFromEnv(nil)
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("expected required error, got %v", err)
	}
}

func TestValidateJoinGrantSuccess(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	now := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	grant := signJoinGrant(t, priv, map[string]any{
		"alg": "EdDSA",
		"typ": "JWT",
	}, map[string]any{
		"iss":         "issuer",
		"aud":         []string{"game-service", "secondary"},
		"exp":         now.Add(2 * time.Hour).Unix(),
		"iat":         now.Add(-time.Minute).Unix(),
		"jti":         "jti-1",
		"campaign_id": "campaign-1",
		"invite_id":   "invite-1",
		"user_id":     "user-1",
	})

	cfg := JoinGrantConfig{Issuer: "issuer", Audience: "game-service", Key: pub, Now: func() time.Time { return now }}
	claims, err := ValidateJoinGrant(grant, JoinGrantExpectation{CampaignID: "campaign-1", InviteID: "invite-1", UserID: "user-1"}, cfg)
	if err != nil {
		t.Fatalf("validate join grant: %v", err)
	}
	if claims.Issuer != "issuer" {
		t.Fatalf("expected issuer claim issuer, got %s", claims.Issuer)
	}
	if claims.CampaignID != "campaign-1" || claims.InviteID != "invite-1" || claims.UserID != "user-1" {
		t.Fatal("expected campaign, invite, and user claims to match")
	}
	if !claims.ExpiresAt.Equal(time.Unix(now.Add(2*time.Hour).Unix(), 0).UTC()) {
		t.Fatal("expected expires at to match exp")
	}
}

func TestValidateJoinGrantExpired(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	now := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	grant := signJoinGrant(t, priv, map[string]any{"alg": "EdDSA"}, map[string]any{
		"iss":         "issuer",
		"aud":         "game-service",
		"exp":         now.Add(-time.Minute).Unix(),
		"jti":         "jti-1",
		"campaign_id": "campaign-1",
		"invite_id":   "invite-1",
		"user_id":     "user-1",
	})

	cfg := JoinGrantConfig{Issuer: "issuer", Audience: "game-service", Key: pub, Now: func() time.Time { return now }}
	_, err = ValidateJoinGrant(grant, JoinGrantExpectation{CampaignID: "campaign-1", InviteID: "invite-1", UserID: "user-1"}, cfg)
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired error, got %v", err)
	}
}

func TestValidateJoinGrantMismatch(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	now := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	grant := signJoinGrant(t, priv, map[string]any{"alg": "EdDSA"}, map[string]any{
		"iss":         "issuer",
		"aud":         "game-service",
		"exp":         now.Add(time.Hour).Unix(),
		"jti":         "jti-1",
		"campaign_id": "campaign-1",
		"invite_id":   "invite-1",
		"user_id":     "user-2",
	})

	cfg := JoinGrantConfig{Issuer: "issuer", Audience: "game-service", Key: pub, Now: func() time.Time { return now }}
	_, err = ValidateJoinGrant(grant, JoinGrantExpectation{CampaignID: "campaign-1", InviteID: "invite-1", UserID: "user-1"}, cfg)
	if err == nil || !strings.Contains(err.Error(), "user mismatch") {
		t.Fatalf("expected user mismatch error, got %v", err)
	}
}

func TestValidateJoinGrantInvalidSignature(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	cfg := JoinGrantConfig{Issuer: "issuer", Audience: "game-service", Key: pub, Now: time.Now}
	_, err = ValidateJoinGrant("invalid.token.parts", JoinGrantExpectation{}, cfg)
	if err == nil {
		t.Fatal("expected error for invalid join grant")
	}
}

func TestValidateJoinGrantEmpty(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	cfg := JoinGrantConfig{Issuer: "issuer", Audience: "game-service", Key: pub, Now: time.Now}
	_, err := ValidateJoinGrant("", JoinGrantExpectation{}, cfg)
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("expected required error, got %v", err)
	}
}

func TestValidateJoinGrantInvalidAlgorithm(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	grant := signJoinGrant(t, priv, map[string]any{"alg": "HS256"}, map[string]any{
		"iss": "issuer", "aud": "game-service", "exp": now.Add(time.Hour).Unix(),
		"jti": "jti-1", "campaign_id": "c1", "invite_id": "i1", "user_id": "u1",
	})
	cfg := JoinGrantConfig{Issuer: "issuer", Audience: "game-service", Key: pub, Now: func() time.Time { return now }}
	_, err := ValidateJoinGrant(grant, JoinGrantExpectation{CampaignID: "c1", InviteID: "i1", UserID: "u1"}, cfg)
	if err == nil {
		t.Fatal("expected error for invalid algorithm")
	}
}

func TestValidateJoinGrantIssuerMismatch(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	grant := signJoinGrant(t, priv, map[string]any{"alg": "EdDSA"}, map[string]any{
		"iss": "wrong-issuer", "aud": "game-service", "exp": now.Add(time.Hour).Unix(),
		"jti": "jti-1", "campaign_id": "c1", "invite_id": "i1", "user_id": "u1",
	})
	cfg := JoinGrantConfig{Issuer: "issuer", Audience: "game-service", Key: pub, Now: func() time.Time { return now }}
	_, err := ValidateJoinGrant(grant, JoinGrantExpectation{CampaignID: "c1", InviteID: "i1", UserID: "u1"}, cfg)
	if err == nil || !strings.Contains(err.Error(), "issuer") {
		t.Fatalf("expected issuer error, got %v", err)
	}
}

func TestValidateJoinGrantAudienceMismatch(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	grant := signJoinGrant(t, priv, map[string]any{"alg": "EdDSA"}, map[string]any{
		"iss": "issuer", "aud": []string{"wrong-service"}, "exp": now.Add(time.Hour).Unix(),
		"jti": "jti-1", "campaign_id": "c1", "invite_id": "i1", "user_id": "u1",
	})
	cfg := JoinGrantConfig{Issuer: "issuer", Audience: "game-service", Key: pub, Now: func() time.Time { return now }}
	_, err := ValidateJoinGrant(grant, JoinGrantExpectation{CampaignID: "c1", InviteID: "i1", UserID: "u1"}, cfg)
	if err == nil || !strings.Contains(err.Error(), "audience") {
		t.Fatalf("expected audience error, got %v", err)
	}
}

func TestValidateJoinGrantMissingJTI(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	grant := signJoinGrant(t, priv, map[string]any{"alg": "EdDSA"}, map[string]any{
		"iss": "issuer", "aud": "game-service", "exp": now.Add(time.Hour).Unix(),
		"campaign_id": "c1", "invite_id": "i1", "user_id": "u1",
	})
	cfg := JoinGrantConfig{Issuer: "issuer", Audience: "game-service", Key: pub, Now: func() time.Time { return now }}
	_, err := ValidateJoinGrant(grant, JoinGrantExpectation{CampaignID: "c1", InviteID: "i1", UserID: "u1"}, cfg)
	if err == nil || !strings.Contains(err.Error(), "jti") {
		t.Fatalf("expected jti error, got %v", err)
	}
}

func TestValidateJoinGrantMissingExp(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	grant := signJoinGrant(t, priv, map[string]any{"alg": "EdDSA"}, map[string]any{
		"iss": "issuer", "aud": "game-service",
		"jti": "jti-1", "campaign_id": "c1", "invite_id": "i1", "user_id": "u1",
	})
	cfg := JoinGrantConfig{Issuer: "issuer", Audience: "game-service", Key: pub, Now: func() time.Time { return now }}
	_, err := ValidateJoinGrant(grant, JoinGrantExpectation{CampaignID: "c1", InviteID: "i1", UserID: "u1"}, cfg)
	if err == nil || !strings.Contains(err.Error(), "exp") {
		t.Fatalf("expected exp error, got %v", err)
	}
}

func TestValidateJoinGrantNotYetActive(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	grant := signJoinGrant(t, priv, map[string]any{"alg": "EdDSA"}, map[string]any{
		"iss": "issuer", "aud": "game-service", "exp": now.Add(time.Hour).Unix(),
		"nbf": now.Add(time.Minute).Unix(), // future
		"jti": "jti-1", "campaign_id": "c1", "invite_id": "i1", "user_id": "u1",
	})
	cfg := JoinGrantConfig{Issuer: "issuer", Audience: "game-service", Key: pub, Now: func() time.Time { return now }}
	_, err := ValidateJoinGrant(grant, JoinGrantExpectation{CampaignID: "c1", InviteID: "i1", UserID: "u1"}, cfg)
	if err == nil || !strings.Contains(err.Error(), "not active") {
		t.Fatalf("expected not active error, got %v", err)
	}
}

func TestValidateJoinGrantWithNbfAndIat(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	grant := signJoinGrant(t, priv, map[string]any{"alg": "EdDSA"}, map[string]any{
		"iss": "issuer", "aud": "game-service", "exp": now.Add(time.Hour).Unix(),
		"nbf": now.Add(-time.Minute).Unix(), "iat": now.Add(-time.Minute).Unix(),
		"jti": "jti-1", "campaign_id": "c1", "invite_id": "i1", "user_id": "u1",
	})
	cfg := JoinGrantConfig{Issuer: "issuer", Audience: "game-service", Key: pub, Now: func() time.Time { return now }}
	claims, err := ValidateJoinGrant(grant, JoinGrantExpectation{CampaignID: "c1", InviteID: "i1", UserID: "u1"}, cfg)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if claims.NotBefore.IsZero() {
		t.Fatal("expected NotBefore to be set")
	}
	if claims.IssuedAt.IsZero() {
		t.Fatal("expected IssuedAt to be set")
	}
}

func TestValidateJoinGrantCampaignMismatch(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	grant := signJoinGrant(t, priv, map[string]any{"alg": "EdDSA"}, map[string]any{
		"iss": "issuer", "aud": "game-service", "exp": now.Add(time.Hour).Unix(),
		"jti": "jti-1", "campaign_id": "c2", "invite_id": "i1", "user_id": "u1",
	})
	cfg := JoinGrantConfig{Issuer: "issuer", Audience: "game-service", Key: pub, Now: func() time.Time { return now }}
	_, err := ValidateJoinGrant(grant, JoinGrantExpectation{CampaignID: "c1", InviteID: "i1", UserID: "u1"}, cfg)
	if err == nil || !strings.Contains(err.Error(), "campaign") {
		t.Fatalf("expected campaign mismatch error, got %v", err)
	}
}

func TestValidateJoinGrantInviteMismatch(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
	grant := signJoinGrant(t, priv, map[string]any{"alg": "EdDSA"}, map[string]any{
		"iss": "issuer", "aud": "game-service", "exp": now.Add(time.Hour).Unix(),
		"jti": "jti-1", "campaign_id": "c1", "invite_id": "i2", "user_id": "u1",
	})
	cfg := JoinGrantConfig{Issuer: "issuer", Audience: "game-service", Key: pub, Now: func() time.Time { return now }}
	_, err := ValidateJoinGrant(grant, JoinGrantExpectation{CampaignID: "c1", InviteID: "i1", UserID: "u1"}, cfg)
	if err == nil || !strings.Contains(err.Error(), "invite") {
		t.Fatalf("expected invite mismatch error, got %v", err)
	}
}

func TestLoadJoinGrantConfigInvalidBase64(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", "issuer")
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", "audience")
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY", "!!!invalid!!!")
	_, err := LoadJoinGrantConfigFromEnv(nil)
	if err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestLoadJoinGrantConfigWrongKeySize(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", "issuer")
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", "audience")
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY", base64.RawStdEncoding.EncodeToString([]byte("short")))
	_, err := LoadJoinGrantConfigFromEnv(nil)
	if err == nil || !strings.Contains(err.Error(), "32 bytes") {
		t.Fatalf("expected key size error, got %v", err)
	}
}

func TestDecodeBase64Empty(t *testing.T) {
	_, err := decodeBase64("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestDecodeBase64StdEncoding(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("hello"))
	decoded, err := decodeBase64(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(decoded) != "hello" {
		t.Fatalf("expected 'hello', got %s", string(decoded))
	}
}

func TestDecodeBase64RawStdEncoding(t *testing.T) {
	encoded := base64.RawStdEncoding.EncodeToString([]byte("hello"))
	decoded, err := decodeBase64(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(decoded) != "hello" {
		t.Fatalf("expected 'hello', got %s", string(decoded))
	}
}

func signJoinGrant(t *testing.T, privateKey ed25519.PrivateKey, header, payload map[string]any) string {
	t.Helper()

	headerJSON, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := encodedHeader + "." + encodedPayload
	signature := ed25519.Sign(privateKey, []byte(signingInput))
	encodedSig := base64.RawURLEncoding.EncodeToString(signature)
	return signingInput + "." + encodedSig
}
