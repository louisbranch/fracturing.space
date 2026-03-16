package app

// CampaignCharacterCreationStep stores one workflow step status.
type CampaignCharacterCreationStep struct {
	Step     int32  `json:"step"`
	Key      string `json:"key"`
	Complete bool   `json:"complete"`
}

// CampaignCharacterCreationProgress stores workflow progress metadata.
type CampaignCharacterCreationProgress struct {
	Steps        []CampaignCharacterCreationStep `json:"steps"`
	NextStep     int32                           `json:"nextStep"`
	Ready        bool                            `json:"ready"`
	UnmetReasons []string                        `json:"unmetReasons"`
}

// CatalogFeature stores a named feature with its description text.
type CatalogFeature struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CatalogAssetReference stores one resolved image reference for content entities.
type CatalogAssetReference struct {
	URL     string `json:"url"`
	Status  string `json:"status"`
	SetID   string `json:"setId"`
	AssetID string `json:"assetId"`
}

// CatalogClass stores class catalog data used by workflow forms.
type CatalogClass struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	DomainIDs       []string              `json:"domainIds"`
	StartingHP      int32                 `json:"startingHp"`
	StartingEvasion int32                 `json:"startingEvasion"`
	HopeFeature     CatalogFeature        `json:"hopeFeature"`
	Features        []CatalogFeature      `json:"features"`
	Illustration    CatalogAssetReference `json:"illustration"`
	Icon            CatalogAssetReference `json:"icon"`
}

// CatalogSubclass stores subclass catalog data used by workflow forms.
type CatalogSubclass struct {
	ID                   string                `json:"id"`
	Name                 string                `json:"name"`
	ClassID              string                `json:"classId"`
	SpellcastTrait       string                `json:"spellcastTrait"`
	CreationRequirements []string              `json:"creationRequirements"`
	Foundation           []CatalogFeature      `json:"foundation"`
	Illustration         CatalogAssetReference `json:"illustration"`
}

// CatalogHeritage stores ancestry/community catalog data.
type CatalogHeritage struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Kind         string                `json:"kind"`
	Features     []CatalogFeature      `json:"features"`
	Illustration CatalogAssetReference `json:"illustration"`
}

// CatalogDomain stores domain catalog data used by class/domain selection flows.
type CatalogDomain struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Illustration CatalogAssetReference `json:"illustration"`
	Icon         CatalogAssetReference `json:"icon"`
}

// CatalogWeapon stores weapon catalog data used by equipment forms.
type CatalogWeapon struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Category     string                `json:"category"`
	Tier         int32                 `json:"tier"`
	Burden       int32                 `json:"burden"`
	Trait        string                `json:"trait"`
	Range        string                `json:"range"`
	Damage       string                `json:"damage"`
	Feature      string                `json:"feature"`
	Illustration CatalogAssetReference `json:"illustration"`
}

// CatalogArmor stores armor catalog data used by equipment forms.
type CatalogArmor struct {
	ID             string                `json:"id"`
	Name           string                `json:"name"`
	Tier           int32                 `json:"tier"`
	ArmorScore     int32                 `json:"armorScore"`
	BaseThresholds string                `json:"baseThresholds"`
	Feature        string                `json:"feature"`
	Illustration   CatalogAssetReference `json:"illustration"`
}

// CatalogItem stores item catalog data used by equipment forms.
type CatalogItem struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Description  string                `json:"description"`
	Illustration CatalogAssetReference `json:"illustration"`
}

// CatalogDomainCard stores domain card catalog data used by forms.
type CatalogDomainCard struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	DomainID     string                `json:"domainId"`
	DomainName   string                `json:"domainName"`
	Level        int32                 `json:"level"`
	Type         string                `json:"type"`
	RecallCost   int32                 `json:"recallCost"`
	FeatureText  string                `json:"featureText"`
	Illustration CatalogAssetReference `json:"illustration"`
}

