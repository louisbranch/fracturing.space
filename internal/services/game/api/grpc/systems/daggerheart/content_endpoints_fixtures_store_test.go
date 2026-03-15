package daggerheart

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
)

// fakeContentStore implements contentstore.DaggerheartContentStore for testing content
// service endpoints. Only the methods needed for tests are implemented with
// real behavior; others return nil/empty to satisfy the interface.
type fakeContentStore struct {
	classes              map[string]contentstore.DaggerheartClass
	subclasses           map[string]contentstore.DaggerheartSubclass
	heritages            map[string]contentstore.DaggerheartHeritage
	experiences          map[string]contentstore.DaggerheartExperienceEntry
	adversaryEntries     map[string]contentstore.DaggerheartAdversaryEntry
	beastforms           map[string]contentstore.DaggerheartBeastformEntry
	companionExperiences map[string]contentstore.DaggerheartCompanionExperienceEntry
	lootEntries          map[string]contentstore.DaggerheartLootEntry
	damageTypes          map[string]contentstore.DaggerheartDamageTypeEntry
	domains              map[string]contentstore.DaggerheartDomain
	domainCards          map[string]contentstore.DaggerheartDomainCard
	weapons              map[string]contentstore.DaggerheartWeapon
	armor                map[string]contentstore.DaggerheartArmor
	items                map[string]contentstore.DaggerheartItem
	environments         map[string]contentstore.DaggerheartEnvironment
	contentStrings       map[fakeContentStringKey]contentstore.DaggerheartContentString
}

type fakeContentStringKey struct {
	ContentID string
	Field     string
	Locale    string
}

func newFakeContentStore() *fakeContentStore {
	return &fakeContentStore{
		classes:              make(map[string]contentstore.DaggerheartClass),
		subclasses:           make(map[string]contentstore.DaggerheartSubclass),
		heritages:            make(map[string]contentstore.DaggerheartHeritage),
		experiences:          make(map[string]contentstore.DaggerheartExperienceEntry),
		adversaryEntries:     make(map[string]contentstore.DaggerheartAdversaryEntry),
		beastforms:           make(map[string]contentstore.DaggerheartBeastformEntry),
		companionExperiences: make(map[string]contentstore.DaggerheartCompanionExperienceEntry),
		lootEntries:          make(map[string]contentstore.DaggerheartLootEntry),
		damageTypes:          make(map[string]contentstore.DaggerheartDamageTypeEntry),
		domains:              make(map[string]contentstore.DaggerheartDomain),
		domainCards:          make(map[string]contentstore.DaggerheartDomainCard),
		weapons:              make(map[string]contentstore.DaggerheartWeapon),
		armor:                make(map[string]contentstore.DaggerheartArmor),
		items:                make(map[string]contentstore.DaggerheartItem),
		environments:         make(map[string]contentstore.DaggerheartEnvironment),
		contentStrings:       make(map[fakeContentStringKey]contentstore.DaggerheartContentString),
	}
}

func (s *fakeContentStore) ListDaggerheartContentStrings(_ context.Context, contentType string, contentIDs []string, locale string) ([]contentstore.DaggerheartContentString, error) {
	if s == nil {
		return nil, nil
	}
	if len(contentIDs) == 0 {
		return nil, nil
	}
	idSet := make(map[string]struct{}, len(contentIDs))
	for _, id := range contentIDs {
		idSet[id] = struct{}{}
	}
	results := make([]contentstore.DaggerheartContentString, 0, len(contentIDs))
	for _, entry := range s.contentStrings {
		if entry.ContentType != contentType || entry.Locale != locale {
			continue
		}
		if _, ok := idSet[entry.ContentID]; !ok {
			continue
		}
		results = append(results, entry)
	}
	return results, nil
}

func (s *fakeContentStore) PutDaggerheartContentString(_ context.Context, entry contentstore.DaggerheartContentString) error {
	if s == nil {
		return nil
	}
	key := fakeContentStringKey{ContentID: entry.ContentID, Field: entry.Field, Locale: entry.Locale}
	s.contentStrings[key] = entry
	return nil
}
