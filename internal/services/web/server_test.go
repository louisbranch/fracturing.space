package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

func TestLoginHandlerRequiresPendingID(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/login?client_id=client-1", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLoginHandlerRendersForm(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/login?pending_id=pending-1&client_id=client-1&client_name=Test+Client", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "pending-1") {
		t.Fatalf("expected pending_id in body")
	}
	if !strings.Contains(body, "http://auth.local/authorize/login") {
		t.Fatalf("expected form action with auth base URL")
	}
	if !strings.Contains(body, "Test Client") {
		t.Fatalf("expected client name in body")
	}
}

func TestLandingPageRenders(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Fracturing.Space") {
		t.Fatalf("expected app name in body")
	}
	if !strings.Contains(body, "Open-source, server-authoritative engine") {
		t.Fatalf("expected hero tagline in body")
	}
}

func TestLandingPageRejectsNonRootPath(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/something", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestLandingPageRejectsNonGETMethod(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestPasskeyLoginStartRequiresClient(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/login/start", bytes.NewBufferString(`{"pending_id":"pending-1"}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestPasskeyLoginStartSuccess(t *testing.T) {
	fake := &fakeAuthClient{
		beginLoginResp: &authv1.BeginPasskeyLoginResponse{
			SessionId:                    "session-1",
			CredentialRequestOptionsJson: []byte(`{"challenge":"test"}`),
		},
	}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/login/start", bytes.NewBufferString(`{"pending_id":"pending-1"}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["session_id"] != "session-1" {
		t.Fatalf("session_id = %v", payload["session_id"])
	}
}

func TestPasskeyLoginFinishRequiresFields(t *testing.T) {
	fake := &fakeAuthClient{}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/login/finish", bytes.NewBufferString(`{"pending_id":"pending-1"}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPasskeyLoginFinishSuccess(t *testing.T) {
	fake := &fakeAuthClient{
		finishLoginResp: &authv1.FinishPasskeyLoginResponse{},
	}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/login/finish", bytes.NewBufferString(`{"pending_id":"pending-1","session_id":"session-1","credential":{}}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["redirect_url"] != "http://auth.local/authorize/consent?pending_id=pending-1" {
		t.Fatalf("redirect_url = %v", payload["redirect_url"])
	}
}

func TestMagicLinkRequiresToken(t *testing.T) {
	fake := &fakeAuthClient{}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodGet, "/magic", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Magic link missing") {
		t.Fatalf("expected error page")
	}
}

func TestMagicLinkRedirectsToConsent(t *testing.T) {
	fake := &fakeAuthClient{
		consumeMagicResp: &authv1.ConsumeMagicLinkResponse{PendingId: "pending-1"},
	}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodGet, "/magic?token=token-1", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "http://auth.local/authorize/consent?pending_id=pending-1" {
		t.Fatalf("location = %q", location)
	}
}

func TestMagicLinkSuccessPage(t *testing.T) {
	fake := &fakeAuthClient{
		consumeMagicResp: &authv1.ConsumeMagicLinkResponse{},
	}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodGet, "/magic?token=token-1", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "Magic link verified") {
		t.Fatalf("expected success page")
	}
}

func TestPasskeyRegisterStartRequiresFields(t *testing.T) {
	fake := &fakeAuthClient{}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/register/start", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPasskeyRegisterStartSuccess(t *testing.T) {
	fake := &fakeAuthClient{
		createUserResp: &authv1.CreateUserResponse{
			User: &authv1.User{Id: "user-1", DisplayName: "Alpha"},
		},
		beginRegResp: &authv1.BeginPasskeyRegistrationResponse{
			SessionId:                     "session-1",
			CredentialCreationOptionsJson: []byte(`{"challenge":"test","user":{"id":"user"}}`),
		},
	}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/register/start", bytes.NewBufferString(`{"display_name":"Alpha"}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["session_id"] != "session-1" {
		t.Fatalf("session_id = %v", payload["session_id"])
	}
	if payload["user_id"] != "user-1" {
		t.Fatalf("user_id = %v", payload["user_id"])
	}
}

