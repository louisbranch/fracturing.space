package charactertransport

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/runtimekit"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	daggerhearttestkit "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/testkit"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
)

// assertStatusCode verifies the gRPC status code for an error.
func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %v, got nil", want)
	}
	if _, ok := status.FromError(err); !ok {
		err = grpcerror.HandleDomainError(err)
	}
	grpcassert.StatusCode(t, err, want)
}

// testRuntime is a shared write-path runtime configured once for all tests.
var testRuntime *domainwrite.Runtime

func TestMain(m *testing.M) {
	testRuntime = runtimekit.SetupRuntime()
	os.Exit(m.Run())
}

// --- test helpers ---

func mustJSON(t *testing.T, payload any) []byte {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}
	return data
}

// testStoresBuilder reduces repeated store construction in tests.
type testStoresBuilder struct {
	Campaign    *gametest.FakeCampaignStore
	Participant *gametest.FakeParticipantStore
	Event       *gametest.FakeEventStore
	Character   *gametest.FakeCharacterStore
	Daggerheart *daggerhearttestkit.FakeDaggerheartStore
	Content     contentstore.DaggerheartContentReadStore

	domain       handler.Domain
	writeRuntime bool
}

func newTestStores() *testStoresBuilder {
	return &testStoresBuilder{
		Campaign:    gametest.NewFakeCampaignStore(),
		Participant: gametest.NewFakeParticipantStore(),
		Event:       gametest.NewFakeEventStore(),
	}
}

func (b *testStoresBuilder) withCharacter() *testStoresBuilder {
	b.Character = gametest.NewFakeCharacterStore()
	b.Daggerheart = daggerhearttestkit.NewFakeDaggerheartStore()
	return b
}

func (b *testStoresBuilder) withDomain(d handler.Domain) *testStoresBuilder {
	b.domain = d
	b.writeRuntime = true
	return b
}

func (b *testStoresBuilder) build() Deps {
	d := Deps{
		Auth: authz.PolicyDeps{
			Participant: b.Participant,
		},
		Campaign:    b.Campaign,
		Participant: b.Participant,
	}
	if b.Character != nil {
		d.Character = b.Character
		d.Auth.Character = b.Character
	}
	if b.Daggerheart != nil {
		d.Daggerheart = b.Daggerheart
		adapters, _ := manifest.AdapterRegistry(manifest.ProjectionStores{Daggerheart: b.Daggerheart})
		d.Applier = projection.Applier{
			Campaign:  b.Campaign,
			Character: b.Character,
			Adapters:  adapters,
		}
	}
	if b.domain != nil {
		d.Write.Executor = b.domain
	}
	if b.writeRuntime {
		d.Write.Runtime = testRuntime
	}
	if b.Content != nil {
		d.DaggerheartContent = b.Content
	}
	return d
}

type fakeDomainEngine struct {
	store         storage.EventStore
	result        engine.Result
	resultsByType map[command.Type]engine.Result
	calls         int
	lastCommand   command.Command
	commands      []command.Command
}

func (f *fakeDomainEngine) Execute(ctx context.Context, cmd command.Command) (engine.Result, error) {
	f.calls++
	f.lastCommand = cmd
	f.commands = append(f.commands, cmd)

	result := f.result
	if len(f.resultsByType) > 0 {
		if selected, ok := f.resultsByType[cmd.Type]; ok {
			result = selected
		}
	}
	if f.store == nil {
		return result, nil
	}
	if len(result.Decision.Events) == 0 {
		return result, nil
	}
	stored := make([]event.Event, 0, len(result.Decision.Events))
	for _, evt := range result.Decision.Events {
		storedEvent, err := f.store.AppendEvent(ctx, evt)
		if err != nil {
			return engine.Result{}, err
		}
		stored = append(stored, storedEvent)
	}
	result.Decision.Events = stored
	return result, nil
}

func testDaggerheartProfile(overrides func(*daggerheartstate.CharacterProfile)) daggerheartstate.CharacterProfile {
	profile := daggerheartstate.CharacterProfile{
		Level:           1,
		HpMax:           6,
		StressMax:       6,
		Evasion:         10,
		MajorThreshold:  1,
		SevereThreshold: 2,
		Proficiency:     1,
		ArmorScore:      0,
		ArmorMax:        0,
	}
	if overrides != nil {
		overrides(&profile)
	}
	return profile
}

