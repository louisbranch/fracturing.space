package game

import (
	"context"
	"errors"
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/readiness"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestListAllCharactersByCampaign_NilStore(t *testing.T) {
	_, err := listAllCharactersByCampaign(context.Background(), nil, "c1")
	assertStatusCode(t, err, codes.Internal)
}

func TestListAllCharactersByCampaign_CollectsPages(t *testing.T) {
	store := &readinessCharacterPagingStore{
		pages: map[string]storage.CharacterPage{
			"": {
				Characters: []storage.CharacterRecord{
					{ID: "char-1", CampaignID: "c1"},
				},
				NextPageToken: "next",
			},
			"next": {
				Characters: []storage.CharacterRecord{
					{ID: "char-2", CampaignID: "c1"},
				},
			},
		},
	}

	characters, err := listAllCharactersByCampaign(context.Background(), store, "c1")
	if err != nil {
		t.Fatalf("listAllCharactersByCampaign() error = %v", err)
	}
	if len(characters) != 2 {
		t.Fatalf("len(characters) = %d, want 2", len(characters))
	}
	if strings.TrimSpace(characters[0].ID) != "char-1" || strings.TrimSpace(characters[1].ID) != "char-2" {
		t.Fatalf("character order = [%s, %s], want [char-1, char-2]", characters[0].ID, characters[1].ID)
	}
}

func TestListAllCharactersByCampaign_RepeatedPageTokenFails(t *testing.T) {
	store := &readinessCharacterPagingStore{
		pages: map[string]storage.CharacterPage{
			"": {
				NextPageToken: "loop",
			},
			"loop": {
				NextPageToken: "loop",
			},
		},
	}

	_, err := listAllCharactersByCampaign(context.Background(), store, "c1")
	assertStatusCode(t, err, codes.Internal)
}

func TestCampaignHasActiveSession_NilStore(t *testing.T) {
	_, err := campaignHasActiveSession(context.Background(), nil, "c1")
	assertStatusCode(t, err, codes.Internal)
}

func TestCampaignHasActiveSession_NoActiveReturnsFalse(t *testing.T) {
	store := newFakeSessionStore()

	hasActive, err := campaignHasActiveSession(context.Background(), store, "c1")
	if err != nil {
		t.Fatalf("campaignHasActiveSession() error = %v", err)
	}
	if hasActive {
		t.Fatal("hasActive = true, want false")
	}
}

func TestCampaignHasActiveSession_ActiveReturnsTrue(t *testing.T) {
	store := newFakeSessionStore()
	store.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {
			ID:         "s1",
			CampaignID: "c1",
			Status:     session.StatusActive,
		},
	}
	store.activeSession["c1"] = "s1"

	hasActive, err := campaignHasActiveSession(context.Background(), store, "c1")
	if err != nil {
		t.Fatalf("campaignHasActiveSession() error = %v", err)
	}
	if !hasActive {
		t.Fatal("hasActive = false, want true")
	}
}

func TestCampaignHasActiveSession_LoadError(t *testing.T) {
	store := newFakeSessionStore()
	store.activeErr = errors.New("boom")

	_, err := campaignHasActiveSession(context.Background(), store, "c1")
	assertStatusCode(t, err, codes.Internal)
}

func TestCampaignReadinessAggregateState_DaggerheartStoreRequired(t *testing.T) {
	_, err := campaignReadinessAggregateState(
		context.Background(),
		nil,
		storage.CampaignRecord{
			ID:     "c1",
			Status: campaign.StatusActive,
			GmMode: campaign.GmModeHuman,
			System: bridge.SystemIDDaggerheart,
		},
		nil,
		[]storage.CharacterRecord{{ID: "char-1", CampaignID: "c1"}},
	)
	assertStatusCode(t, err, codes.Internal)
}

func TestCampaignReadinessAggregateState_DaggerheartProfileLoadError(t *testing.T) {
	daggerheartStore := newFakeDaggerheartStore()
	daggerheartStore.getErr = errors.New("boom")

	_, err := campaignReadinessAggregateState(
		context.Background(),
		daggerheartStore,
		storage.CampaignRecord{
			ID:     "c1",
			Status: campaign.StatusActive,
			GmMode: campaign.GmModeHuman,
			System: bridge.SystemIDDaggerheart,
		},
		nil,
		[]storage.CharacterRecord{{ID: "char-1", CampaignID: "c1"}},
	)
	assertStatusCode(t, err, codes.Internal)
}

