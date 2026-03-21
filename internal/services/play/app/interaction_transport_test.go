package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	playprotocol "github.com/louisbranch/fracturing.space/internal/services/play/protocol"
	gogrpccodes "google.golang.org/grpc/codes"
	gogrpcstatus "google.golang.org/grpc/status"
)

func TestInteractionMutationHandlersProxyRequests(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		path       string
		body       string
		wantMethod string
	}{
		{name: "set active scene", path: "/api/campaigns/c1/interaction/set-active-scene", body: `{}`, wantMethod: "SetActiveScene"},
		{name: "start scene player phase", path: "/api/campaigns/c1/interaction/start-scene-player-phase", body: `{}`, wantMethod: "StartScenePlayerPhase"},
		{name: "submit scene player post", path: "/api/campaigns/c1/interaction/submit-scene-player-post", body: `{}`, wantMethod: "SubmitScenePlayerPost"},
		{name: "yield scene player phase", path: "/api/campaigns/c1/interaction/yield-scene-player-phase", body: `{}`, wantMethod: "YieldScenePlayerPhase"},
		{name: "unyield scene player phase", path: "/api/campaigns/c1/interaction/unyield-scene-player-phase", body: `{}`, wantMethod: "UnyieldScenePlayerPhase"},
		{name: "end scene player phase", path: "/api/campaigns/c1/interaction/end-scene-player-phase", body: `{}`, wantMethod: "EndScenePlayerPhase"},
		{name: "commit scene gm output", path: "/api/campaigns/c1/interaction/commit-scene-gm-output", body: `{}`, wantMethod: "CommitSceneGMOutput"},
		{name: "accept scene player phase", path: "/api/campaigns/c1/interaction/accept-scene-player-phase", body: `{}`, wantMethod: "AcceptScenePlayerPhase"},
		{name: "request scene player revisions", path: "/api/campaigns/c1/interaction/request-scene-player-revisions", body: `{}`, wantMethod: "RequestScenePlayerRevisions"},
		{name: "pause session for ooc", path: "/api/campaigns/c1/interaction/pause-session-for-ooc", body: `{}`, wantMethod: "PauseSessionForOOC"},
		{name: "post session ooc", path: "/api/campaigns/c1/interaction/post-session-ooc", body: `{}`, wantMethod: "PostSessionOOC"},
		{name: "mark ooc ready", path: "/api/campaigns/c1/interaction/mark-ooc-ready-to-resume", wantMethod: "MarkOOCReadyToResume"},
		{name: "clear ooc ready", path: "/api/campaigns/c1/interaction/clear-ooc-ready-to-resume", wantMethod: "ClearOOCReadyToResume"},
		{name: "resume from ooc", path: "/api/campaigns/c1/interaction/resume-from-ooc", wantMethod: "ResumeFromOOC"},
		{name: "set gm authority", path: "/api/campaigns/c1/interaction/set-session-gm-authority", body: `{}`, wantMethod: "SetSessionGMAuthority"},
		{name: "retry ai gm turn", path: "/api/campaigns/c1/interaction/retry-ai-gm-turn", body: `{}`, wantMethod: "RetryAIGMTurn"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			interaction := newRecordingInteractionClient(playTestState())
			transcripts := &scriptTranscriptStore{latest: 11}
			server := newAuthedPlayServer(interaction, transcripts)
			handler, err := server.newHandler(testPlayLaunchGrantConfig(t))
			if err != nil {
				t.Fatalf("newHandler() error = %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "http://play.example.com"+tc.path, strings.NewReader(tc.body))
			req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
			}
			if interaction.lastMethod != tc.wantMethod {
				t.Fatalf("method = %q, want %q", interaction.lastMethod, tc.wantMethod)
			}
			if interaction.lastCampaignID != "c1" {
				t.Fatalf("campaign_id = %q, want %q", interaction.lastCampaignID, "c1")
			}

			var payload playprotocol.RoomSnapshot
			if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode interaction response: %v", err)
			}
			if payload.InteractionState.CampaignID != "c1" {
				t.Fatalf("interaction_state.campaign_id = %q, want %q", payload.InteractionState.CampaignID, "c1")
			}
			if payload.InteractionState.Viewer == nil || payload.InteractionState.Viewer.ParticipantID != "p1" {
				t.Fatalf("interaction_state.viewer = %#v", payload.InteractionState.Viewer)
			}
			if payload.Chat.LatestSequenceID != 11 {
				t.Fatalf("latest_sequence_id = %d, want %d", payload.Chat.LatestSequenceID, 11)
			}
		})
	}
}

func TestInteractionMutationRejectsInvalidJSONAndAuthFailures(t *testing.T) {
	t.Parallel()

	t.Run("invalid json body", func(t *testing.T) {
		t.Parallel()

		server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
		handler, err := server.newHandler(testPlayLaunchGrantConfig(t))
		if err != nil {
			t.Fatalf("newHandler() error = %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "http://play.example.com/api/campaigns/c1/interaction/set-active-scene", strings.NewReader(`{"unknown":true}`))
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assertJSONError(t, rr, http.StatusBadRequest, "invalid json body")
	})

	t.Run("missing play session", func(t *testing.T) {
		t.Parallel()

		server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
		handler, err := server.newHandler(testPlayLaunchGrantConfig(t))
		if err != nil {
			t.Fatalf("newHandler() error = %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "http://play.example.com/api/campaigns/c1/interaction/set-active-scene", strings.NewReader(`{}`))
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assertJSONError(t, rr, http.StatusUnauthorized, "authentication required")
	})

	t.Run("upstream invalid argument", func(t *testing.T) {
		t.Parallel()

		interaction := newRecordingInteractionClient(playTestState())
		interaction.mutationErr = gogrpcstatus.Error(gogrpccodes.InvalidArgument, "bad scene")
		server := newAuthedPlayServer(interaction, &scriptTranscriptStore{})
		handler, err := server.newHandler(testPlayLaunchGrantConfig(t))
		if err != nil {
			t.Fatalf("newHandler() error = %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "http://play.example.com/api/campaigns/c1/interaction/set-active-scene", strings.NewReader(`{}`))
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assertJSONError(t, rr, http.StatusBadRequest, "bad scene")
	})
}
