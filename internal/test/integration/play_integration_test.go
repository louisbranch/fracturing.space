//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/play/playtest"
	playprotocol "github.com/louisbranch/fracturing.space/internal/services/play/protocol"
	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	"golang.org/x/net/websocket"
)

func TestPlayLaunchAndRealtimeIntegration(t *testing.T) {
	fixture := newSuiteFixture(t)
	playRuntime := playtest.StartRuntime(t, fixture.authAddr, fixture.grpcAddr)
	eventClient, closeEventConn := newEventClient(t, fixture.grpcAddr)
	defer closeEventConn()

	userID := createAuthUser(t, fixture.authAddr, uniqueTestUsername(t, "play-user"))
	suite := fixture.newGameSuite(t, userID)
	campaignID, sessionID := createPlayCampaignAndSession(t, suite)

	grant, _, err := playlaunchgrant.Issue(playRuntime.LaunchGrantConfig, playlaunchgrant.IssueInput{
		GrantID:    "play-launch-" + strings.ReplaceAll(uniqueTestUsername(t, "grant"), "-", ""),
		CampaignID: campaignID,
		UserID:     userID,
	})
	if err != nil {
		t.Fatalf("issue play launch grant: %v", err)
	}

	httpClient := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	launchResp, err := httpClient.Get(playRuntime.BaseURL + "/campaigns/" + campaignID + "?launch=" + url.QueryEscape(grant))
	if err != nil {
		t.Fatalf("launch play shell: %v", err)
	}
	defer launchResp.Body.Close()

	if launchResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("launch status = %d, want %d", launchResp.StatusCode, http.StatusSeeOther)
	}
	if got := launchResp.Header.Get("Location"); got != "/campaigns/"+campaignID {
		t.Fatalf("launch redirect = %q, want %q", got, "/campaigns/"+campaignID)
	}
	playSessionID := playtest.RequireCookieValue(t, launchResp.Cookies(), "play_session")

	shellReq, err := http.NewRequest(http.MethodGet, playRuntime.BaseURL+"/campaigns/"+campaignID, nil)
	if err != nil {
		t.Fatalf("build shell request: %v", err)
	}
	shellReq.AddCookie(&http.Cookie{Name: "play_session", Value: playSessionID})
	shellResp, err := httpClient.Do(shellReq)
	if err != nil {
		t.Fatalf("load campaign shell: %v", err)
	}
	shellBody, err := io.ReadAll(shellResp.Body)
	_ = shellResp.Body.Close()
	if err != nil {
		t.Fatalf("read campaign shell: %v", err)
	}
	if shellResp.StatusCode != http.StatusOK {
		t.Fatalf("shell status = %d, want %d", shellResp.StatusCode, http.StatusOK)
	}
	if !strings.Contains(string(shellBody), "/api/campaigns/"+campaignID+"/bootstrap") {
		t.Fatalf("shell body missing bootstrap path: %q", string(shellBody))
	}

	bootstrapReq, err := http.NewRequest(http.MethodGet, playRuntime.BaseURL+"/api/campaigns/"+campaignID+"/bootstrap", nil)
	if err != nil {
		t.Fatalf("build bootstrap request: %v", err)
	}
	bootstrapReq.AddCookie(&http.Cookie{Name: "play_session", Value: playSessionID})
	bootstrapResp, err := httpClient.Do(bootstrapReq)
	if err != nil {
		t.Fatalf("load bootstrap: %v", err)
	}
	defer bootstrapResp.Body.Close()

	var bootstrap playprotocol.Bootstrap
	if err := decodeHTTPJSON(bootstrapResp, &bootstrap); err != nil {
		t.Fatalf("decode bootstrap: %v", err)
	}
	if bootstrap.CampaignID != campaignID {
		t.Fatalf("bootstrap campaign_id = %q, want %q", bootstrap.CampaignID, campaignID)
	}
	if bootstrap.InteractionState.ActiveSession == nil || bootstrap.InteractionState.ActiveSession.SessionID != sessionID {
		t.Fatalf("bootstrap active session = %#v, want session %q", bootstrap.InteractionState.ActiveSession, sessionID)
	}
	if bootstrap.Chat.HistoryURL != "/api/campaigns/"+campaignID+"/chat/history" {
		t.Fatalf("bootstrap history_url = %q", bootstrap.Chat.HistoryURL)
	}
	if bootstrap.Realtime.URL != "/realtime" {
		t.Fatalf("bootstrap realtime.url = %q", bootstrap.Realtime.URL)
	}

	wsConfig, err := websocket.NewConfig(playtest.WebsocketURLFromHTTP(playRuntime.BaseURL, "/realtime"), playRuntime.BaseURL)
	if err != nil {
		t.Fatalf("build websocket config: %v", err)
	}
	wsConfig.Header = sessionCookieHeader(playSessionID)
	wsConn, err := websocket.DialConfig(wsConfig)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer wsConn.Close()

	if err := websocket.JSON.Send(wsConn, playSendFrame{
		Type:      "play.connect",
		RequestID: "req-connect",
		Payload: playprotocol.ConnectRequest{
			CampaignID: campaignID,
		},
	}); err != nil {
		t.Fatalf("send connect frame: %v", err)
	}

	readyFrame := waitForPlayFrame(t, wsConn, "play.ready")
	readySnapshot := decodePlayPayload[playprotocol.RoomSnapshot](t, readyFrame.Payload)
	if readySnapshot.InteractionState.ActiveSession == nil || readySnapshot.InteractionState.ActiveSession.SessionID != sessionID {
		t.Fatalf("ready active session = %#v, want %q", readySnapshot.InteractionState.ActiveSession, sessionID)
	}

	if err := websocket.JSON.Send(wsConn, playSendFrame{
		Type:      "play.chat.send",
		RequestID: "req-chat",
		Payload: playprotocol.ChatSendRequest{
			ClientMessageID: "cm-1",
			Body:            "Hello from play integration",
		},
	}); err != nil {
		t.Fatalf("send chat frame: %v", err)
	}

	chatFrame := waitForPlayFrame(t, wsConn, "play.chat.message")
	chatEnvelope := decodePlayPayload[playprotocol.ChatMessageEnvelope](t, chatFrame.Payload)
	if chatEnvelope.Message.Body != "Hello from play integration" {
		t.Fatalf("chat body = %q, want %q", chatEnvelope.Message.Body, "Hello from play integration")
	}
	if chatEnvelope.Message.ClientMessageID != "cm-1" {
		t.Fatalf("chat client_message_id = %q, want %q", chatEnvelope.Message.ClientMessageID, "cm-1")
	}
	if chatEnvelope.Message.SessionID != sessionID {
		t.Fatalf("chat session_id = %q, want %q", chatEnvelope.Message.SessionID, sessionID)
	}

	historyReq, err := http.NewRequest(http.MethodGet, playRuntime.BaseURL+"/api/campaigns/"+campaignID+"/chat/history", nil)
	if err != nil {
		t.Fatalf("build history request: %v", err)
	}
	historyReq.AddCookie(&http.Cookie{Name: "play_session", Value: playSessionID})
	historyResp, err := httpClient.Do(historyReq)
	if err != nil {
		t.Fatalf("load chat history: %v", err)
	}
	defer historyResp.Body.Close()

	var history playprotocol.HistoryResponse
	if err := decodeHTTPJSON(historyResp, &history); err != nil {
		t.Fatalf("decode history: %v", err)
	}
	if history.SessionID != sessionID {
		t.Fatalf("history session_id = %q, want %q", history.SessionID, sessionID)
	}
	if len(history.Messages) != 1 || history.Messages[0].Body != "Hello from play integration" {
		t.Fatalf("history messages = %#v", history.Messages)
	}

	time.Sleep(150 * time.Millisecond)

	endCtx, cancel := context.WithTimeout(suite.ctx(context.Background()), integrationTimeout())
	defer cancel()
	if _, err := suite.session.EndSession(endCtx, &gamev1.EndSessionRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	}); err != nil {
		t.Fatalf("end session: %v", err)
	}

	eventCtx, eventCancel := context.WithTimeout(suite.ctx(context.Background()), integrationTimeout())
	defer eventCancel()
	targetSeq := requireLatestSeq(t, eventCtx, eventClient, campaignID)

	updateSnapshot := waitForPlayInteractionUpdate(t, wsConn, func(snapshot playprotocol.RoomSnapshot) bool {
		return snapshot.LatestGameSeq >= targetSeq && snapshot.InteractionState.ActiveSession == nil
	})
	if updateSnapshot.InteractionState.ActiveSession != nil {
		t.Fatalf("updated active session = %#v, want nil", updateSnapshot.InteractionState.ActiveSession)
	}
}

