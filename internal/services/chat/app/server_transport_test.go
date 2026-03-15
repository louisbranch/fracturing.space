package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gogrpccodes "google.golang.org/grpc/codes"
	gogrpcstatus "google.golang.org/grpc/status"
)

func TestNewHandlerWithAuthorizerRejectsMissingCookies(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)

	NewHandlerWithAuthorizer(fakeWSAuthorizer{userID: "user-1"}).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestNewHandlerWithAuthorizerAllowsAuthenticatedJoin(t *testing.T) {
	t.Parallel()

	conn := openWSConn(t, NewHandlerWithAuthorizer(fakeWSAuthorizer{
		tokenToUser: map[string]string{
			"token-a": "user-a",
		},
		welcomeByKey: map[string]joinWelcome{
			"camp-1::sess-1::user-a": {
				ParticipantID:   "part-a",
				ParticipantName: "Ari",
				CampaignName:    "Guildhouse",
				SessionID:       "sess-1",
				SessionName:     "One",
			},
		},
	}), "fs_token=token-a")

	conn.writeFrame(t, map[string]any{
		"type": "chat.join",
		"payload": map[string]any{
			"campaign_id": "camp-1",
			"session_id":  "sess-1",
		},
	})

	frame := conn.readFrame(t)
	if frame.Type != "chat.joined" {
		t.Fatalf("frame.Type = %q, want chat.joined", frame.Type)
	}
}

