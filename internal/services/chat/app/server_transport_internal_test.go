package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewHandlerWithAuthorizerRequiresAuthenticationCookie(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)

	NewHandlerWithAuthorizer(fakeWSAuthorizer{userID: "user-1"}).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestNewHandlerWithAuthorizerRequiresConfiguredAuthorizer(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.AddCookie(&http.Cookie{Name: tokenCookieName, Value: "token-1"})

	NewHandlerWithAuthorizer(nil).ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestAccessTokenFromRequest(t *testing.T) {
	t.Parallel()

	if got := accessTokenFromRequest(nil); got != "" {
		t.Fatalf("nil request token = %q, want empty", got)
	}

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.AddCookie(&http.Cookie{Name: webSessionCookieName, Value: "sess-1"})
	if got := accessTokenFromRequest(req); got != webSessionTokenPrefix+"sess-1" {
		t.Fatalf("web session token = %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.AddCookie(&http.Cookie{Name: webSessionCookieName, Value: "sess-1"})
	req.AddCookie(&http.Cookie{Name: tokenCookieName, Value: "token-1"})
	if got := accessTokenFromRequest(req); got != "token-1" {
		t.Fatalf("fs token precedence = %q", got)
	}
}

func TestWriteWSRPCErrorAndCodeMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		err           error
		wantCode      string
		wantRetryable bool
	}{
		{name: "invalid argument", err: status.Error(codes.InvalidArgument, "bad payload"), wantCode: "INVALID_ARGUMENT"},
		{name: "permission denied", err: status.Error(codes.PermissionDenied, "nope"), wantCode: "FORBIDDEN"},
		{name: "unavailable", err: status.Error(codes.Unavailable, "try later"), wantCode: "UNAVAILABLE", wantRetryable: true},
		{name: "deadline exceeded", err: status.Error(codes.DeadlineExceeded, "timeout"), wantCode: "UNAVAILABLE", wantRetryable: true},
		{name: "internal fallback", err: status.Error(codes.Internal, "boom"), wantCode: "INTERNAL"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			peer := newWSPeer(json.NewEncoder(&buf))
			if err := writeWSRPCError(peer, "req-1", tc.err); err != nil {
				t.Fatalf("writeWSRPCError() error = %v", err)
			}

			var frame wsFrame
			if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &frame); err != nil {
				t.Fatalf("json.Unmarshal(frame) error = %v", err)
			}
			if frame.Type != "chat.error" || frame.RequestID != "req-1" {
				t.Fatalf("frame = %+v", frame)
			}

			var payload wsErrorEnvelope
			if err := json.Unmarshal(frame.Payload, &payload); err != nil {
				t.Fatalf("json.Unmarshal(payload) error = %v", err)
			}
			if payload.Error.Code != tc.wantCode {
				t.Fatalf("payload.Error.Code = %q, want %q", payload.Error.Code, tc.wantCode)
			}
			if payload.Error.Retryable != tc.wantRetryable {
				t.Fatalf("payload.Error.Retryable = %v, want %v", payload.Error.Retryable, tc.wantRetryable)
			}
		})
	}
}

func TestHandleHistoryBeforeFrameRequiresJoinedRoom(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	session := newWSSession("user-1", newWSPeer(json.NewEncoder(&buf)))

	handleHistoryBeforeFrame(session, wsFrame{
		Type:      "chat.history.before",
		RequestID: "req-1",
		Payload:   mustJSON(historyBeforePayload{BeforeSequenceID: 1, Limit: 10}),
	})

	var frame wsFrame
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &frame); err != nil {
		t.Fatalf("json.Unmarshal(frame) error = %v", err)
	}
	var payload wsErrorEnvelope
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		t.Fatalf("json.Unmarshal(payload) error = %v", err)
	}
	if payload.Error.Code != "FORBIDDEN" {
		t.Fatalf("payload.Error.Code = %q, want FORBIDDEN", payload.Error.Code)
	}
}
