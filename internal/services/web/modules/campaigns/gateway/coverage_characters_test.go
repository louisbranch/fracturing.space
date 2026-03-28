package gateway

import (
	"context"
	"net/http"
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestCampaignCharacterLoadsOwnerAndDaggerheartSummary(t *testing.T) {
	t.Parallel()

	characters := &fakeCharacterWorkflowClient{
		sheetResp: &statev1.GetCharacterSheetResponse{
			Character: &statev1.Character{
				Id:                 "char-1",
				Name:               "  Aria  ",
				Kind:               statev1.CharacterKind_PC,
				OwnerParticipantId: wrapperspb.String("p1"),
				Pronouns:           pronouns.ToProto(pronouns.PronounTheyThem),
				Aliases:            []string{"  Scout  ", "Blade"},
				AvatarSetId:        "set-a",
				AvatarAssetId:      "asset-1",
			},
		},
		profilesResp: &statev1.ListCharacterProfilesResponse{
			Profiles: []*statev1.CharacterProfile{{
				CharacterId: "char-1",
				SystemProfile: &statev1.CharacterProfile_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{
					Level:      3,
					ClassId:    "class-1",
					SubclassId: "sub-1",
					Heritage: &daggerheartv1.DaggerheartHeritageSelection{
						FirstFeatureAncestryId: "anc-1",
						CommunityId:            "com-1",
					},
				}},
			}},
		},
	}
	participants := &contractParticipantClient{
		listResp: &statev1.ListParticipantsResponse{
			Participants: []*statev1.Participant{{
				Id:     "p1",
				UserId: "user-1",
				Name:   "Lead",
			}},
		},
	}
	content := &fakeDaggerheartContentClient{
		resp: &daggerheartv1.GetDaggerheartContentCatalogResponse{
			Catalog: &daggerheartv1.DaggerheartContentCatalog{
				Classes:    []*daggerheartv1.DaggerheartClass{{Id: "class-1", Name: "Warrior"}},
				Subclasses: []*daggerheartv1.DaggerheartSubclass{{Id: "sub-1", Name: "Guardian"}},
				Heritages: []*daggerheartv1.DaggerheartHeritage{
					{Id: "anc-1", Name: "Ridgeborn", Kind: daggerheartv1.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_ANCESTRY},
					{Id: "com-1", Name: "Lorekeepers", Kind: daggerheartv1.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_COMMUNITY},
				},
			},
		},
	}

	gateway := characterReadGateway{
		read: CharacterReadDeps{
			Character:          characters,
			Participant:        participants,
			DaggerheartContent: content,
		},
		assetBaseURL: "https://cdn.example.test",
	}

	got, err := gateway.CampaignCharacter(context.Background(), " camp-1 ", " char-1 ", campaignapp.CharacterReadContext{
		System:       "Daggerheart",
		Locale:       language.BrazilianPortuguese,
		ViewerUserID: "user-1",
	})
	if err != nil {
		t.Fatalf("CampaignCharacter() error = %v", err)
	}
	if got.ID != "char-1" || got.Name != "Aria" || got.Owner != "Lead" {
		t.Fatalf("character = %#v", got)
	}
	if !got.OwnedByViewer {
		t.Fatalf("OwnedByViewer = false, want true")
	}
	if got.Pronouns != pronouns.PronounTheyThem {
		t.Fatalf("Pronouns = %q, want %q", got.Pronouns, pronouns.PronounTheyThem)
	}
	if got.Daggerheart == nil || got.Daggerheart.ClassName != "Warrior" || got.Daggerheart.SubclassName != "Guardian" || got.Daggerheart.HeritageName != "Ridgeborn" || got.Daggerheart.CommunityName != "Lorekeepers" {
		t.Fatalf("Daggerheart summary = %#v", got.Daggerheart)
	}
	if !strings.Contains(got.AvatarURL, "https://cdn.example.test") {
		t.Fatalf("AvatarURL = %q, want CDN base", got.AvatarURL)
	}
	if characters.lastSheetReq == nil || characters.lastSheetReq.GetCampaignId() != "camp-1" || characters.lastSheetReq.GetCharacterId() != "char-1" {
		t.Fatalf("GetCharacterSheet req = %#v", characters.lastSheetReq)
	}
	if characters.lastProfilesReq == nil || characters.lastProfilesReq.GetCampaignId() != "camp-1" {
		t.Fatalf("ListCharacterProfiles req = %#v", characters.lastProfilesReq)
	}
	if content.lastReq == nil || content.lastReq.GetLocale() != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("GetContentCatalog locale = %v, want %v", content.lastReq.GetLocale(), commonv1.Locale_LOCALE_PT_BR)
	}
}

