//go:build integration

package integration

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/state/v1"
	"github.com/louisbranch/fracturing.space/internal/web"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestWebHTMXIntegration validates web HTMX endpoints against real gRPC data.
func TestWebHTMXIntegration(t *testing.T) {
	grpcAddr, stopGRPC := startGRPCServer(t)
	defer stopGRPC()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	webAddr, stopWeb := startWebServer(ctx, t, grpcAddr)
	defer stopWeb()

	conn, err := grpc.NewClient(grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial gRPC: %v", err)
	}
	defer conn.Close()

	campaignClient := statev1.NewCampaignServiceClient(conn)
	sessionClient := statev1.NewSessionServiceClient(conn)
	characterClient := statev1.NewCharacterServiceClient(conn)
	participantClient := statev1.NewParticipantServiceClient(conn)
	httpClient := &http.Client{Timeout: 5 * time.Second}
	baseURL := "http://" + webAddr

	t.Run("campaigns table empty", func(t *testing.T) {
		status, body := htmxGet(t, httpClient, baseURL+"/campaigns/table")

		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d", status)
		}

		assertHTMLContains(t, body, "No campaigns yet.")
		assertHTMLNotContains(t, body, "<!doctype html>", "<!DOCTYPE html>")
	})

	t.Run("campaigns table with data", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:        "Web Test Campaign",
			System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode:      statev1.GmMode_HUMAN,
			ThemePrompt: "Dark fantasy adventure",
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		status, body := htmxGet(t, httpClient, baseURL+"/campaigns/table")

		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d", status)
		}

		assertHTMLContains(t, body,
			"Web Test Campaign",
			"/campaigns/"+campaignID,
			"Human",
		)
		assertHTMLNotContains(t, body, "<!doctype html>", "<!DOCTYPE html>")
	})

	t.Run("campaign detail full page", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Detail Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/campaigns/"+campaignID, nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		body := string(bodyBytes)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		assertHTMLContains(t, body,
			"<!doctype html>",
			"Fracturing.Space",
			"Detail Test Campaign",
			campaignID,
		)
	})

	t.Run("campaign detail htmx fragment", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Fragment Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_HYBRID,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		status, body := htmxGet(t, httpClient, baseURL+"/campaigns/"+campaignID)

		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d", status)
		}

		assertHTMLContains(t, body,
			"Fragment Test Campaign",
			"Hybrid",
		)
		assertHTMLNotContains(t, body, "<!doctype html>", "<!DOCTYPE html>")
	})

	t.Run("campaign sessions htmx fragment", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Session Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		status, body := htmxGet(t, httpClient, baseURL+"/campaigns/"+campaignID+"/sessions")

		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d", status)
		}

		// The HTMX fragment returns the sessions page with a loading placeholder
		assertHTMLContains(t, body,
			"Session Test Campaign",
			"<h3>Sessions</h3>",
		)
		assertHTMLNotContains(t, body, "<!doctype html>", "<!DOCTYPE html>")
	})

	t.Run("campaign not found", func(t *testing.T) {
		status, body := htmxGet(t, httpClient, baseURL+"/campaigns/nonexistent-campaign-id")

		if status != http.StatusOK {
			t.Fatalf("expected status 200 (graceful degradation), got %d", status)
		}

		assertHTMLContains(t, body, "Campaign unavailable.")
	})

	// Dashboard tests
	t.Run("dashboard full page", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/", nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		body := string(bodyBytes)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		assertHTMLContains(t, body,
			"<!doctype html>",
			"Fracturing.Space",
			"<h2>Dashboard</h2>",
		)
	})

	t.Run("dashboard htmx fragment", func(t *testing.T) {
		status, body := htmxGet(t, httpClient, baseURL+"/")

		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d", status)
		}

		assertHTMLContains(t, body, "<h2>Dashboard</h2>")
		assertHTMLNotContains(t, body, "<!doctype html>", "<!DOCTYPE html>")
	})

	t.Run("dashboard content with data", func(t *testing.T) {
		// Create campaign and session to ensure dashboard has data
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Dashboard Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		_, err = sessionClient.StartSession(ctx, &statev1.StartSessionRequest{
			CampaignId: campaignID,
			Name:       "Dashboard Test Session",
		})
		if err != nil {
			t.Fatalf("start session: %v", err)
		}

		status, body := htmxGet(t, httpClient, baseURL+"/dashboard/content")

		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d", status)
		}

		assertHTMLContains(t, body,
			"Campaigns",
			"Active Sessions",
		)
		assertHTMLNotContains(t, body, "<!doctype html>", "<!DOCTYPE html>")
	})

	// Sessions tests
	t.Run("sessions list full page", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Sessions List Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/campaigns/"+campaignID+"/sessions", nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		body := string(bodyBytes)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		assertHTMLContains(t, body,
			"<!doctype html>",
			"Fracturing.Space",
			"<h3>Sessions</h3>",
		)
	})

	t.Run("sessions table empty", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Empty Sessions Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		status, body := htmxGet(t, httpClient, baseURL+"/campaigns/"+campaignID+"/sessions/table")

		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d", status)
		}

		assertHTMLContains(t, body, "No sessions yet.")
		assertHTMLNotContains(t, body, "<!doctype html>", "<!DOCTYPE html>")
	})

	t.Run("sessions table with data", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Sessions Table Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		_, err = sessionClient.StartSession(ctx, &statev1.StartSessionRequest{
			CampaignId: campaignID,
			Name:       "Test Session Alpha",
		})
		if err != nil {
			t.Fatalf("start session: %v", err)
		}

		status, body := htmxGet(t, httpClient, baseURL+"/campaigns/"+campaignID+"/sessions/table")

		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d", status)
		}

		assertHTMLContains(t, body,
			"Test Session Alpha",
			"Active",
		)
		assertHTMLNotContains(t, body, "<!doctype html>", "<!DOCTYPE html>")
	})

	t.Run("session detail full page", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Session Detail Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		sessionResp, err := sessionClient.StartSession(ctx, &statev1.StartSessionRequest{
			CampaignId: campaignID,
			Name:       "Detail Session Test",
		})
		if err != nil {
			t.Fatalf("start session: %v", err)
		}
		sessionID := sessionResp.Session.Id

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/campaigns/"+campaignID+"/sessions/"+sessionID, nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		body := string(bodyBytes)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		assertHTMLContains(t, body,
			"<!doctype html>",
			"Fracturing.Space",
			"<h3>Session Info</h3>",
			"<h3>Event Timeline</h3>",
		)
	})

	t.Run("session detail htmx fragment", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Session Detail HTMX Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		sessionResp, err := sessionClient.StartSession(ctx, &statev1.StartSessionRequest{
			CampaignId: campaignID,
			Name:       "HTMX Session Detail",
		})
		if err != nil {
			t.Fatalf("start session: %v", err)
		}
		sessionID := sessionResp.Session.Id

		status, body := htmxGet(t, httpClient, baseURL+"/campaigns/"+campaignID+"/sessions/"+sessionID)

		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d", status)
		}

		assertHTMLContains(t, body,
			"HTMX Session Detail",
			"<h3>Session Info</h3>",
			"<h3>Event Timeline</h3>",
		)
		assertHTMLNotContains(t, body, "<!doctype html>", "<!DOCTYPE html>")
	})

	t.Run("session events", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Session Events Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		sessionResp, err := sessionClient.StartSession(ctx, &statev1.StartSessionRequest{
			CampaignId: campaignID,
			Name:       "Events Test Session",
		})
		if err != nil {
			t.Fatalf("start session: %v", err)
		}
		sessionID := sessionResp.Session.Id

		status, body := htmxGet(t, httpClient, baseURL+"/campaigns/"+campaignID+"/sessions/"+sessionID+"/events")

		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d", status)
		}

		// Session events should contain the event table (may be empty or have session.started event)
		assertHTMLNotContains(t, body, "<!doctype html>", "<!DOCTYPE html>")
	})

	// Characters tests
	t.Run("characters list full page", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Characters List Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/campaigns/"+campaignID+"/characters", nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		body := string(bodyBytes)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		assertHTMLContains(t, body,
			"<!doctype html>",
			"Fracturing.Space",
			"<h3>Characters</h3>",
		)
	})

	t.Run("characters list htmx fragment", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Characters HTMX Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		status, body := htmxGet(t, httpClient, baseURL+"/campaigns/"+campaignID+"/characters")

		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d", status)
		}

		assertHTMLContains(t, body,
			"Characters HTMX Test Campaign",
			"<h3>Characters</h3>",
		)
		assertHTMLNotContains(t, body, "<!doctype html>", "<!DOCTYPE html>")
	})

	t.Run("characters table empty", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Empty Characters Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		status, body := htmxGet(t, httpClient, baseURL+"/campaigns/"+campaignID+"/characters/table")

		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d", status)
		}

		assertHTMLContains(t, body, "No characters yet.")
		assertHTMLNotContains(t, body, "<!doctype html>", "<!DOCTYPE html>")
	})

	t.Run("characters table with data", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Characters Data Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		_, err = characterClient.CreateCharacter(ctx, &statev1.CreateCharacterRequest{
			CampaignId: campaignID,
			Name:       "Test Hero Alpha",
			Kind:       statev1.CharacterKind_PC,
		})
		if err != nil {
			t.Fatalf("create character: %v", err)
		}

		status, body := htmxGet(t, httpClient, baseURL+"/campaigns/"+campaignID+"/characters/table")

		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d", status)
		}

		assertHTMLContains(t, body,
			"Test Hero Alpha",
			"PC",
		)
		assertHTMLNotContains(t, body, "<!doctype html>", "<!DOCTYPE html>")
	})

	t.Run("character sheet full page", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Character Sheet Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		charResp, err := characterClient.CreateCharacter(ctx, &statev1.CreateCharacterRequest{
			CampaignId: campaignID,
			Name:       "Sheet Test Character",
			Kind:       statev1.CharacterKind_PC,
		})
		if err != nil {
			t.Fatalf("create character: %v", err)
		}
		characterID := charResp.Character.Id

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/campaigns/"+campaignID+"/characters/"+characterID, nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		body := string(bodyBytes)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		assertHTMLContains(t, body,
			"<!doctype html>",
			"Fracturing.Space",
			"<h3>Character Info</h3>",
		)
	})

	t.Run("character sheet htmx fragment", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Character Sheet HTMX Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		charResp, err := characterClient.CreateCharacter(ctx, &statev1.CreateCharacterRequest{
			CampaignId: campaignID,
			Name:       "HTMX Sheet Character",
			Kind:       statev1.CharacterKind_NPC,
		})
		if err != nil {
			t.Fatalf("create character: %v", err)
		}
		characterID := charResp.Character.Id

		status, body := htmxGet(t, httpClient, baseURL+"/campaigns/"+campaignID+"/characters/"+characterID)

		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d", status)
		}

		assertHTMLContains(t, body,
			"HTMX Sheet Character",
			"<h3>Character Info</h3>",
		)
		assertHTMLNotContains(t, body, "<!doctype html>", "<!DOCTYPE html>")
	})

	// Participants tests
	t.Run("participants list full page", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Participants List Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/campaigns/"+campaignID+"/participants", nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		body := string(bodyBytes)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		assertHTMLContains(t, body,
			"<!doctype html>",
			"Fracturing.Space",
			"<h3>Participants</h3>",
		)
	})

	t.Run("participants list htmx fragment", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Participants HTMX Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		status, body := htmxGet(t, httpClient, baseURL+"/campaigns/"+campaignID+"/participants")

		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d", status)
		}

		assertHTMLContains(t, body,
			"Participants HTMX Test Campaign",
			"<h3>Participants</h3>",
		)
		assertHTMLNotContains(t, body, "<!doctype html>", "<!DOCTYPE html>")
	})

	t.Run("participants table empty", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Empty Participants Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		status, body := htmxGet(t, httpClient, baseURL+"/campaigns/"+campaignID+"/participants/table")

		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d", status)
		}

		assertHTMLContains(t, body, "No participants yet.")
		assertHTMLNotContains(t, body, "<!doctype html>", "<!DOCTYPE html>")
	})

	t.Run("participants table with data", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Participants Data Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		_, err = participantClient.CreateParticipant(ctx, &statev1.CreateParticipantRequest{
			CampaignId:  campaignID,
			DisplayName: "Test Player One",
			Role:        statev1.ParticipantRole_PLAYER,
			Controller:  statev1.Controller_CONTROLLER_HUMAN,
		})
		if err != nil {
			t.Fatalf("create participant: %v", err)
		}

		status, body := htmxGet(t, httpClient, baseURL+"/campaigns/"+campaignID+"/participants/table")

		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d", status)
		}

		assertHTMLContains(t, body,
			"Test Player One",
			"Player",
		)
		assertHTMLNotContains(t, body, "<!doctype html>", "<!DOCTYPE html>")
	})

	// Event Log tests
	t.Run("event log full page", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Event Log Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/campaigns/"+campaignID+"/events", nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		body := string(bodyBytes)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		assertHTMLContains(t, body,
			"<!doctype html>",
			"Fracturing.Space",
			"<h3>Event Log</h3>",
		)
	})

	t.Run("event log htmx fragment", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Event Log HTMX Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		status, body := htmxGet(t, httpClient, baseURL+"/campaigns/"+campaignID+"/events")

		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d", status)
		}

		assertHTMLContains(t, body,
			"<h3>Event Log</h3>",
			"Event Type",
		)
		assertHTMLNotContains(t, body, "<!doctype html>", "<!DOCTYPE html>")
	})

	t.Run("event log table with data", func(t *testing.T) {
		createResp, err := campaignClient.CreateCampaign(ctx, &statev1.CreateCampaignRequest{
			Name:   "Event Table Data Test Campaign",
			System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode: statev1.GmMode_AI,
		})
		if err != nil {
			t.Fatalf("create campaign: %v", err)
		}
		campaignID := createResp.Campaign.Id

		// Start a session to generate events
		_, err = sessionClient.StartSession(ctx, &statev1.StartSessionRequest{
			CampaignId: campaignID,
			Name:       "Event Generation Session",
		})
		if err != nil {
			t.Fatalf("start session: %v", err)
		}

		status, body := htmxGet(t, httpClient, baseURL+"/campaigns/"+campaignID+"/events/table")

		if status != http.StatusOK {
			t.Fatalf("expected status 200, got %d", status)
		}

		// Should have campaign.created and session.started events
		assertHTMLNotContains(t, body, "<!doctype html>", "<!DOCTYPE html>")
	})
}

