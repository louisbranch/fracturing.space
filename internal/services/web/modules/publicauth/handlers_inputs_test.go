package publicauth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

func TestParseRecoveryStartInputPreservesRawFields(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/login/recovery/start", strings.NewReader(`{"username":" louis ","recovery_code":" code-1 "}`))

	input, err := parseRecoveryStartInput(req)
	if err != nil {
		t.Fatalf("parseRecoveryStartInput() error = %v", err)
	}
	if input.Username != " louis " || input.RecoveryCode != " code-1 " {
		t.Fatalf("input = %+v, want raw values", input)
	}
}

func TestParseRecoveryFinishInputPreservesRawFields(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/login/recovery/finish", strings.NewReader(`{"recovery_session_id":" rec-1 ","session_id":" sess-1 ","pending_id":" pending-1 ","next":" /invite/inv-1 ","credential":{"id":"cred-1"}}`))

	input, err := parseRecoveryFinishInput(req)
	if err != nil {
		t.Fatalf("parseRecoveryFinishInput() error = %v", err)
	}
	if input.RecoverySessionID != " rec-1 " || input.SessionID != " sess-1 " || input.PendingID != " pending-1 " || input.NextPath != "/invite/inv-1" {
		t.Fatalf("input = %+v, want preserved values", input)
	}
	if string(input.Credential) != `{"id":"cred-1"}` {
		t.Fatalf("Credential = %s", string(input.Credential))
	}
}

func TestDecodeJSONBodyStrictRejectsMissingOrInvalidBodies(t *testing.T) {
	t.Parallel()

	var payload map[string]any
	if err := decodeJSONBodyStrict(nil, &payload); apperrors.HTTPStatus(err) != http.StatusBadRequest {
		t.Fatalf("nil request status = %d, want %d", apperrors.HTTPStatus(err), http.StatusBadRequest)
	}

	req := httptest.NewRequest(http.MethodPost, "/login/recovery/start", strings.NewReader(`{"username":`))
	if err := decodeJSONBodyStrict(req, &payload); apperrors.HTTPStatus(err) != http.StatusBadRequest {
		t.Fatalf("invalid json status = %d, want %d", apperrors.HTTPStatus(err), http.StatusBadRequest)
	}
}
