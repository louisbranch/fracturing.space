package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestContentStoreNilErrors(t *testing.T) {
	ctx := context.Background()
	var s *Store
	now := time.Now()

	if err := s.PutDaggerheartClass(ctx, storage.DaggerheartClass{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected error from nil store PutDaggerheartClass")
	}
	if _, err := s.GetDaggerheartClass(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store GetDaggerheartClass")
	}
	if _, err := s.ListDaggerheartClasses(ctx); err == nil {
		t.Fatal("expected error from nil store ListDaggerheartClasses")
	}
	if err := s.DeleteDaggerheartClass(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store DeleteDaggerheartClass")
	}
	if err := s.PutDaggerheartSubclass(ctx, storage.DaggerheartSubclass{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected error from nil store PutDaggerheartSubclass")
	}
	if _, err := s.GetDaggerheartSubclass(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store GetDaggerheartSubclass")
	}
	if _, err := s.ListDaggerheartSubclasses(ctx); err == nil {
		t.Fatal("expected error from nil store ListDaggerheartSubclasses")
	}
	if err := s.DeleteDaggerheartSubclass(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store DeleteDaggerheartSubclass")
	}
	if err := s.PutDaggerheartHeritage(ctx, storage.DaggerheartHeritage{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected error from nil store PutDaggerheartHeritage")
	}
	if _, err := s.GetDaggerheartHeritage(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store GetDaggerheartHeritage")
	}
	if _, err := s.ListDaggerheartHeritages(ctx); err == nil {
		t.Fatal("expected error from nil store ListDaggerheartHeritages")
	}
	if err := s.DeleteDaggerheartHeritage(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store DeleteDaggerheartHeritage")
	}
	if err := s.PutDaggerheartExperience(ctx, storage.DaggerheartExperienceEntry{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected error from nil store PutDaggerheartExperience")
	}
	if _, err := s.GetDaggerheartExperience(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store GetDaggerheartExperience")
	}
	if _, err := s.ListDaggerheartExperiences(ctx); err == nil {
		t.Fatal("expected error from nil store ListDaggerheartExperiences")
	}
	if err := s.DeleteDaggerheartExperience(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store DeleteDaggerheartExperience")
	}
	if err := s.PutDaggerheartAdversaryEntry(ctx, storage.DaggerheartAdversaryEntry{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected error from nil store PutDaggerheartAdversaryEntry")
	}
	if _, err := s.GetDaggerheartAdversaryEntry(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store GetDaggerheartAdversaryEntry")
	}
	if _, err := s.ListDaggerheartAdversaryEntries(ctx); err == nil {
		t.Fatal("expected error from nil store ListDaggerheartAdversaryEntries")
	}
	if err := s.DeleteDaggerheartAdversaryEntry(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store DeleteDaggerheartAdversaryEntry")
	}
	if err := s.PutDaggerheartBeastform(ctx, storage.DaggerheartBeastformEntry{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected error from nil store PutDaggerheartBeastform")
	}
	if _, err := s.GetDaggerheartBeastform(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store GetDaggerheartBeastform")
	}
	if _, err := s.ListDaggerheartBeastforms(ctx); err == nil {
		t.Fatal("expected error from nil store ListDaggerheartBeastforms")
	}
	if err := s.DeleteDaggerheartBeastform(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store DeleteDaggerheartBeastform")
	}
	if err := s.PutDaggerheartCompanionExperience(ctx, storage.DaggerheartCompanionExperienceEntry{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected error from nil store PutDaggerheartCompanionExperience")
	}
	if _, err := s.GetDaggerheartCompanionExperience(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store GetDaggerheartCompanionExperience")
	}
	if _, err := s.ListDaggerheartCompanionExperiences(ctx); err == nil {
		t.Fatal("expected error from nil store ListDaggerheartCompanionExperiences")
	}
	if err := s.DeleteDaggerheartCompanionExperience(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store DeleteDaggerheartCompanionExperience")
	}
	if err := s.PutDaggerheartLootEntry(ctx, storage.DaggerheartLootEntry{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected error from nil store PutDaggerheartLootEntry")
	}
	if _, err := s.GetDaggerheartLootEntry(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store GetDaggerheartLootEntry")
	}
	if _, err := s.ListDaggerheartLootEntries(ctx); err == nil {
		t.Fatal("expected error from nil store ListDaggerheartLootEntries")
	}
	if err := s.DeleteDaggerheartLootEntry(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store DeleteDaggerheartLootEntry")
	}
	if err := s.PutDaggerheartDamageType(ctx, storage.DaggerheartDamageTypeEntry{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected error from nil store PutDaggerheartDamageType")
	}
	if _, err := s.GetDaggerheartDamageType(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store GetDaggerheartDamageType")
	}
	if _, err := s.ListDaggerheartDamageTypes(ctx); err == nil {
		t.Fatal("expected error from nil store ListDaggerheartDamageTypes")
	}
	if err := s.DeleteDaggerheartDamageType(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store DeleteDaggerheartDamageType")
	}
	if err := s.PutDaggerheartDomain(ctx, storage.DaggerheartDomain{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected error from nil store PutDaggerheartDomain")
	}
	if _, err := s.GetDaggerheartDomain(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store GetDaggerheartDomain")
	}
	if _, err := s.ListDaggerheartDomains(ctx); err == nil {
		t.Fatal("expected error from nil store ListDaggerheartDomains")
	}
	if err := s.DeleteDaggerheartDomain(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store DeleteDaggerheartDomain")
	}
	if err := s.PutDaggerheartDomainCard(ctx, storage.DaggerheartDomainCard{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected error from nil store PutDaggerheartDomainCard")
	}
	if _, err := s.GetDaggerheartDomainCard(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store GetDaggerheartDomainCard")
	}
	if _, err := s.ListDaggerheartDomainCards(ctx); err == nil {
		t.Fatal("expected error from nil store ListDaggerheartDomainCards")
	}
	if _, err := s.ListDaggerheartDomainCardsByDomain(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store ListDaggerheartDomainCardsByDomain")
	}
	if err := s.DeleteDaggerheartDomainCard(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store DeleteDaggerheartDomainCard")
	}
	if err := s.PutDaggerheartWeapon(ctx, storage.DaggerheartWeapon{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected error from nil store PutDaggerheartWeapon")
	}
	if _, err := s.GetDaggerheartWeapon(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store GetDaggerheartWeapon")
	}
	if _, err := s.ListDaggerheartWeapons(ctx); err == nil {
		t.Fatal("expected error from nil store ListDaggerheartWeapons")
	}
	if err := s.DeleteDaggerheartWeapon(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store DeleteDaggerheartWeapon")
	}
	if err := s.PutDaggerheartArmor(ctx, storage.DaggerheartArmor{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected error from nil store PutDaggerheartArmor")
	}
	if _, err := s.GetDaggerheartArmor(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store GetDaggerheartArmor")
	}
	if _, err := s.ListDaggerheartArmor(ctx); err == nil {
		t.Fatal("expected error from nil store ListDaggerheartArmor")
	}
	if err := s.DeleteDaggerheartArmor(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store DeleteDaggerheartArmor")
	}
	if err := s.PutDaggerheartItem(ctx, storage.DaggerheartItem{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected error from nil store PutDaggerheartItem")
	}
	if _, err := s.GetDaggerheartItem(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store GetDaggerheartItem")
	}
	if _, err := s.ListDaggerheartItems(ctx); err == nil {
		t.Fatal("expected error from nil store ListDaggerheartItems")
	}
	if err := s.DeleteDaggerheartItem(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store DeleteDaggerheartItem")
	}
	if err := s.PutDaggerheartEnvironment(ctx, storage.DaggerheartEnvironment{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected error from nil store PutDaggerheartEnvironment")
	}
	if _, err := s.GetDaggerheartEnvironment(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store GetDaggerheartEnvironment")
	}
	if _, err := s.ListDaggerheartEnvironments(ctx); err == nil {
		t.Fatal("expected error from nil store ListDaggerheartEnvironments")
	}
	if err := s.DeleteDaggerheartEnvironment(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store DeleteDaggerheartEnvironment")
	}
	if err := s.PutDaggerheartContentString(ctx, storage.DaggerheartContentString{ContentID: "x", ContentType: "t", Field: "f", Locale: "en"}); err == nil {
		t.Fatal("expected error from nil store PutDaggerheartContentString")
	}
}

func TestContentStoreCancelledContextErrors(t *testing.T) {
	store := openTestContentStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	now := time.Now()

	if err := store.PutDaggerheartClass(ctx, storage.DaggerheartClass{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected context error from PutDaggerheartClass")
	}
	if _, err := store.GetDaggerheartClass(ctx, "x"); err == nil {
		t.Fatal("expected context error from GetDaggerheartClass")
	}
	if _, err := store.ListDaggerheartClasses(ctx); err == nil {
		t.Fatal("expected context error from ListDaggerheartClasses")
	}
	if err := store.DeleteDaggerheartClass(ctx, "x"); err == nil {
		t.Fatal("expected context error from DeleteDaggerheartClass")
	}
	if err := store.PutDaggerheartSubclass(ctx, storage.DaggerheartSubclass{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected context error from PutDaggerheartSubclass")
	}
	if _, err := store.GetDaggerheartSubclass(ctx, "x"); err == nil {
		t.Fatal("expected context error from GetDaggerheartSubclass")
	}
	if _, err := store.ListDaggerheartSubclasses(ctx); err == nil {
		t.Fatal("expected context error from ListDaggerheartSubclasses")
	}
	if err := store.DeleteDaggerheartSubclass(ctx, "x"); err == nil {
		t.Fatal("expected context error from DeleteDaggerheartSubclass")
	}
	if err := store.PutDaggerheartHeritage(ctx, storage.DaggerheartHeritage{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected context error from PutDaggerheartHeritage")
	}
	if _, err := store.GetDaggerheartHeritage(ctx, "x"); err == nil {
		t.Fatal("expected context error from GetDaggerheartHeritage")
	}
	if _, err := store.ListDaggerheartHeritages(ctx); err == nil {
		t.Fatal("expected context error from ListDaggerheartHeritages")
	}
	if err := store.DeleteDaggerheartHeritage(ctx, "x"); err == nil {
		t.Fatal("expected context error from DeleteDaggerheartHeritage")
	}
	if err := store.PutDaggerheartExperience(ctx, storage.DaggerheartExperienceEntry{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected context error from PutDaggerheartExperience")
	}
	if _, err := store.GetDaggerheartExperience(ctx, "x"); err == nil {
		t.Fatal("expected context error from GetDaggerheartExperience")
	}
	if _, err := store.ListDaggerheartExperiences(ctx); err == nil {
		t.Fatal("expected context error from ListDaggerheartExperiences")
	}
	if err := store.DeleteDaggerheartExperience(ctx, "x"); err == nil {
		t.Fatal("expected context error from DeleteDaggerheartExperience")
	}
	if err := store.PutDaggerheartAdversaryEntry(ctx, storage.DaggerheartAdversaryEntry{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected context error from PutDaggerheartAdversaryEntry")
	}
	if _, err := store.GetDaggerheartAdversaryEntry(ctx, "x"); err == nil {
		t.Fatal("expected context error from GetDaggerheartAdversaryEntry")
	}
	if _, err := store.ListDaggerheartAdversaryEntries(ctx); err == nil {
		t.Fatal("expected context error from ListDaggerheartAdversaryEntries")
	}
	if err := store.DeleteDaggerheartAdversaryEntry(ctx, "x"); err == nil {
		t.Fatal("expected context error from DeleteDaggerheartAdversaryEntry")
	}
	if err := store.PutDaggerheartBeastform(ctx, storage.DaggerheartBeastformEntry{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected context error from PutDaggerheartBeastform")
	}
	if _, err := store.GetDaggerheartBeastform(ctx, "x"); err == nil {
		t.Fatal("expected context error from GetDaggerheartBeastform")
	}
	if _, err := store.ListDaggerheartBeastforms(ctx); err == nil {
		t.Fatal("expected context error from ListDaggerheartBeastforms")
	}
	if err := store.DeleteDaggerheartBeastform(ctx, "x"); err == nil {
		t.Fatal("expected context error from DeleteDaggerheartBeastform")
	}
	if err := store.PutDaggerheartCompanionExperience(ctx, storage.DaggerheartCompanionExperienceEntry{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected context error from PutDaggerheartCompanionExperience")
	}
	if _, err := store.GetDaggerheartCompanionExperience(ctx, "x"); err == nil {
		t.Fatal("expected context error from GetDaggerheartCompanionExperience")
	}
	if _, err := store.ListDaggerheartCompanionExperiences(ctx); err == nil {
		t.Fatal("expected context error from ListDaggerheartCompanionExperiences")
	}
	if err := store.DeleteDaggerheartCompanionExperience(ctx, "x"); err == nil {
		t.Fatal("expected context error from DeleteDaggerheartCompanionExperience")
	}
	if err := store.PutDaggerheartLootEntry(ctx, storage.DaggerheartLootEntry{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected context error from PutDaggerheartLootEntry")
	}
	if _, err := store.GetDaggerheartLootEntry(ctx, "x"); err == nil {
		t.Fatal("expected context error from GetDaggerheartLootEntry")
	}
	if _, err := store.ListDaggerheartLootEntries(ctx); err == nil {
		t.Fatal("expected context error from ListDaggerheartLootEntries")
	}
	if err := store.DeleteDaggerheartLootEntry(ctx, "x"); err == nil {
		t.Fatal("expected context error from DeleteDaggerheartLootEntry")
	}
	if err := store.PutDaggerheartDamageType(ctx, storage.DaggerheartDamageTypeEntry{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected context error from PutDaggerheartDamageType")
	}
	if _, err := store.GetDaggerheartDamageType(ctx, "x"); err == nil {
		t.Fatal("expected context error from GetDaggerheartDamageType")
	}
	if _, err := store.ListDaggerheartDamageTypes(ctx); err == nil {
		t.Fatal("expected context error from ListDaggerheartDamageTypes")
	}
	if err := store.DeleteDaggerheartDamageType(ctx, "x"); err == nil {
		t.Fatal("expected context error from DeleteDaggerheartDamageType")
	}
	if err := store.PutDaggerheartDomain(ctx, storage.DaggerheartDomain{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected context error from PutDaggerheartDomain")
	}
	if _, err := store.GetDaggerheartDomain(ctx, "x"); err == nil {
		t.Fatal("expected context error from GetDaggerheartDomain")
	}
	if _, err := store.ListDaggerheartDomains(ctx); err == nil {
		t.Fatal("expected context error from ListDaggerheartDomains")
	}
	if err := store.DeleteDaggerheartDomain(ctx, "x"); err == nil {
		t.Fatal("expected context error from DeleteDaggerheartDomain")
	}
	if err := store.PutDaggerheartDomainCard(ctx, storage.DaggerheartDomainCard{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected context error from PutDaggerheartDomainCard")
	}
	if _, err := store.GetDaggerheartDomainCard(ctx, "x"); err == nil {
		t.Fatal("expected context error from GetDaggerheartDomainCard")
	}
	if _, err := store.ListDaggerheartDomainCards(ctx); err == nil {
		t.Fatal("expected context error from ListDaggerheartDomainCards")
	}
	if _, err := store.ListDaggerheartDomainCardsByDomain(ctx, "x"); err == nil {
		t.Fatal("expected context error from ListDaggerheartDomainCardsByDomain")
	}
	if err := store.DeleteDaggerheartDomainCard(ctx, "x"); err == nil {
		t.Fatal("expected context error from DeleteDaggerheartDomainCard")
	}
	if err := store.PutDaggerheartWeapon(ctx, storage.DaggerheartWeapon{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected context error from PutDaggerheartWeapon")
	}
	if _, err := store.GetDaggerheartWeapon(ctx, "x"); err == nil {
		t.Fatal("expected context error from GetDaggerheartWeapon")
	}
	if _, err := store.ListDaggerheartWeapons(ctx); err == nil {
		t.Fatal("expected context error from ListDaggerheartWeapons")
	}
	if err := store.DeleteDaggerheartWeapon(ctx, "x"); err == nil {
		t.Fatal("expected context error from DeleteDaggerheartWeapon")
	}
	if err := store.PutDaggerheartArmor(ctx, storage.DaggerheartArmor{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected context error from PutDaggerheartArmor")
	}
	if _, err := store.GetDaggerheartArmor(ctx, "x"); err == nil {
		t.Fatal("expected context error from GetDaggerheartArmor")
	}
	if _, err := store.ListDaggerheartArmor(ctx); err == nil {
		t.Fatal("expected context error from ListDaggerheartArmor")
	}
	if err := store.DeleteDaggerheartArmor(ctx, "x"); err == nil {
		t.Fatal("expected context error from DeleteDaggerheartArmor")
	}
	if err := store.PutDaggerheartItem(ctx, storage.DaggerheartItem{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected context error from PutDaggerheartItem")
	}
	if _, err := store.GetDaggerheartItem(ctx, "x"); err == nil {
		t.Fatal("expected context error from GetDaggerheartItem")
	}
	if _, err := store.ListDaggerheartItems(ctx); err == nil {
		t.Fatal("expected context error from ListDaggerheartItems")
	}
	if err := store.DeleteDaggerheartItem(ctx, "x"); err == nil {
		t.Fatal("expected context error from DeleteDaggerheartItem")
	}
	if err := store.PutDaggerheartEnvironment(ctx, storage.DaggerheartEnvironment{ID: "x", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected context error from PutDaggerheartEnvironment")
	}
	if _, err := store.GetDaggerheartEnvironment(ctx, "x"); err == nil {
		t.Fatal("expected context error from GetDaggerheartEnvironment")
	}
	if _, err := store.ListDaggerheartEnvironments(ctx); err == nil {
		t.Fatal("expected context error from ListDaggerheartEnvironments")
	}
	if err := store.DeleteDaggerheartEnvironment(ctx, "x"); err == nil {
		t.Fatal("expected context error from DeleteDaggerheartEnvironment")
	}
	if err := store.PutDaggerheartContentString(ctx, storage.DaggerheartContentString{ContentID: "x", ContentType: "t", Field: "f", Locale: "en"}); err == nil {
		t.Fatal("expected context error from PutDaggerheartContentString")
	}
}

func TestContentStoreEmptyIDValidation(t *testing.T) {
	store := openTestContentStore(t)
	ctx := context.Background()

	if err := store.PutDaggerheartClass(ctx, storage.DaggerheartClass{}); err == nil {
		t.Fatal("expected error for empty class ID")
	}
	if _, err := store.GetDaggerheartClass(ctx, ""); err == nil {
		t.Fatal("expected error for empty class ID in Get")
	}
	if err := store.DeleteDaggerheartClass(ctx, ""); err == nil {
		t.Fatal("expected error for empty class ID in Delete")
	}
	if err := store.PutDaggerheartSubclass(ctx, storage.DaggerheartSubclass{}); err == nil {
		t.Fatal("expected error for empty subclass ID")
	}
	if _, err := store.GetDaggerheartSubclass(ctx, ""); err == nil {
		t.Fatal("expected error for empty subclass ID in Get")
	}
	if err := store.DeleteDaggerheartSubclass(ctx, ""); err == nil {
		t.Fatal("expected error for empty subclass ID in Delete")
	}
	if err := store.PutDaggerheartHeritage(ctx, storage.DaggerheartHeritage{}); err == nil {
		t.Fatal("expected error for empty heritage ID")
	}
	if _, err := store.GetDaggerheartHeritage(ctx, ""); err == nil {
		t.Fatal("expected error for empty heritage ID in Get")
	}
	if err := store.DeleteDaggerheartHeritage(ctx, ""); err == nil {
		t.Fatal("expected error for empty heritage ID in Delete")
	}
	if err := store.PutDaggerheartExperience(ctx, storage.DaggerheartExperienceEntry{}); err == nil {
		t.Fatal("expected error for empty experience ID")
	}
	if _, err := store.GetDaggerheartExperience(ctx, ""); err == nil {
		t.Fatal("expected error for empty experience ID in Get")
	}
	if err := store.DeleteDaggerheartExperience(ctx, ""); err == nil {
		t.Fatal("expected error for empty experience ID in Delete")
	}
	if err := store.PutDaggerheartAdversaryEntry(ctx, storage.DaggerheartAdversaryEntry{}); err == nil {
		t.Fatal("expected error for empty adversary entry ID")
	}
	if _, err := store.GetDaggerheartAdversaryEntry(ctx, ""); err == nil {
		t.Fatal("expected error for empty adversary entry ID in Get")
	}
	if err := store.DeleteDaggerheartAdversaryEntry(ctx, ""); err == nil {
		t.Fatal("expected error for empty adversary entry ID in Delete")
	}
	if err := store.PutDaggerheartBeastform(ctx, storage.DaggerheartBeastformEntry{}); err == nil {
		t.Fatal("expected error for empty beastform ID")
	}
	if _, err := store.GetDaggerheartBeastform(ctx, ""); err == nil {
		t.Fatal("expected error for empty beastform ID in Get")
	}
	if err := store.DeleteDaggerheartBeastform(ctx, ""); err == nil {
		t.Fatal("expected error for empty beastform ID in Delete")
	}
	if err := store.PutDaggerheartWeapon(ctx, storage.DaggerheartWeapon{}); err == nil {
		t.Fatal("expected error for empty weapon ID")
	}
	if _, err := store.GetDaggerheartWeapon(ctx, ""); err == nil {
		t.Fatal("expected error for empty weapon ID in Get")
	}
	if err := store.DeleteDaggerheartWeapon(ctx, ""); err == nil {
		t.Fatal("expected error for empty weapon ID in Delete")
	}
	if err := store.PutDaggerheartArmor(ctx, storage.DaggerheartArmor{}); err == nil {
		t.Fatal("expected error for empty armor ID")
	}
	if _, err := store.GetDaggerheartArmor(ctx, ""); err == nil {
		t.Fatal("expected error for empty armor ID in Get")
	}
	if err := store.DeleteDaggerheartArmor(ctx, ""); err == nil {
		t.Fatal("expected error for empty armor ID in Delete")
	}
	if err := store.PutDaggerheartItem(ctx, storage.DaggerheartItem{}); err == nil {
		t.Fatal("expected error for empty item ID")
	}
	if _, err := store.GetDaggerheartItem(ctx, ""); err == nil {
		t.Fatal("expected error for empty item ID in Get")
	}
	if err := store.DeleteDaggerheartItem(ctx, ""); err == nil {
		t.Fatal("expected error for empty item ID in Delete")
	}
	if err := store.PutDaggerheartEnvironment(ctx, storage.DaggerheartEnvironment{}); err == nil {
		t.Fatal("expected error for empty environment ID")
	}
	if _, err := store.GetDaggerheartEnvironment(ctx, ""); err == nil {
		t.Fatal("expected error for empty environment ID in Get")
	}
	if err := store.DeleteDaggerheartEnvironment(ctx, ""); err == nil {
		t.Fatal("expected error for empty environment ID in Delete")
	}
	if err := store.PutDaggerheartDomain(ctx, storage.DaggerheartDomain{}); err == nil {
		t.Fatal("expected error for empty domain ID")
	}
	if _, err := store.GetDaggerheartDomain(ctx, ""); err == nil {
		t.Fatal("expected error for empty domain ID in Get")
	}
	if err := store.DeleteDaggerheartDomain(ctx, ""); err == nil {
		t.Fatal("expected error for empty domain ID in Delete")
	}
	if err := store.PutDaggerheartDomainCard(ctx, storage.DaggerheartDomainCard{}); err == nil {
		t.Fatal("expected error for empty domain card ID")
	}
	if _, err := store.GetDaggerheartDomainCard(ctx, ""); err == nil {
		t.Fatal("expected error for empty domain card ID in Get")
	}
	if err := store.DeleteDaggerheartDomainCard(ctx, ""); err == nil {
		t.Fatal("expected error for empty domain card ID in Delete")
	}
	if _, err := store.ListDaggerheartDomainCardsByDomain(ctx, ""); err == nil {
		t.Fatal("expected error for empty domain ID in ListDaggerheartDomainCardsByDomain")
	}
	if err := store.PutDaggerheartCompanionExperience(ctx, storage.DaggerheartCompanionExperienceEntry{}); err == nil {
		t.Fatal("expected error for empty companion experience ID")
	}
	if _, err := store.GetDaggerheartCompanionExperience(ctx, ""); err == nil {
		t.Fatal("expected error for empty companion experience ID in Get")
	}
	if err := store.DeleteDaggerheartCompanionExperience(ctx, ""); err == nil {
		t.Fatal("expected error for empty companion experience ID in Delete")
	}
	if err := store.PutDaggerheartLootEntry(ctx, storage.DaggerheartLootEntry{}); err == nil {
		t.Fatal("expected error for empty loot entry ID")
	}
	if _, err := store.GetDaggerheartLootEntry(ctx, ""); err == nil {
		t.Fatal("expected error for empty loot entry ID in Get")
	}
	if err := store.DeleteDaggerheartLootEntry(ctx, ""); err == nil {
		t.Fatal("expected error for empty loot entry ID in Delete")
	}
	if err := store.PutDaggerheartDamageType(ctx, storage.DaggerheartDamageTypeEntry{}); err == nil {
		t.Fatal("expected error for empty damage type ID")
	}
	if _, err := store.GetDaggerheartDamageType(ctx, ""); err == nil {
		t.Fatal("expected error for empty damage type ID in Get")
	}
	if err := store.DeleteDaggerheartDamageType(ctx, ""); err == nil {
		t.Fatal("expected error for empty damage type ID in Delete")
	}
}

func TestContentStoreNotFoundPaths(t *testing.T) {
	store := openTestContentStore(t)
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetDaggerheartClass", func() error { _, err := store.GetDaggerheartClass(ctx, "nope"); return err }},
		{"GetDaggerheartSubclass", func() error { _, err := store.GetDaggerheartSubclass(ctx, "nope"); return err }},
		{"GetDaggerheartHeritage", func() error { _, err := store.GetDaggerheartHeritage(ctx, "nope"); return err }},
		{"GetDaggerheartExperience", func() error { _, err := store.GetDaggerheartExperience(ctx, "nope"); return err }},
		{"GetDaggerheartAdversaryEntry", func() error { _, err := store.GetDaggerheartAdversaryEntry(ctx, "nope"); return err }},
		{"GetDaggerheartBeastform", func() error { _, err := store.GetDaggerheartBeastform(ctx, "nope"); return err }},
		{"GetDaggerheartCompanionExperience", func() error { _, err := store.GetDaggerheartCompanionExperience(ctx, "nope"); return err }},
		{"GetDaggerheartLootEntry", func() error { _, err := store.GetDaggerheartLootEntry(ctx, "nope"); return err }},
		{"GetDaggerheartDamageType", func() error { _, err := store.GetDaggerheartDamageType(ctx, "nope"); return err }},
		{"GetDaggerheartDomain", func() error { _, err := store.GetDaggerheartDomain(ctx, "nope"); return err }},
		{"GetDaggerheartDomainCard", func() error { _, err := store.GetDaggerheartDomainCard(ctx, "nope"); return err }},
		{"GetDaggerheartWeapon", func() error { _, err := store.GetDaggerheartWeapon(ctx, "nope"); return err }},
		{"GetDaggerheartArmor", func() error { _, err := store.GetDaggerheartArmor(ctx, "nope"); return err }},
		{"GetDaggerheartItem", func() error { _, err := store.GetDaggerheartItem(ctx, "nope"); return err }},
		{"GetDaggerheartEnvironment", func() error { _, err := store.GetDaggerheartEnvironment(ctx, "nope"); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil || !errors.Is(err, storage.ErrNotFound) {
				t.Fatalf("expected ErrNotFound, got %v", err)
			}
		})
	}
}

func TestContentStoreEmptyLists(t *testing.T) {
	store := openTestContentStore(t)
	ctx := context.Background()

	if list, err := store.ListDaggerheartClasses(ctx); err != nil || len(list) != 0 {
		t.Fatalf("expected empty class list, got len=%d err=%v", len(list), err)
	}
	if list, err := store.ListDaggerheartSubclasses(ctx); err != nil || len(list) != 0 {
		t.Fatalf("expected empty subclass list, got len=%d err=%v", len(list), err)
	}
	if list, err := store.ListDaggerheartHeritages(ctx); err != nil || len(list) != 0 {
		t.Fatalf("expected empty heritage list, got len=%d err=%v", len(list), err)
	}
	if list, err := store.ListDaggerheartExperiences(ctx); err != nil || len(list) != 0 {
		t.Fatalf("expected empty experience list, got len=%d err=%v", len(list), err)
	}
	if list, err := store.ListDaggerheartAdversaryEntries(ctx); err != nil || len(list) != 0 {
		t.Fatalf("expected empty adversary entry list, got len=%d err=%v", len(list), err)
	}
	if list, err := store.ListDaggerheartBeastforms(ctx); err != nil || len(list) != 0 {
		t.Fatalf("expected empty beastform list, got len=%d err=%v", len(list), err)
	}
	if list, err := store.ListDaggerheartCompanionExperiences(ctx); err != nil || len(list) != 0 {
		t.Fatalf("expected empty companion experience list, got len=%d err=%v", len(list), err)
	}
	if list, err := store.ListDaggerheartLootEntries(ctx); err != nil || len(list) != 0 {
		t.Fatalf("expected empty loot entry list, got len=%d err=%v", len(list), err)
	}
	if list, err := store.ListDaggerheartDamageTypes(ctx); err != nil || len(list) != 0 {
		t.Fatalf("expected empty damage type list, got len=%d err=%v", len(list), err)
	}
	if list, err := store.ListDaggerheartDomains(ctx); err != nil || len(list) != 0 {
		t.Fatalf("expected empty domain list, got len=%d err=%v", len(list), err)
	}
	if list, err := store.ListDaggerheartDomainCards(ctx); err != nil || len(list) != 0 {
		t.Fatalf("expected empty domain card list, got len=%d err=%v", len(list), err)
	}
	if list, err := store.ListDaggerheartWeapons(ctx); err != nil || len(list) != 0 {
		t.Fatalf("expected empty weapon list, got len=%d err=%v", len(list), err)
	}
	if list, err := store.ListDaggerheartArmor(ctx); err != nil || len(list) != 0 {
		t.Fatalf("expected empty armor list, got len=%d err=%v", len(list), err)
	}
	if list, err := store.ListDaggerheartItems(ctx); err != nil || len(list) != 0 {
		t.Fatalf("expected empty item list, got len=%d err=%v", len(list), err)
	}
	if list, err := store.ListDaggerheartEnvironments(ctx); err != nil || len(list) != 0 {
		t.Fatalf("expected empty environment list, got len=%d err=%v", len(list), err)
	}
	if list, err := store.ListDaggerheartDomainCardsByDomain(ctx, "no-domain"); err != nil || len(list) != 0 {
		t.Fatalf("expected empty domain cards by domain list, got len=%d err=%v", len(list), err)
	}
}