func testDaggerheartProfileReplacedEvent(
	t *testing.T,
	now time.Time,
	campaignID, characterID string,
	actorType event.ActorType,
	actorID string,
	profile daggerheartstate.CharacterProfile,
) event.Event {
	t.Helper()
	return event.Event{
		CampaignID:    ids.CampaignID(campaignID),
		Type:          daggerheartpayload.EventTypeCharacterProfileReplaced,
		Timestamp:     now,
		ActorType:     actorType,
		ActorID:       actorID,
		EntityType:    "character",
		EntityID:      characterID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON: mustJSON(t, daggerheartstate.CharacterProfileReplacedPayload{
			CharacterID: ids.CharacterID(characterID),
			Profile:     profile,
		}),
	}
}

func testCreateCharacterResults(
	t *testing.T,
	now time.Time,
	campaignID, characterID string,
	actorType event.ActorType,
	actorID string,
	createPayload any,
) map[command.Type]engine.Result {
	t.Helper()
	return map[command.Type]engine.Result{
		commandids.CharacterCreate: {
			Decision: command.Accept(event.Event{
				CampaignID:  ids.CampaignID(campaignID),
				Type:        event.Type("character.created"),
				Timestamp:   now,
				ActorType:   actorType,
				ActorID:     actorID,
				EntityType:  "character",
				EntityID:    characterID,
				PayloadJSON: mustJSON(t, createPayload),
			}),
		},
	}
}

func newWorkflowCharacterService(t *testing.T, profile projectionstore.DaggerheartCharacterProfile) *Service {
	t.Helper()

	participantStore := characterManagerParticipantStore("c1")
	characterStore := gametest.NewFakeCharacterStore()
	characterStore.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {
			ID:                 "ch1",
			CampaignID:         "c1",
			OwnerParticipantID: "manager-1",
			Name:               "Hero",
			Kind:               character.KindPC,
		},
	}

	dhStore := daggerhearttestkit.NewFakeDaggerheartStore()
	if profile.CharacterID != "" {
		dhStore.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
			"ch1": profile,
		}
	}

	domain := &fakeDomainEngine{result: engine.Result{Decision: command.Accept()}}
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
		System: bridge.SystemIDDaggerheart,
	}

	return NewService(Deps{
		Auth:        authz.PolicyDeps{Participant: participantStore, Character: characterStore},
		Campaign:    campaignStore,
		Participant: participantStore,
		Character:   characterStore,
		Daggerheart: dhStore,
		DaggerheartContent: workflowContentStore{
			classes: map[string]contentstore.DaggerheartClass{
				"class.guardian": {
					ID:              "class.guardian",
					Name:            "Guardian",
					StartingEvasion: 9,
					StartingHP:      7,
					DomainIDs:       []string{"domain.valor", "domain.blade"},
				},
			},
			subclasses: map[string]contentstore.DaggerheartSubclass{
				"subclass.stalwart": {ID: "subclass.stalwart", ClassID: "class.guardian", Name: "Stalwart"},
			},
			heritages: map[string]contentstore.DaggerheartHeritage{
				"heritage.ancestry.clank": {
					ID:   "heritage.ancestry.clank",
					Kind: "ancestry",
					Name: "Clank",
					Features: []contentstore.DaggerheartFeature{
						{ID: "heritage.ancestry.clank.feature-1", Name: "Clank One"},
						{ID: "heritage.ancestry.clank.feature-2", Name: "Clank Two"},
					},
				},
				"heritage.community.farmer": {ID: "heritage.community.farmer", Kind: "community", Name: "Farmer"},
			},
			weapons: map[string]contentstore.DaggerheartWeapon{
				"weapon.longsword": {ID: "weapon.longsword", Tier: 1, Category: "primary", Burden: 2},
			},
			armors: map[string]contentstore.DaggerheartArmor{
				"armor.gambeson-armor": {ID: "armor.gambeson-armor", Tier: 1, ArmorScore: 1, BaseMajorThreshold: 8, BaseSevereThreshold: 14},
			},
			items: map[string]contentstore.DaggerheartItem{
				"item.minor-health-potion":  {ID: "item.minor-health-potion"},
				"item.minor-stamina-potion": {ID: "item.minor-stamina-potion"},
			},
			domainCards: map[string]contentstore.DaggerheartDomainCard{
				"domain-card.ward":         {ID: "domain-card.ward", DomainID: "domain.valor", Name: "Ward", Level: 1},
				"domain-card.arcana-bolt":  {ID: "domain-card.arcana-bolt", DomainID: "domain.arcana", Name: "Arcana Bolt", Level: 1},
				"domain-card.blade-strike": {ID: "domain-card.blade-strike", DomainID: "domain.blade", Name: "Blade Strike", Level: 1},
			},
		},
		Write: domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
	})
}

