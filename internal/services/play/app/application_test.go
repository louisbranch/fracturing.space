package app

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	playprotocol "github.com/louisbranch/fracturing.space/internal/services/play/protocol"
)

func TestPlayApplicationSystemMetadata(t *testing.T) {
	t.Parallel()

	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
	server.campaign = fakePlayCampaignClient{response: &gamev1.GetCampaignResponse{
		Campaign: &gamev1.Campaign{Id: "c1", System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART},
	}}
	server.system = fakePlaySystemClient{response: &gamev1.GetGameSystemResponse{
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
	server.characters = fakePlayCharacterClient{sheetResponse: enrichedCharacterSheetResponse()}
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
	server.daggerheartContent = content

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
	sheet, ok := inspection.Sheet.(playprotocol.DaggerheartCharacterSheetData)
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
	server.characters = fakePlayCharacterClient{sheetResponse: enrichedCharacterSheetResponse()}
	server.daggerheartContent = &fakeDaggerheartContentClient{
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
	sheet := catalog["char-1"].Sheet.(playprotocol.DaggerheartCharacterSheetData)
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
