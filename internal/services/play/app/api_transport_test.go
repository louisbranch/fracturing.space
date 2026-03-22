package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	playprotocol "github.com/louisbranch/fracturing.space/internal/services/play/protocol"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestParseChatHistoryPage(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()

		page, err := parseChatHistoryPage(httptest.NewRequest(http.MethodGet, "/api/campaigns/c1/chat/history", nil))
		if err != nil {
			t.Fatalf("parseChatHistoryPage(defaults) error = %v", err)
		}
		if page.BeforeSequenceID != 1<<62 {
			t.Fatalf("BeforeSequenceID = %d, want %d", page.BeforeSequenceID, int64(1<<62))
		}
		if page.Limit != transcript.DefaultHistoryLimit {
			t.Fatalf("Limit = %d, want %d", page.Limit, transcript.DefaultHistoryLimit)
		}
	})

	t.Run("explicit values", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/api/campaigns/c1/chat/history?before_seq=9&limit=2", nil)
		page, err := parseChatHistoryPage(req)
		if err != nil {
			t.Fatalf("parseChatHistoryPage(explicit values) error = %v", err)
		}
		if page.BeforeSequenceID != 9 || page.Limit != 2 {
			t.Fatalf("page = %#v", page)
		}
	})

	t.Run("invalid before sequence", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/api/campaigns/c1/chat/history?before_seq=oops", nil)
		if _, err := parseChatHistoryPage(req); err != errInvalidBeforeSequence {
			t.Fatalf("parseChatHistoryPage(invalid before) error = %v, want %v", err, errInvalidBeforeSequence)
		}
	})

	t.Run("invalid limit", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/api/campaigns/c1/chat/history?limit=oops", nil)
		if _, err := parseChatHistoryPage(req); err != errInvalidLimit {
			t.Fatalf("parseChatHistoryPage(invalid limit) error = %v, want %v", err, errInvalidLimit)
		}
	})
}

func TestRequirePlayRequest(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		server := &Server{auth: &fakePlayAuthClient{sessions: map[string]string{"ps-1": "user-1"}}}
		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/bootstrap", nil)
		req.SetPathValue("campaignID", "c1")
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		got, ok := server.requirePlayRequest(rr, req)
		if !ok {
			t.Fatal("requirePlayRequest() reported failure")
		}
		if got.CampaignID != "c1" || got.UserID != "user-1" {
			t.Fatalf("request = %#v", got)
		}
	})

	t.Run("missing campaign id returns not found", func(t *testing.T) {
		t.Parallel()

		server := &Server{}
		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/bootstrap", nil)
		rr := httptest.NewRecorder()

		if _, ok := server.requirePlayRequest(rr, req); ok {
			t.Fatal("requirePlayRequest() unexpectedly succeeded")
		}
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("missing play session returns unauthorized", func(t *testing.T) {
		t.Parallel()

		server := &Server{auth: &fakePlayAuthClient{}}
		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/bootstrap", nil)
		req.SetPathValue("campaignID", "c1")
		rr := httptest.NewRecorder()

		if _, ok := server.requirePlayRequest(rr, req); ok {
			t.Fatal("requirePlayRequest() unexpectedly succeeded")
		}
		assertJSONError(t, rr, http.StatusUnauthorized, "authentication required")
	})
}

func TestHandleChatHistoryVariants(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		interaction := newRecordingInteractionClient(playTestState())
		transcripts := &scriptTranscriptStore{
			latest: 7,
			before: []transcript.Message{{
				MessageID:  "m1",
				CampaignID: "c1",
				SessionID:  "s1",
				SequenceID: 4,
				SentAt:     "2026-03-13T12:00:00Z",
				Actor: transcript.MessageActor{
					ParticipantID: "p1",
					Name:          "Avery",
				},
				Body:            "Hello",
				ClientMessageID: "cm-1",
			}},
		}
		server := newAuthedPlayServer(interaction, transcripts)

		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/chat/history?before_seq=9&limit=2", nil)
		req.SetPathValue("campaignID", "c1")
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		server.handleChatHistory(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
		var payload playprotocol.HistoryResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode history response: %v", err)
		}
		if payload.SessionID != "s1" {
			t.Fatalf("session_id = %q, want %q", payload.SessionID, "s1")
		}
		if len(payload.Messages) != 1 || payload.Messages[0].MessageID != "m1" {
			t.Fatalf("messages = %#v", payload.Messages)
		}
		if transcripts.beforeArgs.before != 9 {
			t.Fatalf("before_seq = %d, want %d", transcripts.beforeArgs.before, 9)
		}
		if transcripts.beforeArgs.limit != 2 {
			t.Fatalf("limit = %d, want %d", transcripts.beforeArgs.limit, 2)
		}
		if transcripts.beforeArgs.scope.CampaignID != "c1" || transcripts.beforeArgs.scope.SessionID != "s1" {
			t.Fatalf("history scope = %#v, want campaign c1 session s1", transcripts.beforeArgs.scope)
		}
	})

	t.Run("invalid before sequence", func(t *testing.T) {
		t.Parallel()

		server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/chat/history?before_seq=oops", nil)
		req.SetPathValue("campaignID", "c1")
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		server.handleChatHistory(rr, req)

		assertJSONError(t, rr, http.StatusBadRequest, "invalid before_seq")
	})

	t.Run("invalid limit", func(t *testing.T) {
		t.Parallel()

		server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/chat/history?limit=oops", nil)
		req.SetPathValue("campaignID", "c1")
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		server.handleChatHistory(rr, req)

		assertJSONError(t, rr, http.StatusBadRequest, "invalid limit")
	})

	t.Run("missing active session returns empty payload", func(t *testing.T) {
		t.Parallel()

		state := playTestState()
		state.ActiveSession = nil
		server := newAuthedPlayServer(newRecordingInteractionClient(state), &scriptTranscriptStore{})
		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/chat/history", nil)
		req.SetPathValue("campaignID", "c1")
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		server.handleChatHistory(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
		var payload playprotocol.HistoryResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode history response: %v", err)
		}
		if payload.SessionID != "" || len(payload.Messages) != 0 {
			t.Fatalf("payload = %#v", payload)
		}
	})
}

