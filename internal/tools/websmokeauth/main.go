package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// config holds web-smoke auth bootstrap inputs.
type config struct {
	authAddr       string
	email          string
	emailPrefix    string
	ttlSeconds     int64
	requestTimeout time.Duration
}

func main() {
	cfg := parseConfig()
	if strings.TrimSpace(cfg.authAddr) == "" {
		fatalf("auth address is required")
	}
	if cfg.ttlSeconds <= 0 {
		fatalf("ttl-seconds must be > 0")
	}
	if cfg.requestTimeout <= 0 {
		fatalf("timeout must be > 0")
	}

	email := strings.TrimSpace(cfg.email)
	if email == "" {
		email = generatedEmail(sanitizeEmailPrefix(cfg.emailPrefix))
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.requestTimeout)
	defer cancel()

	conn, err := grpc.NewClient(
		cfg.authAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		fatalf("dial auth %q: %v", cfg.authAddr, err)
	}
	defer conn.Close()

	client := authv1.NewAuthServiceClient(conn)
	userID := createUserID(ctx, client, email)
	recipientEmail := generatedEmail(sanitizeEmailPrefix(cfg.emailPrefix) + "-recipient")
	recipientUserID := createUserID(ctx, client, recipientEmail)

	createSessionResp, err := client.CreateWebSession(ctx, &authv1.CreateWebSessionRequest{
		UserId:     userID,
		TtlSeconds: cfg.ttlSeconds,
	})
	if err != nil {
		fatalf("create web session for user %q: %v", userID, err)
	}

	sessionID := strings.TrimSpace(createSessionResp.GetSession().GetId())
	if sessionID == "" {
		fatalf("create web session for user %q: missing session id", userID)
	}

	fmt.Printf("WEB_SMOKE_AUTH_EMAIL=%s\n", email)
	fmt.Printf("WEB_SMOKE_USER_ID=%s\n", userID)
	fmt.Printf("WEB_SMOKE_AUTH_RECIPIENT_EMAIL=%s\n", recipientEmail)
	fmt.Printf("WEB_SMOKE_RECIPIENT_USER_ID=%s\n", recipientUserID)
	fmt.Printf("WEB_SMOKE_SESSION_ID=%s\n", sessionID)
}

// parseConfig parses CLI flags into a config.
func parseConfig() config {
	var cfg config
	flag.StringVar(&cfg.authAddr, "auth-addr", "127.0.0.1:8083", "auth gRPC address")
	flag.StringVar(&cfg.email, "email", "", "explicit email to create (optional)")
	flag.StringVar(&cfg.emailPrefix, "email-prefix", "web-smoke", "generated email prefix when -email is empty")
	flag.Int64Var(&cfg.ttlSeconds, "ttl-seconds", 3600, "web session ttl in seconds")
	flag.DurationVar(&cfg.requestTimeout, "timeout", 10*time.Second, "request timeout for auth RPC calls")
	flag.Parse()
	return cfg
}

// sanitizeEmailPrefix normalizes arbitrary prefixes into a safe local-part stem.
func sanitizeEmailPrefix(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "web-smoke"
	}

	var b strings.Builder
	lastWasDash := false
	for _, r := range value {
		isAlphaNum := r >= 'a' && r <= 'z' || r >= '0' && r <= '9'
		if isAlphaNum {
			b.WriteRune(r)
			lastWasDash = false
			continue
		}
		if !lastWasDash {
			b.WriteByte('-')
			lastWasDash = true
		}
	}
	normalized := strings.Trim(strings.TrimSpace(b.String()), "-")
	if normalized == "" {
		return "web-smoke"
	}
	return normalized
}

func generatedEmail(prefix string) string {
	return fmt.Sprintf("%s-%d@example.com", prefix, time.Now().UTC().UnixNano())
}

func createUserID(ctx context.Context, client authv1.AuthServiceClient, email string) string {
	createUserResp, err := client.CreateUser(ctx, &authv1.CreateUserRequest{Email: email})
	if err != nil {
		fatalf("create auth user %q: %v", email, err)
	}
	userID := strings.TrimSpace(createUserResp.GetUser().GetId())
	if userID == "" {
		fatalf("create auth user %q: missing user id", email)
	}
	return userID
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "websmokeauth: "+format+"\n", args...)
	os.Exit(1)
}
