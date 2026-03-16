package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/discovery/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/discovery/storage"
	discoverysqlite "github.com/louisbranch/fracturing.space/internal/services/discovery/storage/sqlite"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func reconcileBuiltinStarterTemplates(
	ctx context.Context,
	store *discoverysqlite.Store,
	campaignClient gamev1.CampaignServiceClient,
	characterClient gamev1.CharacterServiceClient,
) error {
	if store == nil || campaignClient == nil || characterClient == nil {
		return nil
	}

	starters, err := catalog.BuiltinStarters()
	if err != nil {
		return fmt.Errorf("load builtin starters: %w", err)
	}
	for _, starter := range starters {
		if err := store.UpsertBuiltinDiscoveryEntry(ctx, starter.Entry); err != nil {
			return fmt.Errorf("upsert builtin starter %q: %w", starter.Entry.EntryID, err)
		}

		entry, err := store.GetDiscoveryEntry(ctx, starter.Entry.EntryID)
		if err != nil {
			return fmt.Errorf("load builtin starter %q: %w", starter.Entry.EntryID, err)
		}
		if templateCampaignExists(ctx, campaignClient, entry.SourceID) {
			continue
		}

		templateCampaignID, err := createStarterTemplateCampaign(ctx, campaignClient, characterClient, starter)
		if err != nil {
			return fmt.Errorf("create starter template %q: %w", starter.Entry.EntryID, err)
		}
		if err := store.UpdateDiscoveryEntrySourceID(ctx, starter.Entry.EntryID, templateCampaignID, time.Now().UTC()); err != nil {
			return fmt.Errorf("persist starter template source_id %q: %w", starter.Entry.EntryID, err)
		}
	}
	return nil
}

func templateCampaignExists(ctx context.Context, client gamev1.CampaignServiceClient, campaignID string) bool {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" || client == nil {
		return false
	}
	resp, err := client.GetCampaign(ctx, &gamev1.GetCampaignRequest{CampaignId: campaignID})
	return err == nil && resp != nil && resp.GetCampaign() != nil && strings.TrimSpace(resp.GetCampaign().GetId()) != ""
}

