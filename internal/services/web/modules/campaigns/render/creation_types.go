package render

// CampaignCharacterCreationStepView carries one step status row for the character-creation workflow.
type CampaignCharacterCreationStepView struct {
	Step     int32
	Key      string
	Complete bool
}

// CampaignCreationClassFeatureView carries one feature paragraph for class, heritage, and subclass cards.
type CampaignCreationClassFeatureView struct {
	Name        string
	Description string
}

// CampaignCreationDomainWatermarkView carries class-domain icon metadata for selectable cards.
type CampaignCreationDomainWatermarkView struct {
	ID      string
	Name    string
	IconURL string
}

// CampaignCreationClassView carries one class option card.
type CampaignCreationClassView struct {
	ID               string
	Name             string
	ImageURL         string
	StartingHP       int32
	StartingEvasion  int32
	HopeFeature      CampaignCreationClassFeatureView
	Features         []CampaignCreationClassFeatureView
	DomainNames      []string
	DomainWatermarks []CampaignCreationDomainWatermarkView
}

// CampaignCreationSubclassView carries one subclass option card.
type CampaignCreationSubclassView struct {
	ID                   string
	Name                 string
	ImageURL             string
	ClassID              string
	SpellcastTrait       string
	CreationRequirements []string
	Foundation           []CampaignCreationClassFeatureView
}

// CampaignCreationHeritageView carries one ancestry or community option card.
type CampaignCreationHeritageView struct {
	ID       string
	Name     string
	ImageURL string
	Features []CampaignCreationClassFeatureView
}

// CampaignCreationWeaponView carries one weapon choice.
type CampaignCreationWeaponView struct {
	ID           string
	Name         string
	ImageURL     string
	Burden       int32
	Trait        string
	Range        string
	Damage       string
	Feature      string
	DisplayGroup string
}

// CampaignCreationWeaponGroupView carries one ordered render-time weapon group.
type CampaignCreationWeaponGroupView struct {
	Key     string
	Weapons []CampaignCreationWeaponView
}

// CampaignCreationArmorView carries one armor choice.
type CampaignCreationArmorView struct {
	ID             string
	Name           string
	ImageURL       string
	ArmorScore     int32
	BaseThresholds string
	Feature        string
}

// CampaignCreationItemView carries one item choice.
type CampaignCreationItemView struct {
	ID          string
	Name        string
	ImageURL    string
	Description string
}

// CampaignCreationExperienceView carries one freeform experience row.
type CampaignCreationExperienceView struct {
	ID       string
	Name     string
	Modifier string
}

// CampaignCreationTraitOptionView carries one enriched trait card for the creation form.
type CampaignCreationTraitOptionView struct {
	FieldName    string // form POST name: "agility"
	LabelKey     string // i18n key: "game.character_creation.field.agility"
	Abbreviation string // "AGI"
	SkillsKey    string // i18n key: "game.character_creation.trait_skills.agility"
	Current      string // "-1", "0", "1", "2"
}

// CampaignCreationCompanionExperienceOptionView carries one companion experience catalog option.
type CampaignCreationCompanionExperienceOptionView struct {
	ID          string
	Name        string
	Description string
}

// CampaignCreationHeritageSelectionView carries structured heritage state for creation rendering.
type CampaignCreationHeritageSelectionView struct {
	AncestryLabel           string
	FirstFeatureAncestryID  string
	FirstFeatureID          string
	SecondFeatureAncestryID string
	SecondFeatureID         string
	CommunityID             string
}

// CampaignCreationCompanionView carries the creation/detail companion sheet.
type CampaignCreationCompanionView struct {
	AnimalKind        string
	Name              string
	Evasion           int32
	Experiences       []CampaignCreationExperienceView
	AttackDescription string
	AttackRange       string
	DamageDieSides    int32
	DamageType        string
}

// CampaignCreationDomainCardView carries one domain-card choice.
type CampaignCreationDomainCardView struct {
	ID          string
	Name        string
	ImageURL    string
	DomainID    string
	DomainName  string
	Level       int32
	Type        string
	RecallCost  int32
	FeatureText string
}

// CampaignCharacterCreationView carries the full transport/render contract for one character-creation workflow state.
type CampaignCharacterCreationView struct {
	Ready                        bool
	NextStep                     int32
	UnmetReasons                 []string
	ClassID                      string
	SubclassID                   string
	SubclassCreationRequirements []string
	Heritage                     CampaignCreationHeritageSelectionView
	CompanionSheet               *CampaignCreationCompanionView
	CompanionExperiences         []CampaignCreationCompanionExperienceOptionView
	Agility                      string
	Strength                     string
	Finesse                      string
	Instinct                     string
	Presence                     string
	Knowledge                    string
	TraitOptions                 []CampaignCreationTraitOptionView
	PrimaryWeaponID              string
	SecondaryWeaponID            string
	ArmorID                      string
	PotionItemID                 string
	Background                   string
	Description                  string
	Experiences                  []CampaignCreationExperienceView
	DomainCardIDs                []string
	Connections                  string
	NextStepPrefetchURLs         []string
	Steps                        []CampaignCharacterCreationStepView
	Classes                      []CampaignCreationClassView
	Subclasses                   []CampaignCreationSubclassView
	Ancestries                   []CampaignCreationHeritageView
	Communities                  []CampaignCreationHeritageView
	PrimaryWeapons               []CampaignCreationWeaponView
	SecondaryWeapons             []CampaignCreationWeaponView
	PrimaryWeaponGroups          []CampaignCreationWeaponGroupView
	SecondaryWeaponGroups        []CampaignCreationWeaponGroupView
	SecondaryWeaponNoneImageURL  string
	Armor                        []CampaignCreationArmorView
	PotionItems                  []CampaignCreationItemView
	DomainCards                  []CampaignCreationDomainCardView
}
