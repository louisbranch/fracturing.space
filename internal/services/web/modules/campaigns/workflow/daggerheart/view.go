package daggerheart

import (
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// CreationView maps the domain creation model to the template view type.
func (Workflow) CreationView(creation campaigns.CampaignCharacterCreation) webtemplates.CampaignCharacterCreationView {
	view := webtemplates.CampaignCharacterCreationView{
		Ready:              creation.Progress.Ready,
		NextStep:           creation.Progress.NextStep,
		UnmetReasons:       append([]string(nil), creation.Progress.UnmetReasons...),
		ClassID:            creation.Profile.ClassID,
		SubclassID:         creation.Profile.SubclassID,
		AncestryID:         creation.Profile.AncestryID,
		CommunityID:        creation.Profile.CommunityID,
		Agility:            creation.Profile.Agility,
		Strength:           creation.Profile.Strength,
		Finesse:            creation.Profile.Finesse,
		Instinct:           creation.Profile.Instinct,
		Presence:           creation.Profile.Presence,
		Knowledge:          creation.Profile.Knowledge,
		PrimaryWeaponID:    creation.Profile.PrimaryWeaponID,
		SecondaryWeaponID:  creation.Profile.SecondaryWeaponID,
		ArmorID:            creation.Profile.ArmorID,
		PotionItemID:       creation.Profile.PotionItemID,
		Background:         creation.Profile.Background,
		ExperienceName:     creation.Profile.ExperienceName,
		ExperienceModifier: creation.Profile.ExperienceModifier,
		DomainCardIDs:      append([]string(nil), creation.Profile.DomainCardIDs...),
		Connections:        creation.Profile.Connections,
		Steps:              make([]webtemplates.CampaignCharacterCreationStepView, 0, len(creation.Progress.Steps)),
		Classes:            make([]webtemplates.CampaignCreationClassView, 0, len(creation.Classes)),
		Subclasses:         make([]webtemplates.CampaignCreationSubclassView, 0, len(creation.Subclasses)),
		Ancestries:         make([]webtemplates.CampaignCreationHeritageView, 0, len(creation.Ancestries)),
		Communities:        make([]webtemplates.CampaignCreationHeritageView, 0, len(creation.Communities)),
		PrimaryWeapons:     make([]webtemplates.CampaignCreationWeaponView, 0, len(creation.PrimaryWeapons)),
		SecondaryWeapons:   make([]webtemplates.CampaignCreationWeaponView, 0, len(creation.SecondaryWeapons)),
		Armor:              make([]webtemplates.CampaignCreationArmorView, 0, len(creation.Armor)),
		PotionItems:        make([]webtemplates.CampaignCreationItemView, 0, len(creation.PotionItems)),
		DomainCards:        make([]webtemplates.CampaignCreationDomainCardView, 0, len(creation.DomainCards)),
	}
	for _, step := range creation.Progress.Steps {
		view.Steps = append(view.Steps, webtemplates.CampaignCharacterCreationStepView{
			Step:     step.Step,
			Key:      step.Key,
			Complete: step.Complete,
		})
	}
	for _, class := range creation.Classes {
		view.Classes = append(view.Classes, webtemplates.CampaignCreationClassView{
			ID:   class.ID,
			Name: class.Name,
		})
	}
	for _, subclass := range creation.Subclasses {
		view.Subclasses = append(view.Subclasses, webtemplates.CampaignCreationSubclassView{
			ID:      subclass.ID,
			Name:    subclass.Name,
			ClassID: subclass.ClassID,
		})
	}
	for _, ancestry := range creation.Ancestries {
		view.Ancestries = append(view.Ancestries, webtemplates.CampaignCreationHeritageView{
			ID:   ancestry.ID,
			Name: ancestry.Name,
		})
	}
	for _, community := range creation.Communities {
		view.Communities = append(view.Communities, webtemplates.CampaignCreationHeritageView{
			ID:   community.ID,
			Name: community.Name,
		})
	}
	for _, weapon := range creation.PrimaryWeapons {
		view.PrimaryWeapons = append(view.PrimaryWeapons, webtemplates.CampaignCreationWeaponView{
			ID:   weapon.ID,
			Name: weapon.Name,
		})
	}
	for _, weapon := range creation.SecondaryWeapons {
		view.SecondaryWeapons = append(view.SecondaryWeapons, webtemplates.CampaignCreationWeaponView{
			ID:   weapon.ID,
			Name: weapon.Name,
		})
	}
	for _, armor := range creation.Armor {
		view.Armor = append(view.Armor, webtemplates.CampaignCreationArmorView{
			ID:   armor.ID,
			Name: armor.Name,
		})
	}
	for _, item := range creation.PotionItems {
		view.PotionItems = append(view.PotionItems, webtemplates.CampaignCreationItemView{
			ID:   item.ID,
			Name: item.Name,
		})
	}
	for _, card := range creation.DomainCards {
		view.DomainCards = append(view.DomainCards, webtemplates.CampaignCreationDomainCardView{
			ID:       card.ID,
			Name:     card.Name,
			DomainID: card.DomainID,
			Level:    card.Level,
		})
	}
	return view
}
