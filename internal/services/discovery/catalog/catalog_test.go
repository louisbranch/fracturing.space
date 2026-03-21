package catalog

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/characterworkflow"
	daggerheartcreation "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/creationworkflow"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sqlitedaggerheartcontent "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/daggerheartcontent"
	catalogimporter "github.com/louisbranch/fracturing.space/internal/tools/importer/content/daggerheart/v1"
)

func TestBuiltinEntries_ReturnsThreeEntries(t *testing.T) {
	entries, err := BuiltinEntries()
	if err != nil {
		t.Fatalf("BuiltinEntries: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3", len(entries))
	}
}

func TestBuiltinEntries_StarterCampaignShape(t *testing.T) {
	entries, err := BuiltinEntries()
	if err != nil {
		t.Fatalf("BuiltinEntries: %v", err)
	}
	for i, e := range entries {
		if e.EntryID == "" {
			t.Fatalf("entries[%d] missing entry id", i)
		}
		if e.Kind != discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER {
			t.Fatalf("entries[%d].kind = %v, want CAMPAIGN_STARTER", i, e.Kind)
		}
		if e.DifficultyTier != discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER {
			t.Fatalf("entries[%d].difficulty = %v, want BEGINNER", i, e.DifficultyTier)
		}
		if e.GmMode != discoveryv1.DiscoveryGmMode_DISCOVERY_GM_MODE_AI {
			t.Fatalf("entries[%d].gm_mode = %v, want AI", i, e.GmMode)
		}
		if e.Intent != discoveryv1.DiscoveryIntent_DISCOVERY_INTENT_STARTER {
			t.Fatalf("entries[%d].intent = %v, want STARTER", i, e.Intent)
		}
		if e.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
			t.Fatalf("entries[%d].system = %v, want DAGGERHEART", i, e.System)
		}
		if e.Storyline == "" {
			t.Fatalf("entries[%d].storyline is empty", i)
		}
		if e.CampaignTheme == "" {
			t.Fatalf("entries[%d].campaign_theme is empty", i)
		}
		if e.PreviewHook == "" || e.PreviewCharacterName == "" || e.PreviewCharacterSummary == "" {
			t.Fatalf("entries[%d] missing structured preview data", i)
		}
	}
}

func TestBuiltinEntries_ReturnsDeepCopies(t *testing.T) {
	a, err := BuiltinEntries()
	if err != nil {
		t.Fatalf("BuiltinEntries: %v", err)
	}
	a[0].Title = "mutated"
	a[0].Tags[0] = "mutated"

	b, err := BuiltinEntries()
	if err != nil {
		t.Fatalf("BuiltinEntries: %v", err)
	}
	if b[0].Title == "mutated" {
		t.Fatal("title mutation leaked")
	}
	if b[0].Tags[0] == "mutated" {
		t.Fatal("tags mutation leaked")
	}
}

func TestBuiltinStarters_PremadeCharacterShape(t *testing.T) {
	starters, err := BuiltinStarters()
	if err != nil {
		t.Fatalf("BuiltinStarters: %v", err)
	}
	for i, starter := range starters {
		if starter.Character.Name == "" || starter.Character.ClassID == "" || starter.Character.SubclassID == "" {
			t.Fatalf("starters[%d] missing premade character identity", i)
		}
		if starter.Character.AncestryID == "" || starter.Character.CommunityID == "" {
			t.Fatalf("starters[%d] missing premade heritage", i)
		}
		switch starter.Character.PotionItemID {
		case "item.minor-health-potion", "item.minor-stamina-potion":
		default:
			t.Fatalf("starters[%d].potion_item_id = %q, want allowed starter potion", i, starter.Character.PotionItemID)
		}
		if len(starter.Character.DomainCardIDs) == 0 {
			t.Fatalf("starters[%d] missing domain cards", i)
		}
	}
}

