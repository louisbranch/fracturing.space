package daggerheart

import (
	"sort"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns"
)

// AssembleCatalog builds the Daggerheart-specific catalog view from generic
// gateway data (progress, catalog, profile).
func (Workflow) AssembleCatalog(
	progress campaigns.CampaignCharacterCreationProgress,
	catalog campaigns.CampaignCharacterCreationCatalog,
	profile campaigns.CampaignCharacterCreationProfile,
) campaigns.CampaignCharacterCreation {
	selectedDomainCardIDs := make([]string, 0, len(profile.DomainCardIDs))
	for _, domainCardID := range profile.DomainCardIDs {
		trimmedDomainCardID := strings.TrimSpace(domainCardID)
		if trimmedDomainCardID == "" {
			continue
		}
		selectedDomainCardIDs = append(selectedDomainCardIDs, trimmedDomainCardID)
	}

	creation := campaigns.CampaignCharacterCreation{
		Progress: campaigns.CampaignCharacterCreationProgress{
			Steps:        append([]campaigns.CampaignCharacterCreationStep(nil), progress.Steps...),
			NextStep:     progress.NextStep,
			Ready:        progress.Ready,
			UnmetReasons: append([]string(nil), progress.UnmetReasons...),
		},
		Profile: campaigns.CampaignCharacterCreationProfile{
			ClassID:            strings.TrimSpace(profile.ClassID),
			SubclassID:         strings.TrimSpace(profile.SubclassID),
			AncestryID:         strings.TrimSpace(profile.AncestryID),
			CommunityID:        strings.TrimSpace(profile.CommunityID),
			Agility:            strings.TrimSpace(profile.Agility),
			Strength:           strings.TrimSpace(profile.Strength),
			Finesse:            strings.TrimSpace(profile.Finesse),
			Instinct:           strings.TrimSpace(profile.Instinct),
			Presence:           strings.TrimSpace(profile.Presence),
			Knowledge:          strings.TrimSpace(profile.Knowledge),
			PrimaryWeaponID:    strings.TrimSpace(profile.PrimaryWeaponID),
			SecondaryWeaponID:  strings.TrimSpace(profile.SecondaryWeaponID),
			ArmorID:            strings.TrimSpace(profile.ArmorID),
			PotionItemID:       strings.TrimSpace(profile.PotionItemID),
			Background:         strings.TrimSpace(profile.Background),
			ExperienceName:     strings.TrimSpace(profile.ExperienceName),
			ExperienceModifier: strings.TrimSpace(profile.ExperienceModifier),
			DomainCardIDs:      selectedDomainCardIDs,
			Connections:        strings.TrimSpace(profile.Connections),
		},
		Classes:          []campaigns.CatalogClass{},
		Subclasses:       []campaigns.CatalogSubclass{},
		Ancestries:       []campaigns.CatalogHeritage{},
		Communities:      []campaigns.CatalogHeritage{},
		PrimaryWeapons:   []campaigns.CatalogWeapon{},
		SecondaryWeapons: []campaigns.CatalogWeapon{},
		Armor:            []campaigns.CatalogArmor{},
		PotionItems:      []campaigns.CatalogItem{},
		DomainCards:      []campaigns.CatalogDomainCard{},
	}

	classDomainsByID := make(map[string]map[string]struct{}, len(catalog.Classes))
	for _, class := range catalog.Classes {
		classID := strings.TrimSpace(class.ID)
		if classID == "" {
			continue
		}
		className := strings.TrimSpace(class.Name)
		if className == "" {
			className = classID
		}
		domainIDs := make([]string, 0, len(class.DomainIDs))
		domains := make(map[string]struct{}, len(class.DomainIDs))
		for _, domainID := range class.DomainIDs {
			trimmedDomainID := strings.TrimSpace(domainID)
			if trimmedDomainID == "" {
				continue
			}
			domainIDs = append(domainIDs, trimmedDomainID)
			domains[trimmedDomainID] = struct{}{}
		}
		classDomainsByID[classID] = domains
		creation.Classes = append(creation.Classes, campaigns.CatalogClass{
			ID:        classID,
			Name:      className,
			DomainIDs: domainIDs,
		})
	}
	sortByName(creation.Classes, func(c campaigns.CatalogClass) string { return c.Name }, func(c campaigns.CatalogClass) string { return c.ID })

	selectedClassID := strings.TrimSpace(creation.Profile.ClassID)
	for _, subclass := range catalog.Subclasses {
		subclassID := strings.TrimSpace(subclass.ID)
		if subclassID == "" {
			continue
		}
		subclassClassID := strings.TrimSpace(subclass.ClassID)
		if selectedClassID != "" && subclassClassID != selectedClassID {
			continue
		}
		subclassName := strings.TrimSpace(subclass.Name)
		if subclassName == "" {
			subclassName = subclassID
		}
		creation.Subclasses = append(creation.Subclasses, campaigns.CatalogSubclass{
			ID:      subclassID,
			Name:    subclassName,
			ClassID: subclassClassID,
		})
	}
	sortByName(creation.Subclasses, func(s campaigns.CatalogSubclass) string { return s.Name }, func(s campaigns.CatalogSubclass) string { return s.ID })

	for _, heritage := range catalog.Heritages {
		heritageID := strings.TrimSpace(heritage.ID)
		if heritageID == "" {
			continue
		}
		heritageName := strings.TrimSpace(heritage.Name)
		if heritageName == "" {
			heritageName = heritageID
		}
		entry := campaigns.CatalogHeritage{
			ID:   heritageID,
			Name: heritageName,
			Kind: strings.TrimSpace(heritage.Kind),
		}
		switch strings.ToLower(strings.TrimSpace(heritage.Kind)) {
		case "ancestry":
			creation.Ancestries = append(creation.Ancestries, entry)
		case "community":
			creation.Communities = append(creation.Communities, entry)
		}
	}
	sortByName(creation.Ancestries, func(h campaigns.CatalogHeritage) string { return h.Name }, func(h campaigns.CatalogHeritage) string { return h.ID })
	sortByName(creation.Communities, func(h campaigns.CatalogHeritage) string { return h.Name }, func(h campaigns.CatalogHeritage) string { return h.ID })

	for _, weapon := range catalog.Weapons {
		weaponID := strings.TrimSpace(weapon.ID)
		if weaponID == "" || weapon.Tier != 1 {
			continue
		}
		weaponName := strings.TrimSpace(weapon.Name)
		if weaponName == "" {
			weaponName = weaponID
		}
		entry := campaigns.CatalogWeapon{
			ID:       weaponID,
			Name:     weaponName,
			Category: strings.TrimSpace(weapon.Category),
			Tier:     weapon.Tier,
		}
		switch strings.ToLower(strings.TrimSpace(weapon.Category)) {
		case "primary":
			creation.PrimaryWeapons = append(creation.PrimaryWeapons, entry)
		case "secondary":
			creation.SecondaryWeapons = append(creation.SecondaryWeapons, entry)
		}
	}
	sortByName(creation.PrimaryWeapons, func(w campaigns.CatalogWeapon) string { return w.Name }, func(w campaigns.CatalogWeapon) string { return w.ID })
	sortByName(creation.SecondaryWeapons, func(w campaigns.CatalogWeapon) string { return w.Name }, func(w campaigns.CatalogWeapon) string { return w.ID })

	for _, armor := range catalog.Armor {
		armorID := strings.TrimSpace(armor.ID)
		if armorID == "" || armor.Tier != 1 {
			continue
		}
		armorName := strings.TrimSpace(armor.Name)
		if armorName == "" {
			armorName = armorID
		}
		creation.Armor = append(creation.Armor, campaigns.CatalogArmor{
			ID:   armorID,
			Name: armorName,
			Tier: armor.Tier,
		})
	}
	sortByName(creation.Armor, func(a campaigns.CatalogArmor) string { return a.Name }, func(a campaigns.CatalogArmor) string { return a.ID })

	for _, item := range catalog.Items {
		itemID := strings.TrimSpace(item.ID)
		if itemID == "" {
			continue
		}
		if !isAllowedPotionItemID(itemID) {
			continue
		}
		itemName := strings.TrimSpace(item.Name)
		if itemName == "" {
			itemName = itemID
		}
		creation.PotionItems = append(creation.PotionItems, campaigns.CatalogItem{ID: itemID, Name: itemName})
	}
	sortByName(creation.PotionItems, func(i campaigns.CatalogItem) string { return i.Name }, func(i campaigns.CatalogItem) string { return i.ID })

	allowedDomains := classDomainsByID[selectedClassID]
	for _, domainCard := range catalog.DomainCards {
		domainCardID := strings.TrimSpace(domainCard.ID)
		if domainCardID == "" {
			continue
		}
		domainID := strings.TrimSpace(domainCard.DomainID)
		if selectedClassID != "" && len(allowedDomains) > 0 {
			if _, ok := allowedDomains[domainID]; !ok {
				continue
			}
		}
		domainCardName := strings.TrimSpace(domainCard.Name)
		if domainCardName == "" {
			domainCardName = domainCardID
		}
		creation.DomainCards = append(creation.DomainCards, campaigns.CatalogDomainCard{
			ID:       domainCardID,
			Name:     domainCardName,
			DomainID: domainID,
			Level:    domainCard.Level,
		})
	}
	sort.SliceStable(creation.DomainCards, func(i, j int) bool {
		leftLevel := creation.DomainCards[i].Level
		rightLevel := creation.DomainCards[j].Level
		if leftLevel == rightLevel {
			leftName := strings.ToLower(strings.TrimSpace(creation.DomainCards[i].Name))
			rightName := strings.ToLower(strings.TrimSpace(creation.DomainCards[j].Name))
			if leftName == rightName {
				return strings.TrimSpace(creation.DomainCards[i].ID) < strings.TrimSpace(creation.DomainCards[j].ID)
			}
			return leftName < rightName
		}
		return leftLevel < rightLevel
	})

	return creation
}

func sortByName[T any](items []T, nameOf func(T) string, idOf func(T) string) {
	sort.SliceStable(items, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(nameOf(items[i])))
		right := strings.ToLower(strings.TrimSpace(nameOf(items[j])))
		if left == right {
			return strings.TrimSpace(idOf(items[i])) < strings.TrimSpace(idOf(items[j]))
		}
		return left < right
	})
}
