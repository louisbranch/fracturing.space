package app

import (
	"context"
	"errors"
	"testing"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	playdaggerheart "github.com/louisbranch/fracturing.space/internal/services/play/protocol/daggerheart"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPlayApplicationSystemMetadata(t *testing.T) {
	t.Parallel()

	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
	server.deps.Campaign = fakePlayCampaignClient{response: &gamev1.GetCampaignResponse{
		Campaign: &gamev1.Campaign{Id: "c1", System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART},
	}}
	server.deps.System = fakePlaySystemClient{response: &gamev1.GetGameSystemResponse{
		System: &gamev1.GameSystemInfo{Id: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, Name: "Daggerheart", Version: "v1"},
	}}

	system, err := server.application().systemMetadata(context.Background(), playRequest{
		campaignRequest: campaignRequest{CampaignID: "c1"},
		UserID:          "user-1",
	})
	if err != nil {
		t.Fatalf("systemMetadata() error = %v", err)
	}
	if system.ID != "daggerheart" || system.Name != "Daggerheart" || system.Version != "v1" {
		t.Fatalf("system = %#v", system)
	}
}

func TestBuildCharacterInspectionCatalogEnrichesLocalizedDomainCards(t *testing.T) {
	t.Parallel()

	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
	server.deps.Characters = fakePlayCharacterClient{sheetResponse: enrichedCharacterSheetResponse()}
	content := &fakeDaggerheartContentClient{
		responses: map[string]*daggerheartv1.DaggerheartDomainCard{
			"domain_card.valor-i-am-your-shield": {
				Id:          "domain_card.valor-i-am-your-shield",
				Name:        "I Am Your Shield",
				DomainId:    "domain.valor",
				FeatureText: "Protect an ally within very close range.",
			},
			"domain_card.blade-get-back-up": {
				Id:          "domain_card.blade-get-back-up",
				Name:        "Get Back Up",
				DomainId:    "domain.blade",
				FeatureText: "Stand again and keep fighting.",
			},
		},
	}
	server.deps.DaggerheartContent = content

	catalog := server.application().buildCharacterInspectionCatalog(
		context.Background(),
		"c1",
		commonv1.Locale_LOCALE_EN_US,
		enrichedCharacterResponse().GetCharacters(),
	)
	if len(catalog) != 1 {
		t.Fatalf("catalog size = %d, want 1", len(catalog))
	}

	inspection := catalog["char-1"]
	sheet, ok := inspection.Sheet.(playdaggerheart.CharacterSheetData)
	if !ok {
		t.Fatalf("sheet type = %T, want DaggerheartCharacterSheetData", inspection.Sheet)
	}
	if len(sheet.DomainCards) != 2 {
		t.Fatalf("domain cards = %#v, want 2", sheet.DomainCards)
	}
	if sheet.DomainCards[0].ID != "domain_card.valor-i-am-your-shield" {
		t.Fatalf("domainCards[0].ID = %q", sheet.DomainCards[0].ID)
	}
	if sheet.DomainCards[0].FeatureText != "Protect an ally within very close range." {
		t.Fatalf("domainCards[0].FeatureText = %q", sheet.DomainCards[0].FeatureText)
	}
	if sheet.DomainCards[1].FeatureText != "Stand again and keep fighting." {
		t.Fatalf("domainCards[1].FeatureText = %q", sheet.DomainCards[1].FeatureText)
	}

	if len(content.requests) != 2 {
		t.Fatalf("content requests = %d, want 2", len(content.requests))
	}
	for _, req := range content.requests {
		if req.GetLocale() != commonv1.Locale_LOCALE_EN_US {
			t.Fatalf("request locale = %v, want %v", req.GetLocale(), commonv1.Locale_LOCALE_EN_US)
		}
	}
}

func TestBuildCharacterInspectionCatalogFallsBackWhenDomainCardLookupFails(t *testing.T) {
	t.Parallel()

	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
	server.deps.Characters = fakePlayCharacterClient{sheetResponse: enrichedCharacterSheetResponse()}
	server.deps.DaggerheartContent = &fakeDaggerheartContentClient{
		responses: map[string]*daggerheartv1.DaggerheartDomainCard{
			"domain_card.blade-get-back-up": {
				Id:          "domain_card.blade-get-back-up",
				Name:        "Get Back Up",
				DomainId:    "domain.blade",
				FeatureText: "Stand again and keep fighting.",
			},
		},
		errByID: map[string]error{
			"domain_card.valor-i-am-your-shield": context.DeadlineExceeded,
		},
	}

	catalog := server.application().buildCharacterInspectionCatalog(
		context.Background(),
		"c1",
		commonv1.Locale_LOCALE_EN_US,
		enrichedCharacterResponse().GetCharacters(),
	)
	sheet := catalog["char-1"].Sheet.(playdaggerheart.CharacterSheetData)
	if len(sheet.DomainCards) != 2 {
		t.Fatalf("domain cards = %#v, want 2", sheet.DomainCards)
	}
	if sheet.DomainCards[0].Name != "I Am Your Shield" {
		t.Fatalf("domainCards[0].Name = %q, want fallback label", sheet.DomainCards[0].Name)
	}
	if sheet.DomainCards[0].FeatureText != "" {
		t.Fatalf("domainCards[0].FeatureText = %q, want empty on fallback", sheet.DomainCards[0].FeatureText)
	}
	if sheet.DomainCards[1].FeatureText != "Stand again and keep fighting." {
		t.Fatalf("domainCards[1].FeatureText = %q", sheet.DomainCards[1].FeatureText)
	}
}

func TestBuildCharacterInspectionCatalogSkipsCharactersWhenSheetFetchFails(t *testing.T) {
	t.Parallel()

	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
	server.deps.Characters = fakePlayCharacterClient{sheetErr: errors.New("unavailable")}

	catalog := server.application().buildCharacterInspectionCatalog(
		context.Background(),
		"c1",
		commonv1.Locale_LOCALE_EN_US,
		enrichedCharacterResponse().GetCharacters(),
	)
	if len(catalog) != 0 {
		t.Fatalf("catalog = %#v, want empty when sheet fetch fails", catalog)
	}
}

func TestPlayApplicationAIDebugAccessors(t *testing.T) {
	t.Parallel()

	t.Run("ai debug turns use active session", func(t *testing.T) {
		t.Parallel()

		aiDebug := &fakePlayAIDebugClient{
			listResp: &aiv1.ListCampaignDebugTurnsResponse{
				Turns: []*aiv1.CampaignDebugTurnSummary{{
					Id:         "turn-1",
					Model:      "gpt-5.4",
					Status:     aiv1.CampaignDebugTurnStatus_CAMPAIGN_DEBUG_TURN_STATUS_RUNNING,
					EntryCount: 1,
				}},
				NextPageToken: "next-1",
			},
		}
		server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
		server.deps.AIDebug = aiDebug

		page, err := server.application().aiDebugTurns(context.Background(), playRequest{
			campaignRequest: campaignRequest{CampaignID: "c1"},
			UserID:          "user-1",
		}, aiDebugPage{PageSize: 10, PageToken: "next-0"})
		if err != nil {
			t.Fatalf("aiDebugTurns() error = %v", err)
		}
		if len(page.Turns) != 1 || page.Turns[0].ID != "turn-1" || page.NextPageToken != "next-1" {
			t.Fatalf("page = %#v", page)
		}
		if aiDebug.listReq == nil || aiDebug.listReq.GetSessionId() != "s1" || aiDebug.listReq.GetPageSize() != 10 || aiDebug.listReq.GetPageToken() != "next-0" {
			t.Fatalf("list request = %#v", aiDebug.listReq)
		}
	})

	t.Run("ai debug turns without active session skip upstream", func(t *testing.T) {
		t.Parallel()

		state := playTestState()
		state.ActiveSession = nil
		aiDebug := &fakePlayAIDebugClient{}
		server := newAuthedPlayServer(newRecordingInteractionClient(state), &scriptTranscriptStore{})
		server.deps.AIDebug = aiDebug

		page, err := server.application().aiDebugTurns(context.Background(), playRequest{
			campaignRequest: campaignRequest{CampaignID: "c1"},
			UserID:          "user-1",
		}, aiDebugPage{PageSize: 10})
		if err != nil {
			t.Fatalf("aiDebugTurns() error = %v", err)
		}
		if len(page.Turns) != 0 || aiDebug.listReq != nil {
			t.Fatalf("page/listReq = (%#v, %#v), want empty result without upstream call", page, aiDebug.listReq)
		}
	})

	t.Run("ai debug turn trims requested id", func(t *testing.T) {
		t.Parallel()

		aiDebug := &fakePlayAIDebugClient{
			getResp: &aiv1.GetCampaignDebugTurnResponse{
				Turn: &aiv1.CampaignDebugTurn{
					Id:     "turn-1",
					Model:  "gpt-5.4",
					Status: aiv1.CampaignDebugTurnStatus_CAMPAIGN_DEBUG_TURN_STATUS_SUCCEEDED,
				},
			},
		}
		server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
		server.deps.AIDebug = aiDebug

		turn, err := server.application().aiDebugTurn(context.Background(), playRequest{
			campaignRequest: campaignRequest{CampaignID: "c1"},
			UserID:          "user-1",
		}, " turn-1 ")
		if err != nil {
			t.Fatalf("aiDebugTurn() error = %v", err)
		}
		if turn.ID != "turn-1" {
			t.Fatalf("turn = %#v", turn)
		}
		if aiDebug.getReq == nil || aiDebug.getReq.GetTurnId() != "turn-1" {
			t.Fatalf("get request = %#v", aiDebug.getReq)
		}
	})

	t.Run("ai debug turn propagates upstream error", func(t *testing.T) {
		t.Parallel()

		aiDebug := &fakePlayAIDebugClient{getErr: status.Error(codes.NotFound, "missing")}
		server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
		server.deps.AIDebug = aiDebug

		if _, err := server.application().aiDebugTurn(context.Background(), playRequest{
			campaignRequest: campaignRequest{CampaignID: "c1"},
			UserID:          "user-1",
		}, "turn-1"); status.Code(err) != codes.NotFound {
			t.Fatalf("aiDebugTurn() code = %v, want %v", status.Code(err), codes.NotFound)
		}
	})
}
