package daggerheart

import (
	"sort"
	"strings"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
)

// AssembleCatalog builds the Daggerheart-specific catalog view from generic
// gateway data (progress, catalog, profile).
func (Workflow) AssembleCatalog(
	progress campaignapp.CampaignCharacterCreationProgress,
	catalog campaignapp.CampaignCharacterCreationCatalog,
	profile campaignapp.CampaignCharacterCreationProfile,
) campaignapp.CampaignCharacterCreation {
	selectedDomainCardIDs := make([]string, 0, len(profile.DomainCardIDs))
	for _, domainCardID := range profile.DomainCardIDs {
		trimmedDomainCardID := strings.TrimSpace(domainCardID)
		if trimmedDomainCardID == "" {
			continue
		}
		selectedDomainCardIDs = append(selectedDomainCardIDs, trimmedDomainCardID)
	}

	creation := campaignapp.CampaignCharacterCreation{
		Progress: campaignapp.CampaignCharacterCreationProgress{
			Steps:        append([]campaignapp.CampaignCharacterCreationStep(nil), progress.Steps...),
			NextStep:     progress.NextStep,
			Ready:        progress.Ready,
			UnmetReasons: append([]string(nil), progress.UnmetReasons...),
		},
		Profile: campaignapp.CampaignCharacterCreationProfile{
			CharacterName:     strings.TrimSpace(profile.CharacterName),
			ClassID:           strings.TrimSpace(profile.ClassID),
			SubclassID:        strings.TrimSpace(profile.SubclassID),
			AncestryID:        strings.TrimSpace(profile.AncestryID),
			CommunityID:       strings.TrimSpace(profile.CommunityID),
			Agility:           strings.TrimSpace(profile.Agility),
			Strength:          strings.TrimSpace(profile.Strength),
			Finesse:           strings.TrimSpace(profile.Finesse),
			Instinct:          strings.TrimSpace(profile.Instinct),
			Presence:          strings.TrimSpace(profile.Presence),
			Knowledge:         strings.TrimSpace(profile.Knowledge),
			PrimaryWeaponID:   strings.TrimSpace(profile.PrimaryWeaponID),
			SecondaryWeaponID: strings.TrimSpace(profile.SecondaryWeaponID),
			ArmorID:           strings.TrimSpace(profile.ArmorID),
			PotionItemID:      strings.TrimSpace(profile.PotionItemID),
			Background:        strings.TrimSpace(profile.Background),
			Description:       strings.TrimSpace(profile.Description),
			Experiences:       trimExperiences(profile.Experiences),
			DomainCardIDs:     selectedDomainCardIDs,
			Connections:       strings.TrimSpace(profile.Connections),
		},
		Classes:          []campaignapp.CatalogClass{},
		Subclasses:       []campaignapp.CatalogSubclass{},
		Ancestries:       []campaignapp.CatalogHeritage{},
		Communities:      []campaignapp.CatalogHeritage{},
		PrimaryWeapons:   []campaignapp.CatalogWeapon{},
		SecondaryWeapons: []campaignapp.CatalogWeapon{},
		Armor:            []campaignapp.CatalogArmor{},
		PotionItems:      []campaignapp.CatalogItem{},
		DomainCards:      []campaignapp.CatalogDomainCard{},
		Domains:          []campaignapp.CatalogDomain{},
	}

	// Build domain name lookup from catalog domains.
	domainNameByID := make(map[string]string, len(catalog.Domains))
	for _, domain := range catalog.Domains {
		domainID := strings.TrimSpace(domain.ID)
		if domainID == "" {
			continue
		}
		domainName := strings.TrimSpace(domain.Name)
		if domainName == "" {
			domainName = domainID
		}
		domainNameByID[domainID] = domainName
		creation.Domains = append(creation.Domains, campaignapp.CatalogDomain{
			ID:           domainID,
			Name:         domainName,
			Illustration: domain.Illustration,
			Icon:         domain.Icon,
		})
	}
	sortByName(creation.Domains, func(d campaignapp.CatalogDomain) string { return d.Name }, func(d campaignapp.CatalogDomain) string { return d.ID })

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
		hopeFeature := campaignapp.CatalogFeature{
			Name:        strings.TrimSpace(class.HopeFeature.Name),
			Description: strings.TrimSpace(class.HopeFeature.Description),
		}
		features := make([]campaignapp.CatalogFeature, len(class.Features))
		copy(features, class.Features)
		creation.Classes = append(creation.Classes, campaignapp.CatalogClass{
			ID:              classID,
			Name:            className,
			DomainIDs:       domainIDs,
			StartingHP:      class.StartingHP,
			StartingEvasion: class.StartingEvasion,
			HopeFeature:     hopeFeature,
			Features:        features,
			Illustration:    class.Illustration,
			Icon:            class.Icon,
		})
	}
	sortByName(creation.Classes, func(c campaignapp.CatalogClass) string { return c.Name }, func(c campaignapp.CatalogClass) string { return c.ID })

	selectedClassID := strings.TrimSpace(creation.Profile.ClassID)
	for _, subclass := range catalog.Subclasses {
		subclassID := strings.TrimSpace(subclass.ID)
		if subclassID == "" {
			continue
		}
		subclassClassID := strings.TrimSpace(subclass.ClassID)
		subclassName := strings.TrimSpace(subclass.Name)
		if subclassName == "" {
			subclassName = subclassID
		}
		foundation := make([]campaignapp.CatalogFeature, len(subclass.Foundation))
		copy(foundation, subclass.Foundation)
		creation.Subclasses = append(creation.Subclasses, campaignapp.CatalogSubclass{
			ID:             subclassID,
			Name:           subclassName,
			ClassID:        subclassClassID,
			SpellcastTrait: strings.TrimSpace(subclass.SpellcastTrait),
			Foundation:     foundation,
			Illustration:   subclass.Illustration,
		})
	}
	sortByName(creation.Subclasses, func(s campaignapp.CatalogSubclass) string { return s.Name }, func(s campaignapp.CatalogSubclass) string { return s.ID })

	for _, heritage := range catalog.Heritages {
		heritageID := strings.TrimSpace(heritage.ID)
		if heritageID == "" {
			continue
		}
		heritageName := strings.TrimSpace(heritage.Name)
		if heritageName == "" {
			heritageName = heritageID
		}
		features := make([]campaignapp.CatalogFeature, len(heritage.Features))
		copy(features, heritage.Features)
		entry := campaignapp.CatalogHeritage{
			ID:           heritageID,
			Name:         heritageName,
			Kind:         strings.TrimSpace(heritage.Kind),
			Features:     features,
			Illustration: heritage.Illustration,
		}
		switch strings.ToLower(strings.TrimSpace(heritage.Kind)) {
		case "ancestry":
			creation.Ancestries = append(creation.Ancestries, entry)
		case "community":
			creation.Communities = append(creation.Communities, entry)
		}
	}
	sortByName(creation.Ancestries, func(h campaignapp.CatalogHeritage) string { return h.Name }, func(h campaignapp.CatalogHeritage) string { return h.ID })
	sortByName(creation.Communities, func(h campaignapp.CatalogHeritage) string { return h.Name }, func(h campaignapp.CatalogHeritage) string { return h.ID })

	for _, weapon := range catalog.Weapons {
		weaponID := strings.TrimSpace(weapon.ID)
		if weaponID == "" || weapon.Tier != 1 {
			continue
		}
		weaponName := strings.TrimSpace(weapon.Name)
		if weaponName == "" {
			weaponName = weaponID
		}
		entry := campaignapp.CatalogWeapon{
			ID:       weaponID,
			Name:     weaponName,
			Category: strings.TrimSpace(weapon.Category),
			Tier:     weapon.Tier,
			Trait:    strings.TrimSpace(weapon.Trait),
			Range:    strings.TrimSpace(weapon.Range),
			Damage:   strings.TrimSpace(weapon.Damage),
			Feature:  strings.TrimSpace(weapon.Feature),
		}
		switch strings.ToLower(strings.TrimSpace(weapon.Category)) {
		case "primary":
			creation.PrimaryWeapons = append(creation.PrimaryWeapons, entry)
		case "secondary":
			creation.SecondaryWeapons = append(creation.SecondaryWeapons, entry)
		}
	}
	sortByName(creation.PrimaryWeapons, func(w campaignapp.CatalogWeapon) string { return w.Name }, func(w campaignapp.CatalogWeapon) string { return w.ID })
	sortByName(creation.SecondaryWeapons, func(w campaignapp.CatalogWeapon) string { return w.Name }, func(w campaignapp.CatalogWeapon) string { return w.ID })

	for _, armor := range catalog.Armor {
		armorID := strings.TrimSpace(armor.ID)
		if armorID == "" || armor.Tier != 1 {
			continue
		}
		armorName := strings.TrimSpace(armor.Name)
		if armorName == "" {
			armorName = armorID
		}
		creation.Armor = append(creation.Armor, campaignapp.CatalogArmor{
			ID:             armorID,
			Name:           armorName,
			Tier:           armor.Tier,
			ArmorScore:     armor.ArmorScore,
			BaseThresholds: strings.TrimSpace(armor.BaseThresholds),
			Feature:        strings.TrimSpace(armor.Feature),
		})
	}
	sortByName(creation.Armor, func(a campaignapp.CatalogArmor) string { return a.Name }, func(a campaignapp.CatalogArmor) string { return a.ID })

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
		creation.PotionItems = append(creation.PotionItems, campaignapp.CatalogItem{
			ID:          itemID,
			Name:        itemName,
			Description: strings.TrimSpace(item.Description),
		})
	}
	sortByName(creation.PotionItems, func(i campaignapp.CatalogItem) string { return i.Name }, func(i campaignapp.CatalogItem) string { return i.ID })

	allowedDomains := classDomainsByID[selectedClassID]
	for _, domainCard := range catalog.DomainCards {
		domainCardID := strings.TrimSpace(domainCard.ID)
		if domainCardID == "" {
			continue
		}
		// SRD: PCs acquire two 1st-level domain cards at character creation.
		if domainCard.Level != 1 {
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
		creation.DomainCards = append(creation.DomainCards, campaignapp.CatalogDomainCard{
			ID:           domainCardID,
			Name:         domainCardName,
			DomainID:     domainID,
			DomainName:   domainNameByID[domainID],
			Level:        domainCard.Level,
			Type:         strings.TrimSpace(domainCard.Type),
			RecallCost:   domainCard.RecallCost,
			FeatureText:  strings.TrimSpace(domainCard.FeatureText),
			Illustration: domainCard.Illustration,
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

// trimExperiences normalizes the experience slice from the profile.
func trimExperiences(exps []campaignapp.CampaignCharacterCreationExperience) []campaignapp.CampaignCharacterCreationExperience {
	result := make([]campaignapp.CampaignCharacterCreationExperience, 0, len(exps))
	for _, exp := range exps {
		name := strings.TrimSpace(exp.Name)
		if name == "" {
			continue
		}
		result = append(result, campaignapp.CampaignCharacterCreationExperience{
			Name:     name,
			Modifier: strings.TrimSpace(exp.Modifier),
		})
	}
	return result
}

// sortByName centralizes this web behavior in one helper seam.
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
