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
		{name: "set active scene", path: "/api/campaigns/c1/interaction/activate-scene", body: `{}`, wantMethod: "ActivateScene"},
		{name: "start scene player phase", path: "/api/campaigns/c1/interaction/open-scene-player-phase", body: `{}`, wantMethod: "OpenScenePlayerPhase"},
		{name: "submit scene player post", path: "/api/campaigns/c1/interaction/submit-scene-player-action", body: `{}`, wantMethod: "SubmitScenePlayerAction"},
		{name: "yield scene player phase", path: "/api/campaigns/c1/interaction/yield-scene-player-phase", body: `{}`, wantMethod: "YieldScenePlayerPhase"},
		{name: "unyield scene player phase", path: "/api/campaigns/c1/interaction/withdraw-scene-player-yield", body: `{}`, wantMethod: "WithdrawScenePlayerYield"},
		{name: "end scene player phase", path: "/api/campaigns/c1/interaction/interrupt-scene-player-phase", body: `{}`, wantMethod: "InterruptScenePlayerPhase"},
		{name: "commit scene gm interaction", path: "/api/campaigns/c1/interaction/record-scene-gm-interaction", body: `{}`, wantMethod: "RecordSceneGMInteraction"},
		{name: "resolve scene player phase review", path: "/api/campaigns/c1/interaction/resolve-scene-player-review", body: `{}`, wantMethod: "ResolveScenePlayerReview"},
		{name: "pause session for ooc", path: "/api/campaigns/c1/interaction/open-session-ooc", body: `{}`, wantMethod: "OpenSessionOOC"},
		{name: "post session ooc", path: "/api/campaigns/c1/interaction/post-session-ooc", body: `{}`, wantMethod: "PostSessionOOC"},
		{name: "mark ooc ready", path: "/api/campaigns/c1/interaction/mark-ooc-ready-to-resume", wantMethod: "MarkOOCReadyToResume"},
		{name: "clear ooc ready", path: "/api/campaigns/c1/interaction/clear-ooc-ready-to-resume", wantMethod: "ClearOOCReadyToResume"},
		{name: "resolve session ooc", path: "/api/campaigns/c1/interaction/resolve-session-ooc", body: `{}`, wantMethod: "ResolveSessionOOC"},
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
		req := httptest.NewRequest(http.MethodPost, "http://play.example.com/api/campaigns/c1/interaction/activate-scene", strings.NewReader(`{"unknown":true}`))
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
		req := httptest.NewRequest(http.MethodPost, "http://play.example.com/api/campaigns/c1/interaction/activate-scene", strings.NewReader(`{}`))
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
		req := httptest.NewRequest(http.MethodPost, "http://play.example.com/api/campaigns/c1/interaction/activate-scene", strings.NewReader(`{}`))
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assertJSONError(t, rr, http.StatusBadRequest, "invalid request")
	})
}

func TestInteractionMutationResponseKeepsParticipantAndCharacterEnrichment(t *testing.T) {
	t.Parallel()

	interaction := newRecordingInteractionClient(playTestState())
	server := newAuthedPlayServer(interaction, &scriptTranscriptStore{})
	participants := &authSensitivePlayParticipantClient{response: enrichedParticipantResponse()}
	characters := &authSensitivePlayCharacterClient{
		listResponse:  enrichedCharacterResponse(),
		sheetResponse: enrichedCharacterSheetResponse(),
	}
	server.deps.Participants = participants
	server.deps.Characters = characters

	handler, err := server.newHandler(testPlayLaunchGrantConfig(t))
	if err != nil {
		t.Fatalf("newHandler() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://play.example.com/api/campaigns/c1/interaction/submit-scene-player-action", strings.NewReader(`{}`))
	req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload playprotocol.RoomSnapshot
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode interaction response: %v", err)
	}
	if len(payload.Participants) != 2 {
		t.Fatalf("participants = %#v, want 2 entries", payload.Participants)
	}
	if got := payload.CharacterInspectionCatalog["char-1"].System; got != "daggerheart" {
		t.Fatalf("character_inspection_catalog[char-1].system = %q, want %q", got, "daggerheart")
	}
	card, ok := payload.CharacterInspectionCatalog["char-1"].Card.(map[string]any)
	if !ok {
		t.Fatalf("character_inspection_catalog[char-1].card type = %T, want object", payload.CharacterInspectionCatalog["char-1"].Card)
	}
	daggerheartCard, ok := card["daggerheart"].(map[string]any)
	if !ok {
		t.Fatalf("character_inspection_catalog[char-1].card.daggerheart = %#v", card["daggerheart"])
	}
	cardSummary, ok := daggerheartCard["summary"].(map[string]any)
	if !ok {
		t.Fatalf("character_inspection_catalog[char-1].card.daggerheart.summary = %#v", daggerheartCard["summary"])
	}
	if got := cardSummary["ancestryName"]; got != "Human" {
		t.Fatalf("character_inspection_catalog[char-1].card.summary.ancestryName = %#v", got)
	}
	if got := cardSummary["communityName"]; got != "Slyborne" {
		t.Fatalf("character_inspection_catalog[char-1].card.summary.communityName = %#v", got)
	}
	sheet, ok := payload.CharacterInspectionCatalog["char-1"].Sheet.(map[string]any)
	if !ok {
		t.Fatalf("character_inspection_catalog[char-1].sheet type = %T, want object", payload.CharacterInspectionCatalog["char-1"].Sheet)
	}
	if got := sheet["ancestryName"]; got != "Human" {
		t.Fatalf("character_inspection_catalog[char-1].sheet.ancestryName = %#v", got)
	}
	if got := sheet["communityName"]; got != "Slyborne" {
		t.Fatalf("character_inspection_catalog[char-1].sheet.communityName = %#v", got)
	}
	if got := sheet["hopeFeature"]; got != "Rogue's Dodge: Spend 3 Hope to gain +2 Evasion until an attack succeeds against you." {
		t.Fatalf("character_inspection_catalog[char-1].sheet.hopeFeature = %#v", got)
	}
	primaryWeapon, ok := sheet["primaryWeapon"].(map[string]any)
	if !ok || primaryWeapon["name"] != "Sword" {
		t.Fatalf("character_inspection_catalog[char-1].sheet.primaryWeapon = %#v", sheet["primaryWeapon"])
	}
	activeArmor, ok := sheet["activeArmor"].(map[string]any)
	if !ok || activeArmor["name"] != "Leather" {
		t.Fatalf("character_inspection_catalog[char-1].sheet.activeArmor = %#v", sheet["activeArmor"])
	}
	if participants.lastUserID != "user-1" || characters.lastUserID != "user-1" {
		t.Fatalf("auth metadata = participant:%q character:%q, want user-1", participants.lastUserID, characters.lastUserID)
	}
}