func TestAccessTokenFromRequestPrefersFSTokenAndFallsBackToWebSession(t *testing.T) {
	t.Parallel()

	if got := accessTokenFromRequest(nil); got != "" {
		t.Fatalf("accessTokenFromRequest(nil) = %q, want empty", got)
	}

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.AddCookie(&http.Cookie{Name: tokenCookieName, Value: " primary "})
	req.AddCookie(&http.Cookie{Name: webSessionCookieName, Value: "secondary"})
	if got := accessTokenFromRequest(req); got != "primary" {
		t.Fatalf("token = %q, want primary", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.AddCookie(&http.Cookie{Name: tokenCookieName, Value: " "})
	req.AddCookie(&http.Cookie{Name: webSessionCookieName, Value: " session-1 "})
	if got := accessTokenFromRequest(req); got != webSessionTokenPrefix+"session-1" {
		t.Fatalf("token = %q, want web session token", got)
	}
}

func TestWriteWSRPCErrorMapsCodeAndRetryableFlag(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	peer := newWSPeer(json.NewEncoder(&buf))

	if err := writeWSRPCError(peer, "req-1", gogrpcstatus.Error(gogrpccodes.Unavailable, "down")); err != nil {
		t.Fatalf("writeWSRPCError() error = %v", err)
	}

	var frame wsFrame
	if err := json.NewDecoder(&buf).Decode(&frame); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if frame.Type != "chat.error" || frame.RequestID != "req-1" {
		t.Fatalf("frame = %#v", frame)
	}

	var payload wsErrorEnvelope
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Error.Code != "UNAVAILABLE" || !payload.Error.Retryable {
		t.Fatalf("error payload = %+v", payload.Error)
	}
}

func TestWSErrorCodeFromRPCMapsExpectedCodes(t *testing.T) {
	t.Parallel()

	tests := map[gogrpccodes.Code]string{
		gogrpccodes.InvalidArgument:    "INVALID_ARGUMENT",
		gogrpccodes.PermissionDenied:   "FORBIDDEN",
		gogrpccodes.NotFound:           "NOT_FOUND",
		gogrpccodes.FailedPrecondition: "FAILED_PRECONDITION",
		gogrpccodes.ResourceExhausted:  "RESOURCE_EXHAUSTED",
		gogrpccodes.DeadlineExceeded:   "UNAVAILABLE",
		gogrpccodes.Unavailable:        "UNAVAILABLE",
		gogrpccodes.Internal:           "INTERNAL",
	}

	for input, want := range tests {
		if got := wsErrorCodeFromRPC(input); got != want {
			t.Fatalf("wsErrorCodeFromRPC(%v) = %q, want %q", input, got, want)
		}
	}
}

func TestWebSocketHistoryBeforeRejectsInvalidState(t *testing.T) {
	t.Parallel()

	t.Run("requires valid before sequence", func(t *testing.T) {
		t.Parallel()

		conn := openWSConn(t, NewHandler())
		conn.writeFrame(t, map[string]any{
			"type":       "chat.history.before",
			"request_id": "req-1",
			"payload": map[string]any{
				"before_sequence_id": 0,
			},
		})

		frame := conn.readFrame(t)
		if frame.Type != "chat.error" {
			t.Fatalf("frame.Type = %q, want chat.error", frame.Type)
		}
		var payload wsErrorEnvelope
		if err := json.Unmarshal(frame.Payload, &payload); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if payload.Error.Message != "before_sequence_id must be >= 1" {
			t.Fatalf("error message = %q", payload.Error.Message)
		}
	})

	t.Run("requires join before history", func(t *testing.T) {
		t.Parallel()

		conn := openWSConn(t, NewHandler())
		conn.writeFrame(t, map[string]any{
			"type":       "chat.history.before",
			"request_id": "req-2",
			"payload": map[string]any{
				"before_sequence_id": 2,
			},
		})

		frame := conn.readFrame(t)
		if frame.Type != "chat.error" {
			t.Fatalf("frame.Type = %q, want chat.error", frame.Type)
		}
		var payload wsErrorEnvelope
		if err := json.Unmarshal(frame.Payload, &payload); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if payload.Error.Code != "FORBIDDEN" {
			t.Fatalf("error code = %q, want FORBIDDEN", payload.Error.Code)
		}
	})
}

func TestWebSocketJoinFrameAuthorizerErrorsMapToTransportErrors(t *testing.T) {
	t.Parallel()

	t.Run("participant required becomes forbidden", func(t *testing.T) {
		t.Parallel()

		conn := openWSConn(t, NewHandlerWithAuthorizer(fakeWSAuthorizer{
			tokenToUser: map[string]string{"token-a": "user-a"},
			resolveErr:  errCampaignParticipantRequired,
		}), "fs_token=token-a")
		conn.writeFrame(t, map[string]any{
			"type": "chat.join",
			"payload": map[string]any{
				"campaign_id": "camp-1",
				"session_id":  "sess-1",
			},
		})

		frame := conn.readFrame(t)
		if frame.Type != "chat.error" {
			t.Fatalf("frame.Type = %q, want chat.error", frame.Type)
		}
		var payload wsErrorEnvelope
		if err := json.Unmarshal(frame.Payload, &payload); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if payload.Error.Code != "FORBIDDEN" {
			t.Fatalf("error code = %q, want FORBIDDEN", payload.Error.Code)
		}
	})

	t.Run("transient lookup failure becomes unavailable", func(t *testing.T) {
		t.Parallel()

		conn := openWSConn(t, NewHandlerWithAuthorizer(fakeWSAuthorizer{
			tokenToUser: map[string]string{"token-a": "user-a"},
			resolveErr:  context.DeadlineExceeded,
		}), "fs_token=token-a")
		conn.writeFrame(t, map[string]any{
			"type": "chat.join",
			"payload": map[string]any{
				"campaign_id": "camp-1",
				"session_id":  "sess-1",
			},
		})

		frame := conn.readFrame(t)
		if frame.Type != "chat.error" {
			t.Fatalf("frame.Type = %q, want chat.error", frame.Type)
		}
		var payload wsErrorEnvelope
		if err := json.Unmarshal(frame.Payload, &payload); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if payload.Error.Code != "UNAVAILABLE" {
			t.Fatalf("error code = %q, want UNAVAILABLE", payload.Error.Code)
		}
	})

	t.Run("grpc permission denied becomes forbidden", func(t *testing.T) {
		t.Parallel()

		conn := openWSConn(t, NewHandlerWithAuthorizer(fakeWSAuthorizer{
			tokenToUser: map[string]string{"token-a": "user-a"},
			resolveErr:  gogrpcstatus.Error(gogrpccodes.PermissionDenied, "no access"),
		}), "fs_token=token-a")
		conn.writeFrame(t, map[string]any{
			"type": "chat.join",
			"payload": map[string]any{
				"campaign_id": "camp-1",
				"session_id":  "sess-1",
			},
		})

		frame := conn.readFrame(t)
		if frame.Type != "chat.error" {
			t.Fatalf("frame.Type = %q, want chat.error", frame.Type)
		}
		var payload wsErrorEnvelope
		if err := json.Unmarshal(frame.Payload, &payload); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if payload.Error.Code != "FORBIDDEN" {
			t.Fatalf("error code = %q, want FORBIDDEN", payload.Error.Code)
		}
	})
}

func TestRunWrapsInitializationErrors(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), Config{})
	if err == nil || !strings.Contains(err.Error(), "init chat server") {
		t.Fatalf("Run() error = %v, want init wrapper", err)
	}
}