// CatalogAdversary stores adversary catalog data with image metadata.
type CatalogAdversary struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Illustration CatalogAssetReference `json:"illustration"`
}

// CatalogEnvironment stores environment catalog data with image metadata.
type CatalogEnvironment struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Illustration CatalogAssetReference `json:"illustration"`
}

// CampaignCharacterCreationCatalog stores Daggerheart catalog subsets used by workflow forms.
type CampaignCharacterCreationCatalog struct {
	AssetTheme           string                       `json:"assetTheme"`
	Classes              []CatalogClass               `json:"classes"`
	Subclasses           []CatalogSubclass            `json:"subclasses"`
	Heritages            []CatalogHeritage            `json:"heritages"`
	CompanionExperiences []CatalogCompanionExperience `json:"companionExperiences"`
	Domains              []CatalogDomain              `json:"domains"`
	Weapons              []CatalogWeapon              `json:"weapons"`
	Armor                []CatalogArmor               `json:"armor"`
	Items                []CatalogItem                `json:"items"`
	DomainCards          []CatalogDomainCard          `json:"domainCards"`
	Adversaries          []CatalogAdversary           `json:"adversaries"`
	Environments         []CatalogEnvironment         `json:"environments"`
}

// CatalogCompanionExperience stores companion experience catalog data used by workflow forms.
type CatalogCompanionExperience struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CampaignCharacterCreationExperience stores one experience name+modifier pair.
type CampaignCharacterCreationExperience struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Modifier string `json:"modifier"`
}

// CampaignCharacterCreationHeritageSelection stores the structured heritage state.
type CampaignCharacterCreationHeritageSelection struct {
	AncestryLabel           string `json:"ancestryLabel"`
	FirstFeatureAncestryID  string `json:"firstFeatureAncestryId"`
	FirstFeatureID          string `json:"firstFeatureId"`
	SecondFeatureAncestryID string `json:"secondFeatureAncestryId"`
	SecondFeatureID         string `json:"secondFeatureId"`
	CommunityID             string `json:"communityId"`
}

// CampaignCharacterCreationCompanionSheet stores the companion sheet returned from profile reads.
type CampaignCharacterCreationCompanionSheet struct {
	AnimalKind        string                                `json:"animalKind"`
	Name              string                                `json:"name"`
	Evasion           int32                                 `json:"evasion"`
	Experiences       []CampaignCharacterCreationExperience `json:"experiences"`
	AttackDescription string                                `json:"attackDescription"`
	AttackRange       string                                `json:"attackRange"`
	DamageDieSides    int32                                 `json:"damageDieSides"`
	DamageType        string                                `json:"damageType"`
}

// CampaignCharacterCreationCompanionInput stores the editable companion input for creation step 1.
type CampaignCharacterCreationCompanionInput struct {
	AnimalKind        string   `json:"animalKind"`
	Name              string   `json:"name"`
	ExperienceIDs     []string `json:"experienceIds"`
	AttackDescription string   `json:"attackDescription"`
	DamageType        string   `json:"damageType"`
}

// CampaignCharacterCreationProfile stores selected workflow fields used for filtering options.
type CampaignCharacterCreationProfile struct {
	CharacterName                string                                     `json:"characterName"`
	ClassID                      string                                     `json:"classId"`
	SubclassID                   string                                     `json:"subclassId"`
	SubclassCreationRequirements []string                                   `json:"subclassCreationRequirements"`
	Heritage                     CampaignCharacterCreationHeritageSelection `json:"heritage"`
	CompanionSheet               *CampaignCharacterCreationCompanionSheet   `json:"companionSheet,omitempty"`
	Agility                      string                                     `json:"agility"`
	Strength                     string                                     `json:"strength"`
	Finesse                      string                                     `json:"finesse"`
	Instinct                     string                                     `json:"instinct"`
	Presence                     string                                     `json:"presence"`
	Knowledge                    string                                     `json:"knowledge"`
	PrimaryWeaponID              string                                     `json:"primaryWeaponId"`
	SecondaryWeaponID            string                                     `json:"secondaryWeaponId"`
	ArmorID                      string                                     `json:"armorId"`
	PotionItemID                 string                                     `json:"potionItemId"`
	Background                   string                                     `json:"background"`
	Description                  string                                     `json:"description"`
	Experiences                  []CampaignCharacterCreationExperience      `json:"experiences"`
	DomainCardIDs                []string                                   `json:"domainCardIds"`
	Connections                  string                                     `json:"connections"`
}