func TestCampaignCharacterValidationAndNotFoundPaths(t *testing.T) {
	t.Parallel()

	if _, err := (characterReadGateway{}).CampaignCharacter(context.Background(), "camp-1", "char-1", campaignapp.CharacterReadContext{}); err == nil {
		t.Fatal("expected unavailable error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}

	gateway := characterReadGateway{read: CharacterReadDeps{Character: &fakeCharacterWorkflowClient{}}}
	if _, err := gateway.CampaignCharacter(context.Background(), " ", "char-1", campaignapp.CharacterReadContext{}); err == nil {
		t.Fatal("expected invalid input error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}

	notFoundGateway := characterReadGateway{read: CharacterReadDeps{Character: &fakeCharacterWorkflowClient{sheetResp: &statev1.GetCharacterSheetResponse{}}}}
	if _, err := notFoundGateway.CampaignCharacter(context.Background(), "camp-1", "char-1", campaignapp.CharacterReadContext{}); err == nil {
		t.Fatal("expected not found error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusNotFound {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusNotFound)
	}
}

func TestCharacterMutationGatewaysTrimAndValidateRequests(t *testing.T) {
	t.Parallel()

	client := &fakeCharacterWorkflowClient{}
	mutationGateway := characterMutationGateway{mutation: CharacterMutationDeps{Character: client}}
	ownershipGateway := characterOwnershipMutationGateway{mutation: CharacterOwnershipMutationDeps{Character: client}}

	if err := mutationGateway.UpdateCharacter(context.Background(), " camp-1 ", " char-1 ", campaignapp.UpdateCharacterInput{Name: "  Aria  ", Pronouns: " she/her "}); err != nil {
		t.Fatalf("UpdateCharacter() error = %v", err)
	}
	if client.updateReq == nil {
		t.Fatal("expected update request")
	}
	if client.updateReq.GetCampaignId() != "camp-1" || client.updateReq.GetCharacterId() != "char-1" || client.updateReq.GetName().GetValue() != "Aria" {
		t.Fatalf("UpdateCharacter req = %#v", client.updateReq)
	}
	if got := pronouns.FromProto(client.updateReq.GetPronouns()); got != pronouns.PronounSheHer {
		t.Fatalf("UpdateCharacter pronouns = %q, want %q", got, pronouns.PronounSheHer)
	}

	if err := mutationGateway.DeleteCharacter(context.Background(), " camp-1 ", " char-1 "); err != nil {
		t.Fatalf("DeleteCharacter() error = %v", err)
	}
	if client.deleteReq == nil || client.deleteReq.GetCampaignId() != "camp-1" || client.deleteReq.GetCharacterId() != "char-1" {
		t.Fatalf("DeleteCharacter req = %#v", client.deleteReq)
	}

	if err := ownershipGateway.SetCharacterOwner(context.Background(), " camp-1 ", " char-1 ", " p1 "); err != nil {
		t.Fatalf("SetCharacterOwner() error = %v", err)
	}
	if client.updateReq == nil || client.updateReq.GetOwnerParticipantId().GetValue() != "p1" {
		t.Fatalf("SetCharacterOwner req = %#v", client.updateReq)
	}
}

func TestCharacterMutationGatewaysMapValidationAndTransportErrors(t *testing.T) {
	t.Parallel()

	if err := (characterMutationGateway{}).UpdateCharacter(context.Background(), "camp-1", "char-1", campaignapp.UpdateCharacterInput{Name: "Aria"}); err == nil {
		t.Fatal("expected unavailable error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}

	mutationGateway := characterMutationGateway{mutation: CharacterMutationDeps{Character: &fakeCharacterWorkflowClient{}}}
	if err := mutationGateway.UpdateCharacter(context.Background(), "camp-1", "char-1", campaignapp.UpdateCharacterInput{}); err == nil {
		t.Fatal("expected invalid input error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}

	failingClient := &fakeCharacterWorkflowClient{
		updateErr: status.Error(codes.NotFound, "missing"),
		deleteErr: status.Error(codes.NotFound, "missing"),
	}
	failingMutationGateway := characterMutationGateway{mutation: CharacterMutationDeps{Character: failingClient}}
	if err := failingMutationGateway.UpdateCharacter(context.Background(), "camp-1", "char-1", campaignapp.UpdateCharacterInput{Name: "Aria"}); err == nil {
		t.Fatal("expected mapped update error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusNotFound {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusNotFound)
	}
	if err := failingMutationGateway.DeleteCharacter(context.Background(), "camp-1", "char-1"); err == nil {
		t.Fatal("expected mapped delete error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusNotFound {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusNotFound)
	}

	ownershipGateway := characterOwnershipMutationGateway{mutation: CharacterOwnershipMutationDeps{Character: &fakeCharacterWorkflowClient{updateErr: status.Error(codes.FailedPrecondition, "conflict")}}}
	if err := ownershipGateway.SetCharacterOwner(context.Background(), "camp-1", "char-1", "p1"); err == nil {
		t.Fatal("expected mapped owner error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusConflict {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusConflict)
	}
}

func TestMapProfileCompanionExperiencesPreservesIdentityAndTrimsValues(t *testing.T) {
	t.Parallel()

	got := mapProfileCompanionExperiences([]*daggerheartv1.DaggerheartCompanionExperience{
		nil,
		{ExperienceId: " exp-1 ", Name: " Keen nose ", Modifier: 2},
	})
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].ID != "exp-1" || got[0].Name != "Keen nose" || got[0].Modifier != "2" {
		t.Fatalf("mapped companion experiences = %#v", got)
	}
}