func TestBuiltinStarters_AreWorkflowValid(t *testing.T) {
	store := openImportedDaggerheartContentStore(t)
	provider := daggerheartcreation.CreationWorkflowProvider{}

	starters, err := BuiltinStarters()
	if err != nil {
		t.Fatalf("BuiltinStarters: %v", err)
	}

	for _, starter := range starters {
		starter := starter
		t.Run(starter.Entry.EntryID, func(t *testing.T) {
			deps := &starterWorkflowDeps{content: store}
			_, progress, err := provider.ApplyWorkflow(
				context.Background(),
				deps,
				characterworkflow.CampaignContext{
					ID:     "starter-contract-campaign",
					System: bridge.SystemIDDaggerheart,
					Status: campaign.StatusActive,
				},
				starterWorkflowRequest(starter),
			)
			if err != nil {
				t.Fatalf("ApplyWorkflow(%q) error = %v", starter.Entry.EntryID, err)
			}
			if !progress.Ready {
				t.Fatalf("ApplyWorkflow(%q) ready = false, unmet = %v", starter.Entry.EntryID, progress.UnmetReasons)
			}
			if deps.replaceCalls != 1 {
				t.Fatalf("ExecuteProfileReplace(%q) call count = %d, want 1", starter.Entry.EntryID, deps.replaceCalls)
			}
		})
	}
}

type starterWorkflowDeps struct {
	content      contentstore.DaggerheartContentReadStore
	replaceCalls int
	savedProfile daggerheartstate.CharacterProfile
}

func (d *starterWorkflowDeps) GetCharacterRecord(context.Context, string, string) (characterworkflow.CharacterContext, error) {
	return characterworkflow.CharacterContext{Kind: character.KindPC}, nil
}

func (d *starterWorkflowDeps) GetCharacterSystemProfile(context.Context, string, string) (projectionstore.DaggerheartCharacterProfile, error) {
	return projectionstore.DaggerheartCharacterProfile{}, storage.ErrNotFound
}

func (d *starterWorkflowDeps) SystemContent() contentstore.DaggerheartContentReadStore {
	return d.content
}

func (d *starterWorkflowDeps) ExecuteProfileReplace(_ context.Context, _ characterworkflow.CampaignContext, _ string, profile daggerheartstate.CharacterProfile) error {
	d.replaceCalls++
	d.savedProfile = profile
	return nil
}

func (d *starterWorkflowDeps) ExecuteProfileDelete(context.Context, characterworkflow.CampaignContext, string) error {
	return nil
}

func (d *starterWorkflowDeps) RequireReadPolicy(context.Context, characterworkflow.CampaignContext) error {
	return nil
}

func (d *starterWorkflowDeps) ProfileToProto(string, string, projectionstore.DaggerheartCharacterProfile) *statev1.CharacterProfile {
	return &statev1.CharacterProfile{}
}

func openImportedDaggerheartContentStore(t *testing.T) *sqlitedaggerheartcontent.Store {
	t.Helper()

	contentDir := filepath.Join("..", "..", "..", "tools", "importer", "content", "daggerheart", "v1")
	dbPath := filepath.Join(t.TempDir(), "game-content.db")
	if err := catalogimporter.Run(context.Background(), catalogimporter.Config{
		Dir:        contentDir,
		DBPath:     dbPath,
		BaseLocale: "en-US",
	}, io.Discard); err != nil {
		t.Fatalf("catalogimporter.Run() error = %v", err)
	}

	store, err := sqlitedaggerheartcontent.Open(dbPath)
	if err != nil {
		t.Fatalf("open content store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close content store: %v", err)
		}
	})
	return store
}

func starterWorkflowRequest(starter StarterDefinition) *statev1.ApplyCharacterCreationWorkflowRequest {
	return &statev1.ApplyCharacterCreationWorkflowRequest{
		CampaignId:  "starter-contract-campaign",
		CharacterId: "starter-contract-character",
		SystemWorkflow: &statev1.ApplyCharacterCreationWorkflowRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCreationWorkflowInput{
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
					Experiences: starterWorkflowExperiences(starter.Character.Experiences),
				},
				DomainCardsInput: &daggerheartv1.DaggerheartCreationStepDomainCardsInput{
					DomainCardIds: append([]string(nil), starter.Character.DomainCardIDs...),
				},
				ConnectionsInput: &daggerheartv1.DaggerheartCreationStepConnectionsInput{
					Connections: starter.Character.Connections,
				},
			},
		},
	}
}

func starterWorkflowExperiences(src []StarterExperienceDefinition) []*daggerheartv1.DaggerheartExperience {
	out := make([]*daggerheartv1.DaggerheartExperience, 0, len(src))
	for _, experience := range src {
		out = append(out, &daggerheartv1.DaggerheartExperience{
			Name:     experience.Name,
			Modifier: experience.Modifier,
		})
	}
	return out
}