func testCreationHeritageInput() *daggerheartv1.DaggerheartCreationStepHeritageSelectionInput {
	return &daggerheartv1.DaggerheartCreationStepHeritageSelectionInput{
		FirstFeatureAncestryId:  "heritage.ancestry.clank",
		SecondFeatureAncestryId: "heritage.ancestry.clank",
		CommunityId:             "heritage.community.farmer",
	}
}

func testProjectionHeritage() projectionstore.DaggerheartHeritageSelection {
	return projectionstore.DaggerheartHeritageSelection{
		FirstFeatureAncestryID:  "heritage.ancestry.clank",
		FirstFeatureID:          "heritage.ancestry.clank.feature-1",
		SecondFeatureAncestryID: "heritage.ancestry.clank",
		SecondFeatureID:         "heritage.ancestry.clank.feature-2",
		CommunityID:             "heritage.community.farmer",
	}
}

type workflowContentStore struct {
	contentstore.DaggerheartContentReadStore
	classes     map[string]contentstore.DaggerheartClass
	subclasses  map[string]contentstore.DaggerheartSubclass
	heritages   map[string]contentstore.DaggerheartHeritage
	weapons     map[string]contentstore.DaggerheartWeapon
	armors      map[string]contentstore.DaggerheartArmor
	items       map[string]contentstore.DaggerheartItem
	domainCards map[string]contentstore.DaggerheartDomainCard
}

func (s workflowContentStore) GetDaggerheartClass(_ context.Context, id string) (contentstore.DaggerheartClass, error) {
	class, ok := s.classes[id]
	if !ok {
		return contentstore.DaggerheartClass{}, storage.ErrNotFound
	}
	return class, nil
}

func (s workflowContentStore) GetDaggerheartSubclass(_ context.Context, id string) (contentstore.DaggerheartSubclass, error) {
	subclass, ok := s.subclasses[id]
	if !ok {
		return contentstore.DaggerheartSubclass{}, storage.ErrNotFound
	}
	return subclass, nil
}

func (s workflowContentStore) GetDaggerheartHeritage(_ context.Context, id string) (contentstore.DaggerheartHeritage, error) {
	heritage, ok := s.heritages[id]
	if !ok {
		return contentstore.DaggerheartHeritage{}, storage.ErrNotFound
	}
	return heritage, nil
}

func (s workflowContentStore) GetDaggerheartDomainCard(_ context.Context, id string) (contentstore.DaggerheartDomainCard, error) {
	card, ok := s.domainCards[id]
	if !ok {
		return contentstore.DaggerheartDomainCard{}, storage.ErrNotFound
	}
	return card, nil
}

func (s workflowContentStore) GetDaggerheartWeapon(_ context.Context, id string) (contentstore.DaggerheartWeapon, error) {
	weapon, ok := s.weapons[id]
	if !ok {
		return contentstore.DaggerheartWeapon{}, storage.ErrNotFound
	}
	return weapon, nil
}

func (s workflowContentStore) GetDaggerheartArmor(_ context.Context, id string) (contentstore.DaggerheartArmor, error) {
	armor, ok := s.armors[id]
	if !ok {
		return contentstore.DaggerheartArmor{}, storage.ErrNotFound
	}
	return armor, nil
}

func (s workflowContentStore) GetDaggerheartItem(_ context.Context, id string) (contentstore.DaggerheartItem, error) {
	item, ok := s.items[id]
	if !ok {
		return contentstore.DaggerheartItem{}, storage.ErrNotFound
	}
	return item, nil
}

// Ensure fakeDomainEngine satisfies the interface at compile time.
var _ handler.Domain = (*fakeDomainEngine)(nil)