// CampaignCharacterCreationStepInput stores one character creation step in domain form.
type CampaignCharacterCreationStepInput struct {
	ClassSubclass *CampaignCharacterCreationStepClassSubclass `json:"classSubclass,omitempty"`
	Heritage      *CampaignCharacterCreationStepHeritage      `json:"heritage,omitempty"`
	Traits        *CampaignCharacterCreationStepTraits        `json:"traits,omitempty"`
	Details       *CampaignCharacterCreationStepDetails       `json:"details,omitempty"`
	Equipment     *CampaignCharacterCreationStepEquipment     `json:"equipment,omitempty"`
	Background    *CampaignCharacterCreationStepBackground    `json:"background,omitempty"`
	Experiences   *CampaignCharacterCreationStepExperiences   `json:"experiences,omitempty"`
	DomainCards   *CampaignCharacterCreationStepDomainCards   `json:"domainCards,omitempty"`
	Connections   *CampaignCharacterCreationStepConnections   `json:"connections,omitempty"`
}

// CampaignCharacterCreationStepClassSubclass stores class/subclass step input.
type CampaignCharacterCreationStepClassSubclass struct {
	ClassID    string                                   `json:"classId"`
	SubclassID string                                   `json:"subclassId"`
	Companion  *CampaignCharacterCreationCompanionInput `json:"companion,omitempty"`
}

// CampaignCharacterCreationStepHeritage stores structured heritage step input.
type CampaignCharacterCreationStepHeritage struct {
	Heritage CampaignCharacterCreationHeritageSelection `json:"heritage"`
}

// CampaignCharacterCreationStepTraits stores trait allocation step input.
type CampaignCharacterCreationStepTraits struct {
	Agility   int32 `json:"agility"`
	Strength  int32 `json:"strength"`
	Finesse   int32 `json:"finesse"`
	Instinct  int32 `json:"instinct"`
	Presence  int32 `json:"presence"`
	Knowledge int32 `json:"knowledge"`
}

// CampaignCharacterCreationStepDetails stores details step input.
type CampaignCharacterCreationStepDetails struct {
	Description string `json:"description"`
}

// CampaignCharacterCreationStepEquipment stores equipment step input.
type CampaignCharacterCreationStepEquipment struct {
	WeaponIDs    []string `json:"weaponIds"`
	ArmorID      string   `json:"armorId"`
	PotionItemID string   `json:"potionItemId"`
}

// CampaignCharacterCreationStepBackground stores background step input.
type CampaignCharacterCreationStepBackground struct {
	Background string `json:"background"`
}

// CampaignCharacterCreationStepExperience stores a single background experience entry.
type CampaignCharacterCreationStepExperience struct {
	Name     string `json:"name"`
	Modifier int32  `json:"modifier"`
}

// CampaignCharacterCreationStepExperiences stores experience step input.
type CampaignCharacterCreationStepExperiences struct {
	Experiences []CampaignCharacterCreationStepExperience `json:"experiences"`
}

// CampaignCharacterCreationStepDomainCards stores selected domain cards step input.
type CampaignCharacterCreationStepDomainCards struct {
	DomainCardIDs []string `json:"domainCardIds"`
}

// CampaignCharacterCreationStepConnections stores player connections step input.
type CampaignCharacterCreationStepConnections struct {
	Connections string `json:"connections"`
}