func TestPasskeyRegisterFinishRequiresFields(t *testing.T) {
	fake := &fakeAuthClient{}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/register/finish", bytes.NewBufferString(`{"session_id":"session-1"}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPasskeyRegisterFinishSuccess(t *testing.T) {
	fake := &fakeAuthClient{
		finishRegResp: &authv1.FinishPasskeyRegistrationResponse{},
	}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/register/finish", bytes.NewBufferString(`{"session_id":"session-1","user_id":"user-1","credential":{}}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestNewServerRequiresHTTPAddr(t *testing.T) {
	_, err := NewServer(Config{AuthBaseURL: "http://auth.local"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewServerRequiresAuthBaseURL(t *testing.T) {
	_, err := NewServer(Config{HTTPAddr: "127.0.0.1:0"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestDialAuthGRPCNilAddr(t *testing.T) {
	conn, client, err := dialAuthGRPC(context.Background(), Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn != nil || client != nil {
		t.Fatalf("expected nil conn and client")
	}
}

func TestDialAuthGRPCNilContextUsesDefaultTimeout(t *testing.T) {
	listener, server := startGRPCServer(t)
	defer server.Stop()

	conn, client, err := dialAuthGRPC(nil, Config{
		AuthAddr: listener.Addr().String(),
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if conn == nil || client == nil {
		t.Fatalf("expected conn and client")
	}
	_ = conn.Close()
}

func TestDialAuthGRPCSuccess(t *testing.T) {
	listener, server := startGRPCServer(t)
	defer server.Stop()

	conn, client, err := dialAuthGRPC(context.Background(), Config{
		AuthAddr:        listener.Addr().String(),
		GRPCDialTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if conn == nil || client == nil {
		t.Fatalf("expected conn and client")
	}
	_ = conn.Close()
}

func TestDialAuthGRPCDialError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _, err := dialAuthGRPC(ctx, Config{
		AuthAddr:        "127.0.0.1:1",
		GRPCDialTimeout: 50 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "dial auth gRPC") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDialAuthGRPCHealthError(t *testing.T) {
	listener, server := startHealthServer(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	defer server.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, _, err := dialAuthGRPC(ctx, Config{
		AuthAddr:        listener.Addr().String(),
		GRPCDialTimeout: 100 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "auth gRPC health check failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildAuthLoginURL(t *testing.T) {
	cases := []struct {
		name string
		base string
		want string
	}{
		{
			name: "empty base",
			base: "",
			want: "/authorize/login",
		},
		{
			name: "base trims slash",
			base: "http://auth.local/",
			want: "http://auth.local/authorize/login",
		},
		{
			name: "whitespace base",
			base: "  ",
			want: "/authorize/login",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildAuthLoginURL(tc.base); got != tc.want {
				t.Fatalf("buildAuthLoginURL(%q) = %q, want %q", tc.base, got, tc.want)
			}
		})
	}
}

func TestBuildAuthConsentURL(t *testing.T) {
	cases := []struct {
		name      string
		base      string
		pendingID string
		want      string
	}{
		{
			name:      "empty base",
			base:      "",
			pendingID: "pending 1",
			want:      "/authorize/consent?pending_id=pending+1",
		},
		{
			name:      "base trims slash",
			base:      "http://auth.local/",
			pendingID: "pending 1",
			want:      "http://auth.local/authorize/consent?pending_id=pending+1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildAuthConsentURL(tc.base, tc.pendingID); got != tc.want {
				t.Fatalf("buildAuthConsentURL(%q, %q) = %q, want %q", tc.base, tc.pendingID, got, tc.want)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true})

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}
	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want %q", got, "application/json")
	}

	var payload map[string]any
	if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["ok"] != true {
		t.Fatalf("ok = %v, want true", payload["ok"])
	}
}

func TestNewServerSuccessAndClose(t *testing.T) {
	listener, server := startGRPCServer(t)
	defer server.Stop()

	webServer, err := NewServer(Config{
		HTTPAddr:        "127.0.0.1:0",
		AuthBaseURL:     "http://auth.local",
		AuthAddr:        listener.Addr().String(),
		GRPCDialTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	webServer.Close()
}

func TestListenAndServeShutsDown(t *testing.T) {
	webServer, err := NewServer(Config{
		HTTPAddr:        "127.0.0.1:0",
		AuthBaseURL:     "http://auth.local",
		GRPCDialTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- webServer.ListenAndServe(ctx)
	}()

	time.Sleep(30 * time.Millisecond)
	cancel()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timeout waiting for shutdown")
	}
}

func TestListenAndServeReturnsServeError(t *testing.T) {
	server := &Server{
		httpAddr:   "127.0.0.1:-1",
		httpServer: &http.Server{Addr: "127.0.0.1:-1"},
	}

	err := server.ListenAndServe(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "serve http") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type fakeAuthClient struct {
	beginLoginResp   *authv1.BeginPasskeyLoginResponse
	finishLoginResp  *authv1.FinishPasskeyLoginResponse
	createUserResp   *authv1.CreateUserResponse
	beginRegResp     *authv1.BeginPasskeyRegistrationResponse
	finishRegResp    *authv1.FinishPasskeyRegistrationResponse
	consumeMagicResp *authv1.ConsumeMagicLinkResponse
}

func (f *fakeAuthClient) CreateUser(ctx context.Context, req *authv1.CreateUserRequest, opts ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
	if f.createUserResp != nil {
		return f.createUserResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeAuthClient) BeginPasskeyRegistration(ctx context.Context, req *authv1.BeginPasskeyRegistrationRequest, opts ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	if f.beginRegResp != nil {
		return f.beginRegResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeAuthClient) FinishPasskeyRegistration(ctx context.Context, req *authv1.FinishPasskeyRegistrationRequest, opts ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error) {
	if f.finishRegResp != nil {
		return f.finishRegResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeAuthClient) BeginPasskeyLogin(ctx context.Context, req *authv1.BeginPasskeyLoginRequest, opts ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error) {
	if f.beginLoginResp != nil {
		return f.beginLoginResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeAuthClient) FinishPasskeyLogin(ctx context.Context, req *authv1.FinishPasskeyLoginRequest, opts ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error) {
	if f.finishLoginResp != nil {
		return f.finishLoginResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeAuthClient) GenerateMagicLink(ctx context.Context, req *authv1.GenerateMagicLinkRequest, opts ...grpc.CallOption) (*authv1.GenerateMagicLinkResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeAuthClient) ConsumeMagicLink(ctx context.Context, req *authv1.ConsumeMagicLinkRequest, opts ...grpc.CallOption) (*authv1.ConsumeMagicLinkResponse, error) {
	if f.consumeMagicResp != nil {
		return f.consumeMagicResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeAuthClient) ListUserEmails(ctx context.Context, req *authv1.ListUserEmailsRequest, opts ...grpc.CallOption) (*authv1.ListUserEmailsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeAuthClient) IssueJoinGrant(ctx context.Context, req *authv1.IssueJoinGrantRequest, opts ...grpc.CallOption) (*authv1.IssueJoinGrantResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeAuthClient) GetUser(ctx context.Context, req *authv1.GetUserRequest, opts ...grpc.CallOption) (*authv1.GetUserResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeAuthClient) ListUsers(ctx context.Context, req *authv1.ListUsersRequest, opts ...grpc.CallOption) (*authv1.ListUsersResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func TestMagicLinkNilAuthClient(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/magic?token=token-1", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(w.Body.String(), "Magic link unavailable") {
		t.Fatalf("expected unavailable page")
	}
}

func TestPasskeyLoginStartMethodNotAllowed(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/passkeys/login/start", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestPasskeyRegisterFinishMethodNotAllowed(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/passkeys/register/finish", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestLoginHandlerMethodNotAllowed(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/login?pending_id=pending-1", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestPasskeyLoginFinishMethodNotAllowed(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/passkeys/login/finish", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestPasskeyRegisterStartMethodNotAllowed(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/passkeys/register/start", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestPasskeyLoginFinishNilAuthClient(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/login/finish", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestPasskeyRegisterStartNilAuthClient(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/register/start", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestPasskeyRegisterFinishNilAuthClient(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/register/finish", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestMagicLinkInvalidToken(t *testing.T) {
	fake := &fakeAuthClient{}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodGet, "/magic?token=bad-token", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Magic link invalid") {
		t.Fatalf("expected invalid page")
	}
}

func TestPasskeyLoginStartNilAuthClient(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/login/start", bytes.NewBufferString(`{"pending_id":"p1"}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestPasskeyLoginFinishError(t *testing.T) {
	fake := &fakeAuthClient{}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/login/finish",
		bytes.NewBufferString(`{"pending_id":"p1","session_id":"s1","credential":{}}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPasskeyLoginStartError(t *testing.T) {
	fake := &fakeAuthClient{}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/login/start",
		bytes.NewBufferString(`{"pending_id":"p1"}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPasskeyRegisterFinishError(t *testing.T) {
	fake := &fakeAuthClient{}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/register/finish",
		bytes.NewBufferString(`{"session_id":"s1","user_id":"u1","credential":{}}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func startGRPCServer(t *testing.T) (net.Listener, *grpc.Server) {
	return startHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
}

func startHealthServer(t *testing.T, status grpc_health_v1.HealthCheckResponse_ServingStatus) (net.Listener, *grpc.Server) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	server := grpc.NewServer()
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthServer)
	healthServer.SetServingStatus("", status)
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})
	return listener, server
}