func TestCampaignReadinessAggregateState_DaggerheartProfileMapped(t *testing.T) {
	daggerheartStore := newFakeDaggerheartStore()
	daggerheartStore.profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"char-1": {
			CampaignID:  "c1",
			CharacterID: "char-1",
			Level:       2,
		},
	}

	state, err := campaignReadinessAggregateState(
		context.Background(),
		daggerheartStore,
		storage.CampaignRecord{
			ID:     "c1",
			Status: campaign.StatusActive,
			GmMode: campaign.GmModeHuman,
			System: bridge.SystemIDDaggerheart,
		},
		[]storage.ParticipantRecord{
			{
				ID:             "player-1",
				CampaignID:     "c1",
				UserID:         "user-1",
				Role:           participant.RolePlayer,
				Controller:     participant.ControllerHuman,
				CampaignAccess: participant.CampaignAccessMember,
			},
		},
		[]storage.CharacterRecord{
			{
				ID:            "char-1",
				CampaignID:    "c1",
				Name:          "Aria",
				ParticipantID: "player-1",
			},
		},
	)
	if err != nil {
		t.Fatalf("campaignReadinessAggregateState() error = %v", err)
	}

	characterState, ok := state.Characters["char-1"]
	if !ok {
		t.Fatal("character state for char-1 not found")
	}
	if characterState.Name != "Aria" {
		t.Fatalf("character name = %q, want %q", characterState.Name, "Aria")
	}

	systemState, ok := state.Systems[module.Key{ID: daggerheartdomain.SystemID, Version: daggerheartdomain.SystemVersion}]
	if !ok {
		t.Fatal("daggerheart system state not found")
	}
	snapshot, ok := systemState.(daggerheartdomain.SnapshotState)
	if !ok {
		t.Fatalf("daggerheart system state type = %T, want SnapshotState", systemState)
	}
	if got := snapshot.CharacterProfiles["char-1"].Level; got != 2 {
		t.Fatalf("system profile level = %v, want 2", got)
	}
}

func TestResolveReadinessLocale(t *testing.T) {
	tests := []struct {
		name      string
		requested commonv1.Locale
		campaign  commonv1.Locale
		expected  commonv1.Locale
	}{
		{
			name:      "requested locale wins",
			requested: commonv1.Locale_LOCALE_PT_BR,
			campaign:  commonv1.Locale_LOCALE_EN_US,
			expected:  commonv1.Locale_LOCALE_PT_BR,
		},
		{
			name:      "campaign locale fallback",
			requested: commonv1.Locale_LOCALE_UNSPECIFIED,
			campaign:  commonv1.Locale_LOCALE_PT_BR,
			expected:  commonv1.Locale_LOCALE_PT_BR,
		},
		{
			name:      "default locale fallback",
			requested: commonv1.Locale_LOCALE_UNSPECIFIED,
			campaign:  commonv1.Locale_LOCALE_UNSPECIFIED,
			expected:  commonv1.Locale_LOCALE_EN_US,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveReadinessLocale(tc.requested, tc.campaign)
			if got != tc.expected {
				t.Fatalf("resolveReadinessLocale(%v, %v) = %v, want %v", tc.requested, tc.campaign, got, tc.expected)
			}
		})
	}
}

func TestReadinessBlockerMetadataValueOrDefault(t *testing.T) {
	metadata := map[string]string{
		"status": " active ",
	}

	if got := readinessBlockerMetadataValueOrDefault(metadata, "status", "unspecified"); got != "active" {
		t.Fatalf("readinessBlockerMetadataValueOrDefault(status) = %q, want %q", got, "active")
	}
	if got := readinessBlockerMetadataValueOrDefault(metadata, "missing", "fallback"); got != "fallback" {
		t.Fatalf("readinessBlockerMetadataValueOrDefault(missing) = %q, want %q", got, "fallback")
	}
}

func TestReadinessBlockerToProto_CopiesMetadataAndTrimsCode(t *testing.T) {
	blocker := readiness.Blocker{
		Code:    " code-1 ",
		Message: "message",
		Metadata: map[string]string{
			"key": "value",
		},
	}

	proto := readinessBlockerToProto(commonv1.Locale_LOCALE_EN_US, blocker)
	if strings.TrimSpace(proto.GetCode()) != "code-1" {
		t.Fatalf("proto.code = %q, want %q", proto.GetCode(), "code-1")
	}
	if proto.GetMetadata()["key"] != "value" {
		t.Fatalf("proto.metadata[key] = %q, want %q", proto.GetMetadata()["key"], "value")
	}

	blocker.Metadata["key"] = "mutated"
	if proto.GetMetadata()["key"] != "value" {
		t.Fatalf("proto metadata mutated by input map, got %q want %q", proto.GetMetadata()["key"], "value")
	}
}

type readinessCharacterPagingStore struct {
	pages     map[string]storage.CharacterPage
	errByPage map[string]error
}

func (s *readinessCharacterPagingStore) PutCharacter(context.Context, storage.CharacterRecord) error {
	return nil
}

func (s *readinessCharacterPagingStore) GetCharacter(context.Context, string, string) (storage.CharacterRecord, error) {
	return storage.CharacterRecord{}, storage.ErrNotFound
}

func (s *readinessCharacterPagingStore) DeleteCharacter(context.Context, string, string) error {
	return nil
}

func (s *readinessCharacterPagingStore) CountCharacters(context.Context, string) (int, error) {
	return 0, nil
}

func (s *readinessCharacterPagingStore) ListCharactersByOwnerParticipant(context.Context, string, string) ([]storage.CharacterRecord, error) {
	return nil, nil
}

func (s *readinessCharacterPagingStore) ListCharacters(_ context.Context, _ string, _ int, pageToken string) (storage.CharacterPage, error) {
	if s.errByPage != nil {
		if err := s.errByPage[pageToken]; err != nil {
			return storage.CharacterPage{}, err
		}
	}
	if s.pages == nil {
		return storage.CharacterPage{}, nil
	}
	if page, ok := s.pages[pageToken]; ok {
		return page, nil
	}
	return storage.CharacterPage{}, nil
}

func systemStoresWithDaggerheart(store projectionstore.Store) SystemStores {
	return SystemStores{Daggerheart: store}
}