// startWebServer creates and starts a web server connected to gRPC.
func startWebServer(ctx context.Context, t *testing.T, grpcAddr string) (string, func()) {
	t.Helper()

	httpAddr := pickUnusedAddress(t)
	config := web.Config{
		HTTPAddr:        httpAddr,
		GRPCAddr:        grpcAddr,
		GRPCDialTimeout: 5 * time.Second,
	}

	server, err := web.NewServer(ctx, config)
	if err != nil {
		t.Fatalf("create web server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.ListenAndServe(ctx)
	}()

	waitForWebHealth(t, "http://"+httpAddr)

	stop := func() {
		server.Close()
	}

	return httpAddr, stop
}

// waitForWebHealth polls the web server until it responds.
func waitForWebHealth(t *testing.T, baseURL string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := &http.Client{Timeout: time.Second}
	backoff := 100 * time.Millisecond

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/", nil)
		if err != nil {
			t.Fatalf("create health request: %v", err)
		}

		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}

		select {
		case <-ctx.Done():
			t.Fatalf("wait for web health: %v", ctx.Err())
		case <-time.After(backoff):
		}

		if backoff < time.Second {
			backoff *= 2
			if backoff > time.Second {
				backoff = time.Second
			}
		}
	}
}

// htmxGet performs an HTTP GET with the HX-Request header set.
func htmxGet(t *testing.T, client *http.Client, url string) (int, string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("HX-Request", "true")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return resp.StatusCode, string(bodyBytes)
}

// assertHTMLContains checks that the HTML body contains all expected fragments.
func assertHTMLContains(t *testing.T, body string, fragments ...string) {
	t.Helper()

	for _, fragment := range fragments {
		if !strings.Contains(body, fragment) {
			t.Errorf("expected HTML to contain %q\nbody:\n%s", fragment, body)
		}
	}
}

// assertHTMLNotContains checks that the HTML body does NOT contain any of the fragments.
func assertHTMLNotContains(t *testing.T, body string, fragments ...string) {
	t.Helper()

	for _, fragment := range fragments {
		if strings.Contains(body, fragment) {
			t.Errorf("expected HTML to NOT contain %q", fragment)
		}
	}
}
