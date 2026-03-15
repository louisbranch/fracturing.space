package gametest

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc/metadata"
)

// FixedClock returns a deterministic clock function for tests.
func FixedClock(t time.Time) func() time.Time {
	return func() time.Time {
		return t
	}
}

// FixedIDGenerator returns an ID generator that always yields the same ID.
func FixedIDGenerator(id string) func() (string, error) {
	return func() (string, error) {
		return id, nil
	}
}

// FixedSequenceIDGenerator returns IDs in order and then repeats the last ID.
func FixedSequenceIDGenerator(ids ...string) func() (string, error) {
	index := 0
	return func() (string, error) {
		if index >= len(ids) {
			return ids[len(ids)-1], nil
		}
		id := ids[index]
		index++
		return id, nil
	}
}

// SequentialIDGenerator returns IDs with an incrementing numeric suffix.
func SequentialIDGenerator(prefix string) func() (string, error) {
	counter := 0
	return func() (string, error) {
		counter++
		return prefix + "-" + string(rune('0'+counter)), nil
	}
}

// ContextWithParticipantID injects a participant ID into incoming gRPC metadata.
func ContextWithParticipantID(participantID string) context.Context {
	if participantID == "" {
		return context.Background()
	}
	md := metadata.Pairs(grpcmeta.ParticipantIDHeader, participantID)
	return metadata.NewIncomingContext(context.Background(), md)
}

// ContextWithUserID injects a user ID into incoming gRPC metadata.
func ContextWithUserID(userID string) context.Context {
	if userID == "" {
		return context.Background()
	}
	md := metadata.Pairs(grpcmeta.UserIDHeader, userID)
	return metadata.NewIncomingContext(context.Background(), md)
}

// ContextWithAdminOverride injects admin override metadata for transport tests.
func ContextWithAdminOverride(reason string) context.Context {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "test-override"
	}
	md := metadata.Pairs(
		grpcmeta.PlatformRoleHeader, grpcmeta.PlatformRoleAdmin,
		grpcmeta.AuthzOverrideReasonHeader, reason,
		grpcmeta.UserIDHeader, "user-admin-test",
	)
	return metadata.NewIncomingContext(context.Background(), md)
}

// JoinGrantSigner signs test join-grant JWTs using a per-test Ed25519 keypair.
type JoinGrantSigner struct {
	Issuer   string
	Audience string
	Key      ed25519.PrivateKey
}

// NewJoinGrantSigner provisions signer material and test env for join grants.
func NewJoinGrantSigner(t *testing.T) JoinGrantSigner {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate join grant key: %v", err)
	}
	issuer := "test-issuer"
	audience := "game-service"
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", issuer)
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", audience)
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY", base64.RawStdEncoding.EncodeToString(publicKey))
	return JoinGrantSigner{
		Issuer:   issuer,
		Audience: audience,
		Key:      privateKey,
	}
}

// Token returns a signed test join-grant JWT for the provided campaign invite.
func (s JoinGrantSigner) Token(t *testing.T, campaignID, inviteID, userID, jti string, now time.Time) string {
	t.Helper()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if s.Key == nil {
		t.Fatal("join grant signer key is required")
	}
	if strings.TrimSpace(jti) == "" {
		jti = fmt.Sprintf("jti-%d", now.UnixNano())
	}
	headerJSON, err := json.Marshal(map[string]string{
		"alg": "EdDSA",
		"typ": "JWT",
	})
	if err != nil {
		t.Fatalf("encode join grant header: %v", err)
	}
	payloadJSON, err := json.Marshal(map[string]any{
		"iss":         s.Issuer,
		"aud":         s.Audience,
		"exp":         now.Add(5 * time.Minute).Unix(),
		"iat":         now.Unix(),
		"jti":         jti,
		"campaign_id": strings.TrimSpace(campaignID),
		"invite_id":   strings.TrimSpace(inviteID),
		"user_id":     strings.TrimSpace(userID),
	})
	if err != nil {
		t.Fatalf("encode join grant payload: %v", err)
	}
	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := encodedHeader + "." + encodedPayload
	signature := ed25519.Sign(s.Key, []byte(signingInput))
	encodedSig := base64.RawURLEncoding.EncodeToString(signature)
	return signingInput + "." + encodedSig
}
