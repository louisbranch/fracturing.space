// Package main bootstraps auth users and session ids for web smoke workflows.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// config holds web-smoke auth bootstrap inputs.
type config struct {
	authAddr          string
	username          string
	recipientUsername string
	ttlSeconds        int64
	requestTimeout    time.Duration
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(exitCode(err))
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	cfg, err := parseConfig(args, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return &exitError{code: 2, err: err}
	}
	if strings.TrimSpace(cfg.authAddr) == "" {
		return failf(stderr, 1, "auth address is required")
	}
	if cfg.ttlSeconds <= 0 {
		return failf(stderr, 1, "ttl-seconds must be > 0")
	}
	if cfg.requestTimeout <= 0 {
		return failf(stderr, 1, "timeout must be > 0")
	}
	username := strings.TrimSpace(cfg.username)
	if username == "" {
		return failf(stderr, 1, "username is required")
	}
	recipientUsername := strings.TrimSpace(cfg.recipientUsername)
	if recipientUsername == "" {
		return failf(stderr, 1, "recipient-username is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.requestTimeout)
	defer cancel()

	conn, err := grpc.NewClient(
		cfg.authAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		return failf(stderr, 1, "dial auth %q: %v", cfg.authAddr, err)
	}
	defer conn.Close()

	client := authv1.NewAuthServiceClient(conn)
	userID, err := lookupUserIDByUsername(ctx, client, username)
	if err != nil {
		return failf(stderr, 1, "%v", err)
	}
	recipientUserID, err := lookupUserIDByUsername(ctx, client, recipientUsername)
	if err != nil {
		return failf(stderr, 1, "%v", err)
	}

	createSessionResp, err := client.CreateWebSession(ctx, &authv1.CreateWebSessionRequest{
		UserId:     userID,
		TtlSeconds: cfg.ttlSeconds,
	})
	if err != nil {
		return failf(stderr, 1, "create web session for user %q: %v", userID, err)
	}

	sessionID := strings.TrimSpace(createSessionResp.GetSession().GetId())
	if sessionID == "" {
		return failf(stderr, 1, "create web session for user %q: missing session id", userID)
	}

	fmt.Fprintf(stdout, "WEB_SMOKE_AUTH_USERNAME=%s\n", username)
	fmt.Fprintf(stdout, "WEB_SMOKE_USER_ID=%s\n", userID)
	fmt.Fprintf(stdout, "WEB_SMOKE_AUTH_RECIPIENT_USERNAME=%s\n", recipientUsername)
	fmt.Fprintf(stdout, "WEB_SMOKE_RECIPIENT_USER_ID=%s\n", recipientUserID)
	fmt.Fprintf(stdout, "WEB_SMOKE_SESSION_ID=%s\n", sessionID)
	return nil
}

// parseConfig parses CLI flags into a config.
func parseConfig(args []string, stderr io.Writer) (config, error) {
	var cfg config
	fs := flag.NewFlagSet("websmokeauth", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&cfg.authAddr, "auth-addr", "127.0.0.1:8083", "auth gRPC address")
	fs.StringVar(&cfg.username, "username", "", "existing username to create a session for")
	fs.StringVar(&cfg.recipientUsername, "recipient-username", "", "existing invite recipient username")
	fs.Int64Var(&cfg.ttlSeconds, "ttl-seconds", 3600, "web session ttl in seconds")
	fs.DurationVar(&cfg.requestTimeout, "timeout", 10*time.Second, "request timeout for auth RPC calls")
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	return cfg, nil
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

func lookupUserIDByUsername(ctx context.Context, client authv1.AuthServiceClient, username string) (string, error) {
	resp, err := client.LookupUserByUsername(ctx, &authv1.LookupUserByUsernameRequest{Username: username})
	if err != nil {
		return "", fmt.Errorf("lookup auth user %q: %w", username, err)
	}
	userID := strings.TrimSpace(resp.GetUser().GetId())
	if userID == "" {
		return "", fmt.Errorf("lookup auth user %q: missing user id", username)
	}
	return userID, nil
}

type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return fmt.Sprintf("exit code %d", e.code)
}

func (e *exitError) Unwrap() error {
	return e.err
}

func (e *exitError) ExitCode() int {
	return e.code
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return 1
}

func failf(stderr io.Writer, code int, format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(stderr, "websmokeauth: %s\n", msg)
	return &exitError{code: code, err: errors.New(msg)}
}
