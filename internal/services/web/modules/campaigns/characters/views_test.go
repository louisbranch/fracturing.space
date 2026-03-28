package characters

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/text/message"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigndetail "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/detail"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

type testLocalizer map[string]string

func (l testLocalizer) Sprintf(key message.Reference, args ...any) string {
	ref := fmt.Sprint(key)
	if value, ok := l[ref]; ok {
		return value
	}
	return ref
}

func TestCharacterViewsMapWorkspaceState(t *testing.T) {
	t.Parallel()

	page := &campaigndetail.PageContext{
		Workspace: campaignapp.CampaignWorkspace{
			ID:     "camp-1",
			Name:   "Starfall",
			System: "daggerheart",
			Locale: "pt-BR",
		},
		CanManageSession: true,
		CanManageInvites: true,
		Loc: testLocalizer{
			"game.characters.title":            "Characters",
			"game.characters.submit_create":    "Add character",
			"game.characters.action_edit_page": "Edit character",
			"game.character_detail.title":      "Character detail",
		},
	}
	character := campaignapp.CampaignCharacter{
		ID:                      "char-1",
		Name:                    "Aria",
		Kind:                    "pc",
		Controller:              "human",
		ControllerParticipantID: "part-1",
		Pronouns:                "she/her",
		Aliases:                 []string{"Starseer"},
		AvatarURL:               "/avatar.png",
		OwnedByViewer:           true,
		CanEdit:                 true,
		EditReasonCode:          "allowed",
		Daggerheart: &campaignapp.CampaignCharacterDaggerheartSummary{
			Level:         2,
			ClassName:     "Wizard",
			SubclassName:  "Storm",
			HeritageName:  "Human",
			CommunityName: "Port City",
		},
	}
	control := campaignapp.CampaignCharacterControl{
		CurrentParticipantName: "Mira",
		CanSelfClaim:           true,
		CanSelfRelease:         true,
		CanManageControl:       true,
		Options: []campaignapp.CampaignCharacterControlOption{
			{ParticipantID: "part-1", Label: "Mira", Selected: true},
		},
	}
	editor := campaignapp.CampaignCharacterEditor{Character: character}

	listView := charactersView(page, "camp-1", []campaignapp.CampaignCharacter{character}, true, true)
	if !listView.CanCreateCharacter || !listView.CharacterCreationEnabled || len(listView.Characters) != 1 {
		t.Fatalf("charactersView() = %#v", listView)
	}

	createView := characterCreateView(page, "camp-1")
	if !createView.CanCreateCharacter || createView.CharacterEditor.Kind != "PC" {
		t.Fatalf("characterCreateView() = %#v", createView)
	}

	editView := characterEditView(page, "camp-1", "char-1", editor)
	if editView.CharacterID != "char-1" || editView.Character.Name != "Aria" {
		t.Fatalf("characterEditView() = %#v", editView)
	}

	detailView := characterDetailView(page, "camp-1", "char-1", character, control, true, campaignrender.CampaignCharacterCreationView{})
	if detailView.CharacterID != "char-1" || detailView.CharacterControl.CurrentParticipantName != "Mira" || !detailView.CharacterCreationEnabled {
		t.Fatalf("characterDetailView() = %#v", detailView)
	}

	if got := campaignCharacterBreadcrumbLabel(page.Loc, detailView); got != "Aria" {
		t.Fatalf("campaignCharacterBreadcrumbLabel() = %q, want %q", got, "Aria")
	}
	if got := campaignCharacterEditBreadcrumbLabel(page.Loc, editView); got != "Aria" {
		t.Fatalf("campaignCharacterEditBreadcrumbLabel() = %q, want %q", got, "Aria")
	}

	if got := charactersBreadcrumbs(page); len(got) != 1 || got[0].Label != "Characters" {
		t.Fatalf("charactersBreadcrumbs() = %#v", got)
	}
	if got := characterCreateBreadcrumbs(page, "camp-1"); len(got) != 2 || got[0].URL != routepath.AppCampaignCharacters("camp-1") {
		t.Fatalf("characterCreateBreadcrumbs() = %#v", got)
	}
	if got := characterEditBreadcrumbs(page, "camp-1", "char-1", editView); len(got) != 3 || got[1].URL != routepath.AppCampaignCharacter("camp-1", "char-1") {
		t.Fatalf("characterEditBreadcrumbs() = %#v", got)
	}
	if got := characterDetailBreadcrumbs(page, "camp-1", detailView); len(got) != 2 || got[1].Label != "Aria" {
		t.Fatalf("characterDetailBreadcrumbs() = %#v", got)
	}
}

func TestRegisterStableRoutesRegistersCharacterSurface(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	RegisterStableRoutes(mux, Handler{})

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: routepath.AppCampaignCharacters("camp-1")},
		{method: http.MethodGet, path: routepath.AppCampaignCharacterCreate("camp-1")},
		{method: http.MethodPost, path: routepath.AppCampaignCharacterCreate("camp-1")},
		{method: http.MethodGet, path: routepath.AppCampaignCharacter("camp-1", "char-1")},
		{method: http.MethodPost, path: routepath.AppCampaignCharacterCreationStep("camp-1", "char-1")},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		if _, pattern := mux.Handler(req); pattern == "" {
			t.Fatalf("route %s %s was not registered", tc.method, tc.path)
		}
	}
}