func createPlayCampaignAndSession(t *testing.T, suite *integrationSuite) (campaignID string, sessionID string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(suite.ctx(context.Background()), integrationTimeout())
	defer cancel()

	createResp, err := suite.campaign.CreateCampaign(ctx, &gamev1.CreateCampaignRequest{
		Name:   "play-integration-" + t.Name(),
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: gamev1.GmMode_HUMAN,
	})
	if err != nil {
		t.Fatalf("create play campaign: %v", err)
	}

	campaignID = createResp.GetCampaign().GetId()
	ownerParticipantID := createResp.GetOwnerParticipant().GetId()
	if campaignID == "" || ownerParticipantID == "" {
		t.Fatalf("campaign/owner ids = %q/%q, want non-empty", campaignID, ownerParticipantID)
	}

	characterResp, err := suite.character.CreateCharacter(ctx, &gamev1.CreateCharacterRequest{
		CampaignId: campaignID,
		Name:       "Play Runner",
		Kind:       gamev1.CharacterKind_PC,
	})
	if err != nil {
		t.Fatalf("create play character: %v", err)
	}
	characterID := characterResp.GetCharacter().GetId()
	if characterID == "" {
		t.Fatal("play character id is empty")
	}

	setCharacterOwner(t, ctx, suite.character, campaignID, characterID, ownerParticipantID)
	ensureDaggerheartCreationReadiness(t, ctx, suite.character, campaignID, characterID)
	ensureSessionStartReadiness(t, ctx, suite.participant, suite.character, campaignID, ownerParticipantID, characterID)

	sessionResp := startSessionWithDefaultControllers(t, ctx, suite.session, suite.character, campaignID, "Play Integration Session")

	sessionID = sessionResp.GetSession().GetId()
	if sessionID == "" {
		t.Fatal("play session id is empty")
	}
	return campaignID, sessionID
}

func decodeHTTPJSON(resp *http.Response, target any) error {
	if resp == nil {
		return io.ErrUnexpectedEOF
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(target)
}