func TestHandleAIDebugVariants(t *testing.T) {
	t.Parallel()

	t.Run("list turns", func(t *testing.T) {
		t.Parallel()

		aiDebug := &fakePlayAIDebugClient{
			listResp: &aiv1.ListCampaignDebugTurnsResponse{
				Turns: []*aiv1.CampaignDebugTurnSummary{{
					Id:         "turn-1",
					Model:      "gpt-4.1-mini",
					Status:     aiv1.CampaignDebugTurnStatus_CAMPAIGN_DEBUG_TURN_STATUS_RUNNING,
					EntryCount: 3,
				}},
				NextPageToken: "next-1",
			},
		}
		server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
		server.aiDebug = aiDebug

		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/ai-debug/turns?page_size=10", nil)
		req.SetPathValue("campaignID", "c1")
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		server.handleAIDebugTurns(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
		}
		var payload playprotocol.AIDebugTurnsPage
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode ai debug list: %v", err)
		}
		if len(payload.Turns) != 1 || payload.Turns[0].ID != "turn-1" {
			t.Fatalf("turns = %#v", payload.Turns)
		}
		if aiDebug.listReq == nil || aiDebug.listReq.GetSessionId() != "s1" {
			t.Fatalf("list request = %#v", aiDebug.listReq)
		}
	})

	t.Run("get turn", func(t *testing.T) {
		t.Parallel()

		aiDebug := &fakePlayAIDebugClient{
			getResp: &aiv1.GetCampaignDebugTurnResponse{
				Turn: &aiv1.CampaignDebugTurn{
					Id:     "turn-1",
					Model:  "gpt-4.1-mini",
					Status: aiv1.CampaignDebugTurnStatus_CAMPAIGN_DEBUG_TURN_STATUS_FAILED,
					Entries: []*aiv1.CampaignDebugEntry{{
						Sequence: 1,
						Kind:     aiv1.CampaignDebugEntryKind_CAMPAIGN_DEBUG_ENTRY_KIND_TOOL_RESULT,
						ToolName: "scene_create",
						Payload:  "tool failed",
						IsError:  true,
					}},
				},
			},
		}
		server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
		server.aiDebug = aiDebug

		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/ai-debug/turns/turn-1", nil)
		req.SetPathValue("campaignID", "c1")
		req.SetPathValue("turnID", "turn-1")
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		server.handleAIDebugTurn(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
		}
		var payload playprotocol.AIDebugTurn
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode ai debug turn: %v", err)
		}
		if payload.ID != "turn-1" || len(payload.Entries) != 1 {
			t.Fatalf("payload = %#v", payload)
		}
		if aiDebug.getReq == nil || aiDebug.getReq.GetTurnId() != "turn-1" {
			t.Fatalf("get request = %#v", aiDebug.getReq)
		}
	})

	t.Run("invalid page size", func(t *testing.T) {
		t.Parallel()

		server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/ai-debug/turns?page_size=oops", nil)
		req.SetPathValue("campaignID", "c1")
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		server.handleAIDebugTurns(rr, req)

		assertJSONError(t, rr, http.StatusBadRequest, "invalid page_size")
	})

	t.Run("list turns without active session returns empty payload", func(t *testing.T) {
		t.Parallel()

		state := playTestState()
		state.ActiveSession = nil
		aiDebug := &fakePlayAIDebugClient{}
		server := newAuthedPlayServer(newRecordingInteractionClient(state), &scriptTranscriptStore{})
		server.aiDebug = aiDebug

		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/ai-debug/turns", nil)
		req.SetPathValue("campaignID", "c1")
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		server.handleAIDebugTurns(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
		}
		var payload playprotocol.AIDebugTurnsPage
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode ai debug list: %v", err)
		}
		if len(payload.Turns) != 0 || aiDebug.listReq != nil {
			t.Fatalf("payload/listReq = (%#v, %#v), want empty result without upstream call", payload, aiDebug.listReq)
		}
	})

	t.Run("list turns maps upstream error", func(t *testing.T) {
		t.Parallel()

		aiDebug := &fakePlayAIDebugClient{listErr: status.Error(codes.Unavailable, "down")}
		server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
		server.aiDebug = aiDebug

		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/ai-debug/turns", nil)
		req.SetPathValue("campaignID", "c1")
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		server.handleAIDebugTurns(rr, req)

		assertJSONError(t, rr, http.StatusBadGateway, "upstream request failed")
	})

	t.Run("missing turn id returns not found", func(t *testing.T) {
		t.Parallel()

		server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/ai-debug/turns/", nil)
		req.SetPathValue("campaignID", "c1")
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		server.handleAIDebugTurn(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("get turn maps upstream error", func(t *testing.T) {
		t.Parallel()

		aiDebug := &fakePlayAIDebugClient{getErr: status.Error(codes.NotFound, "missing")}
		server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
		server.aiDebug = aiDebug

		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/ai-debug/turns/turn-1", nil)
		req.SetPathValue("campaignID", "c1")
		req.SetPathValue("turnID", "turn-1")
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		server.handleAIDebugTurn(rr, req)

		assertJSONError(t, rr, http.StatusNotFound, "resource not found")
	})
}

func TestParseAIDebugPage(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()

		page, err := parseAIDebugPage(httptest.NewRequest(http.MethodGet, "/api/campaigns/c1/ai-debug/turns", nil))
		if err != nil {
			t.Fatalf("parseAIDebugPage(defaults) error = %v", err)
		}
		if page.PageSize != 20 || page.PageToken != "" {
			t.Fatalf("page = %#v", page)
		}
	})

	t.Run("clamps page size", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/api/campaigns/c1/ai-debug/turns?page_size=200&page_token=next-1", nil)
		page, err := parseAIDebugPage(req)
		if err != nil {
			t.Fatalf("parseAIDebugPage(clamps) error = %v", err)
		}
		if page.PageSize != 50 || page.PageToken != "next-1" {
			t.Fatalf("page = %#v", page)
		}
	})

	t.Run("rejects non-positive page size", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/api/campaigns/c1/ai-debug/turns?page_size=0", nil)
		if _, err := parseAIDebugPage(req); err != errInvalidAIDebugPageSize {
			t.Fatalf("parseAIDebugPage(non-positive) error = %v, want %v", err, errInvalidAIDebugPageSize)
		}
	})
}
