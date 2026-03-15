package daggerheart

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"

func newContentTestService() *DaggerheartContentService {
	cs := newFakeContentStore()
	cs.classes["class-1"] = contentstore.DaggerheartClass{ID: "class-1", Name: "Guardian"}
	cs.classes["class-2"] = contentstore.DaggerheartClass{ID: "class-2", Name: "Sorcerer"}
	cs.subclasses["sub-1"] = contentstore.DaggerheartSubclass{ID: "sub-1", Name: "Bladeweaver"}
	cs.heritages["her-1"] = contentstore.DaggerheartHeritage{ID: "her-1", Name: "Elf", Kind: "ancestry"}
	cs.experiences["exp-1"] = contentstore.DaggerheartExperienceEntry{ID: "exp-1", Name: "Wanderer"}
	cs.adversaryEntries["adv-1"] = contentstore.DaggerheartAdversaryEntry{ID: "adv-1", Name: "Goblin"}
	cs.beastforms["beast-1"] = contentstore.DaggerheartBeastformEntry{ID: "beast-1", Name: "Wolf"}
	cs.companionExperiences["cexp-1"] = contentstore.DaggerheartCompanionExperienceEntry{ID: "cexp-1", Name: "Guard"}
	cs.lootEntries["loot-1"] = contentstore.DaggerheartLootEntry{ID: "loot-1", Name: "Gold"}
	cs.damageTypes["dt-1"] = contentstore.DaggerheartDamageTypeEntry{ID: "dt-1", Name: "Fire"}
	cs.domains["dom-1"] = contentstore.DaggerheartDomain{ID: "dom-1", Name: "Valor"}
	cs.domainCards["card-1"] = contentstore.DaggerheartDomainCard{ID: "card-1", Name: "Fireball", DomainID: "dom-1"}
	cs.weapons["weap-1"] = contentstore.DaggerheartWeapon{ID: "weap-1", Name: "Blade"}
	cs.armor["armor-1"] = contentstore.DaggerheartArmor{ID: "armor-1", Name: "Chain Mail"}
	cs.items["item-1"] = contentstore.DaggerheartItem{ID: "item-1", Name: "Potion"}
	cs.environments["env-1"] = contentstore.DaggerheartEnvironment{ID: "env-1", Name: "Forest"}

	svc, err := NewDaggerheartContentService(cs)
	if err != nil {
		panic(err)
	}
	return svc
}