func createStarterTemplateCampaign(
	ctx context.Context,
	campaignClient gamev1.CampaignServiceClient,
	characterClient gamev1.CharacterServiceClient,
	starter catalog.StarterDefinition,
) (string, error) {
	createResp, err := campaignClient.CreateCampaign(ctx, &gamev1.CreateCampaignRequest{
		Name:         starter.Entry.Title,
		System:       commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:       gamev1.GmMode_AI,
		Intent:       gamev1.CampaignIntent_STARTER,
		AccessPolicy: gamev1.CampaignAccessPolicy_PUBLIC,
		ThemePrompt:  starterThemePrompt(starter.Entry),
		Locale:       commonv1.Locale_LOCALE_EN_US,
	})
	if err != nil {
		return "", err
	}
	if createResp == nil || createResp.GetCampaign() == nil {
		return "", fmt.Errorf("create campaign returned no campaign")
	}
	if createResp.GetOwnerParticipant() == nil || strings.TrimSpace(createResp.GetOwnerParticipant().GetId()) == "" {
		return "", fmt.Errorf("create campaign returned no owner participant")
	}

	campaignID := strings.TrimSpace(createResp.GetCampaign().GetId())
	ownerParticipantID := strings.TrimSpace(createResp.GetOwnerParticipant().GetId())
	participantCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(grpcmeta.ParticipantIDHeader, ownerParticipantID))
	archiveOnFailure := func(err error) error {
		archiveStarterTemplateCampaign(participantCtx, campaignClient, campaignID)
		return err
	}

	createCharacterResp, err := characterClient.CreateCharacter(participantCtx, &gamev1.CreateCharacterRequest{
		CampaignId: campaignID,
		Name:       starter.Character.Name,
		Pronouns:   sharedpronouns.ToProto(starter.Character.Pronouns),
		Kind:       gamev1.CharacterKind_PC,
		Notes:      starter.Character.Summary,
	})
	if err != nil {
		return "", archiveOnFailure(err)
	}
	if createCharacterResp == nil || createCharacterResp.GetCharacter() == nil {
		return "", archiveOnFailure(fmt.Errorf("create character returned no character"))
	}
	characterID := strings.TrimSpace(createCharacterResp.GetCharacter().GetId())
	if characterID == "" {
		return "", archiveOnFailure(fmt.Errorf("create character returned empty character id"))
	}

	workflow := &daggerheartv1.DaggerheartCreationWorkflowInput{
		ClassSubclassInput: &daggerheartv1.DaggerheartCreationStepClassSubclassInput{
			ClassId:    starter.Character.ClassID,
			SubclassId: starter.Character.SubclassID,
		},
		HeritageInput: &daggerheartv1.DaggerheartCreationStepHeritageInput{
			Heritage: &daggerheartv1.DaggerheartCreationStepHeritageSelectionInput{
				FirstFeatureAncestryId:  starter.Character.AncestryID,
				SecondFeatureAncestryId: starter.Character.AncestryID,
				CommunityId:             starter.Character.CommunityID,
			},
		},
		TraitsInput: &daggerheartv1.DaggerheartCreationStepTraitsInput{
			Agility:   starter.Character.Traits.Agility,
			Strength:  starter.Character.Traits.Strength,
			Finesse:   starter.Character.Traits.Finesse,
			Instinct:  starter.Character.Traits.Instinct,
			Presence:  starter.Character.Traits.Presence,
			Knowledge: starter.Character.Traits.Knowledge,
		},
		DetailsInput: &daggerheartv1.DaggerheartCreationStepDetailsInput{
			Description: starter.Character.Description,
		},
		EquipmentInput: &daggerheartv1.DaggerheartCreationStepEquipmentInput{
			WeaponIds:    append([]string(nil), starter.Character.WeaponIDs...),
			ArmorId:      starter.Character.ArmorID,
			PotionItemId: starter.Character.PotionItemID,
		},
		BackgroundInput: &daggerheartv1.DaggerheartCreationStepBackgroundInput{
			Background: starter.Character.Background,
		},
		ExperiencesInput: &daggerheartv1.DaggerheartCreationStepExperiencesInput{
			Experiences: toDaggerheartExperiences(starter.Character.Experiences),
		},
		DomainCardsInput: &daggerheartv1.DaggerheartCreationStepDomainCardsInput{
			DomainCardIds: append([]string(nil), starter.Character.DomainCardIDs...),
		},
		ConnectionsInput: &daggerheartv1.DaggerheartCreationStepConnectionsInput{
			Connections: starter.Character.Connections,
		},
	}
	_, err = characterClient.ApplyCharacterCreationWorkflow(participantCtx, &gamev1.ApplyCharacterCreationWorkflowRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
		SystemWorkflow: &gamev1.ApplyCharacterCreationWorkflowRequest_Daggerheart{
			Daggerheart: workflow,
		},
	})
	if err != nil {
		return "", archiveOnFailure(err)
	}
	return campaignID, nil
}

func archiveStarterTemplateCampaign(ctx context.Context, client gamev1.CampaignServiceClient, campaignID string) {
	campaignID = strings.TrimSpace(campaignID)
	if client == nil || campaignID == "" {
		return
	}
	_, _ = client.ArchiveCampaign(ctx, &gamev1.ArchiveCampaignRequest{CampaignId: campaignID})
}

func isRetryableStarterReconciliationError(err error) bool {
	if err == nil {
		return false
	}
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	switch st.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.Aborted:
		return true
	case codes.InvalidArgument:
		return strings.Contains(st.Message(), " is not found")
	default:
		return false
	}
}

func toDaggerheartExperiences(src []catalog.StarterExperienceDefinition) []*daggerheartv1.DaggerheartExperience {
	out := make([]*daggerheartv1.DaggerheartExperience, 0, len(src))
	for _, experience := range src {
		name := strings.TrimSpace(experience.Name)
		if name == "" {
			continue
		}
		out = append(out, &daggerheartv1.DaggerheartExperience{
			Name:     name,
			Modifier: experience.Modifier,
		})
	}
	return out
}

func starterThemePrompt(entry storage.DiscoveryEntry) string {
	if theme := strings.TrimSpace(entry.CampaignTheme); theme != "" {
		return theme
	}
	return strings.TrimSpace(entry.Description)
}
