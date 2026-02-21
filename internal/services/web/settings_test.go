package web

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"golang.org/x/text/message"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAppSettingsRouteRedirectsUnauthenticatedToLogin(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

func TestAppSettingsPageRendersForAuthenticatedUser(t *testing.T) {
	h := &handler{
		config:       Config{AuthBaseURL: "http://auth.local"},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppSettings(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "AI Keys") {
		t.Fatalf("body should include settings ai keys entry, got %q", body)
	}
	if !strings.Contains(body, "menu-title") {
		t.Fatalf("body should include sidebar menu title styling, got %q", body)
	}
}

func TestAppAIKeysPageRendersUnavailableStateWhenCredentialServiceMissing(t *testing.T) {
	h := &handler{
		config:       Config{AuthBaseURL: "http://auth.local"},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/settings/ai-keys", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppAIKeys(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "AI key service is currently unavailable.") {
		t.Fatalf("body should include unavailable warning, got %q", body)
	}
}

func TestAppAIKeysPageListsCredentialsForAuthenticatedUser(t *testing.T) {
	fakeClient := &fakeCredentialClient{
		listResp: &aiv1.ListCredentialsResponse{
			Credentials: []*aiv1.Credential{
				{
					Id:        "cred-1",
					Label:     "Primary",
					Provider:  aiv1.Provider_PROVIDER_OPENAI,
					Status:    aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE,
					CreatedAt: timestamppb.New(time.Date(2026, 2, 20, 15, 0, 0, 0, time.UTC)),
				},
			},
		},
	}
	h := &handler{
		config:            Config{AuthBaseURL: "http://auth.local"},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		credentialClient:  fakeClient,
		campaignNameCache: map[string]campaignNameCache{},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	sess := h.sessions.get(sessionID, "token-1")
	sess.cachedUserID = "user-1"
	sess.cachedUserIDResolved = true

	req := httptest.NewRequest(http.MethodGet, "/settings/ai-keys", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppAIKeys(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if fakeClient.lastListReq == nil {
		t.Fatal("expected ListCredentials call")
	}
	if got := fakeClient.lastListUserID; got != "user-1" {
		t.Fatalf("list user metadata = %q, want %q", got, "user-1")
	}
	body := w.Body.String()
	if !strings.Contains(body, "Primary") {
		t.Fatalf("body should include credential label, got %q", body)
	}
	if !strings.Contains(body, `href="/settings">Settings</a>`) {
		t.Fatalf("body should include settings breadcrumb link, got %q", body)
	}
	if !strings.Contains(body, "Are you sure you want to revoke this key?") {
		t.Fatalf("body should include localized revoke confirm message, got %q", body)
	}
}

func TestAppAIKeysPageLogsListErrorsAndRendersUnavailableState(t *testing.T) {
	fakeClient := &fakeCredentialClient{
		listErr: status.Error(codes.Unavailable, "upstream unavailable"),
	}
	h := &handler{
		config:            Config{AuthBaseURL: "http://auth.local"},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		credentialClient:  fakeClient,
		campaignNameCache: map[string]campaignNameCache{},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	sess := h.sessions.get(sessionID, "token-1")
	sess.cachedUserID = "user-1"
	sess.cachedUserIDResolved = true

	var logBuffer bytes.Buffer
	originalLogWriter := log.Writer()
	originalLogFlags := log.Flags()
	originalLogPrefix := log.Prefix()
	log.SetOutput(&logBuffer)
	log.SetFlags(0)
	log.SetPrefix("")
	defer func() {
		log.SetOutput(originalLogWriter)
		log.SetFlags(originalLogFlags)
		log.SetPrefix(originalLogPrefix)
	}()

	req := httptest.NewRequest(http.MethodGet, "/settings/ai-keys", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppAIKeys(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if fakeClient.lastListReq == nil {
		t.Fatal("expected ListCredentials call")
	}
	if got := fakeClient.lastListUserID; got != "user-1" {
		t.Fatalf("list user metadata = %q, want %q", got, "user-1")
	}
	body := w.Body.String()
	if !strings.Contains(body, "AI key service is currently unavailable.") {
		t.Fatalf("body should include unavailable warning, got %q", body)
	}
	if gotLog := logBuffer.String(); !strings.Contains(gotLog, "list ai credentials failed") {
		t.Fatalf("log should include failure marker, got %q", gotLog)
	}
	if gotLog := logBuffer.String(); !strings.Contains(gotLog, "user_id=user-1") {
		t.Fatalf("log should include user id context, got %q", gotLog)
	}
}

func TestAppAIKeysCreateCreatesCredentialAndRedirects(t *testing.T) {
	fakeClient := &fakeCredentialClient{
		createResp: &aiv1.CreateCredentialResponse{
			Credential: &aiv1.Credential{
				Id:       "cred-1",
				Label:    "Primary",
				Provider: aiv1.Provider_PROVIDER_OPENAI,
				Status:   aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE,
			},
		},
	}
	h := &handler{
		config:            Config{AuthBaseURL: "http://auth.local"},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		credentialClient:  fakeClient,
		campaignNameCache: map[string]campaignNameCache{},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	sess := h.sessions.get(sessionID, "token-1")
	sess.cachedUserID = "user-1"
	sess.cachedUserIDResolved = true

	form := url.Values{}
	form.Set("label", "Primary")
	form.Set("secret", "sk-test-1")
	req := httptest.NewRequest(http.MethodPost, "/settings/ai-keys", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppAIKeys(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/settings/ai-keys" {
		t.Fatalf("location = %q, want %q", location, "/settings/ai-keys")
	}
	if fakeClient.lastCreateReq == nil {
		t.Fatal("expected CreateCredential call")
	}
	if got := fakeClient.lastCreateReq.GetProvider(); got != aiv1.Provider_PROVIDER_OPENAI {
		t.Fatalf("provider = %v, want %v", got, aiv1.Provider_PROVIDER_OPENAI)
	}
	if got := fakeClient.lastCreateReq.GetLabel(); got != "Primary" {
		t.Fatalf("label = %q, want %q", got, "Primary")
	}
	if got := fakeClient.lastCreateReq.GetSecret(); got != "sk-test-1" {
		t.Fatalf("secret = %q, want %q", got, "sk-test-1")
	}
	if got := fakeClient.lastCreateUserID; got != "user-1" {
		t.Fatalf("create user metadata = %q, want %q", got, "user-1")
	}
}

func TestAppAIKeysRevokeRevokesCredentialAndRedirects(t *testing.T) {
	fakeClient := &fakeCredentialClient{
		revokeResp: &aiv1.RevokeCredentialResponse{
			Credential: &aiv1.Credential{
				Id:     "cred-1",
				Status: aiv1.CredentialStatus_CREDENTIAL_STATUS_REVOKED,
			},
		},
	}
	h := &handler{
		config:            Config{AuthBaseURL: "http://auth.local"},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		credentialClient:  fakeClient,
		campaignNameCache: map[string]campaignNameCache{},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	sess := h.sessions.get(sessionID, "token-1")
	sess.cachedUserID = "user-1"
	sess.cachedUserIDResolved = true

	req := httptest.NewRequest(http.MethodPost, "/settings/ai-keys/cred-1/revoke", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppSettingsRoutes(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/settings/ai-keys" {
		t.Fatalf("location = %q, want %q", location, "/settings/ai-keys")
	}
	if fakeClient.lastRevokeReq == nil {
		t.Fatal("expected RevokeCredential call")
	}
	if got := fakeClient.lastRevokeReq.GetCredentialId(); got != "cred-1" {
		t.Fatalf("credential_id = %q, want %q", got, "cred-1")
	}
	if got := fakeClient.lastRevokeUserID; got != "user-1" {
		t.Fatalf("revoke user metadata = %q, want %q", got, "user-1")
	}
}

type fakeCredentialClient struct {
	listResp   *aiv1.ListCredentialsResponse
	listErr    error
	createResp *aiv1.CreateCredentialResponse
	createErr  error
	revokeResp *aiv1.RevokeCredentialResponse
	revokeErr  error

	lastListReq      *aiv1.ListCredentialsRequest
	lastListUserID   string
	lastCreateReq    *aiv1.CreateCredentialRequest
	lastCreateUserID string
	lastRevokeReq    *aiv1.RevokeCredentialRequest
	lastRevokeUserID string
}

func (f *fakeCredentialClient) ListCredentials(ctx context.Context, in *aiv1.ListCredentialsRequest, _ ...grpc.CallOption) (*aiv1.ListCredentialsResponse, error) {
	f.lastListReq = in
	md, _ := metadata.FromOutgoingContext(ctx)
	if values := md.Get("x-fracturing-space-user-id"); len(values) > 0 {
		f.lastListUserID = strings.TrimSpace(values[0])
	}
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listResp != nil {
		return f.listResp, nil
	}
	return &aiv1.ListCredentialsResponse{}, nil
}

func (f *fakeCredentialClient) CreateCredential(ctx context.Context, in *aiv1.CreateCredentialRequest, _ ...grpc.CallOption) (*aiv1.CreateCredentialResponse, error) {
	f.lastCreateReq = in
	md, _ := metadata.FromOutgoingContext(ctx)
	if values := md.Get("x-fracturing-space-user-id"); len(values) > 0 {
		f.lastCreateUserID = strings.TrimSpace(values[0])
	}
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createResp != nil {
		return f.createResp, nil
	}
	return &aiv1.CreateCredentialResponse{}, nil
}

func (f *fakeCredentialClient) RevokeCredential(ctx context.Context, in *aiv1.RevokeCredentialRequest, _ ...grpc.CallOption) (*aiv1.RevokeCredentialResponse, error) {
	f.lastRevokeReq = in
	md, _ := metadata.FromOutgoingContext(ctx)
	if values := md.Get("x-fracturing-space-user-id"); len(values) > 0 {
		f.lastRevokeUserID = strings.TrimSpace(values[0])
	}
	if f.revokeErr != nil {
		return nil, f.revokeErr
	}
	if f.revokeResp != nil {
		return f.revokeResp, nil
	}
	return &aiv1.RevokeCredentialResponse{}, nil
}

type testLocalizer struct{}

func (testLocalizer) Sprintf(key message.Reference, _ ...any) string {
	if text, ok := key.(string); ok {
		return text
	}
	return ""
}

func TestToAIKeyRowsDisablesRevokeForUnsafeCredentialID(t *testing.T) {
	rows := toAIKeyRows(testLocalizer{}, []*aiv1.Credential{
		{
			Id:       "cred/unsafe",
			Label:    "Primary",
			Provider: aiv1.Provider_PROVIDER_OPENAI,
			Status:   aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE,
		},
	})

	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if rows[0].ID != "" {
		t.Fatalf("row.ID = %q, want empty for unsafe path id", rows[0].ID)
	}
	if rows[0].CanRevoke {
		t.Fatal("row.CanRevoke = true, want false for unsafe path id")
	}
}

func TestAppAIKeysCreateRendersErrorOnInvalidInput(t *testing.T) {
	fakeClient := &fakeCredentialClient{
		createErr: status.Error(codes.InvalidArgument, "label is required"),
	}
	h := &handler{
		config:            Config{AuthBaseURL: "http://auth.local"},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		credentialClient:  fakeClient,
		campaignNameCache: map[string]campaignNameCache{},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	sess := h.sessions.get(sessionID, "token-1")
	sess.cachedUserID = "user-1"
	sess.cachedUserIDResolved = true

	form := url.Values{}
	form.Set("label", "")
	form.Set("secret", "")
	req := httptest.NewRequest(http.MethodPost, "/settings/ai-keys", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppAIKeys(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
