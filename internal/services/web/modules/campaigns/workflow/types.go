package workflow

import campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"

// StepInput keeps workflow mutation parsing aligned with the campaigns app mutation contract.
type StepInput = campaignapp.CampaignCharacterCreationStepInput

// Step stores one workflow step status.
type Step struct {
	Step     int32
	Key      string
	Complete bool
}

// Progress stores workflow progress metadata for workflow-local page assembly.
type Progress struct {
	Steps        []Step
	NextStep     int32
	Ready        bool
	UnmetReasons []string
}

// Feature stores one named rules text block used by workflow forms.
type Feature struct {
	Name        string
	Description string
}

// AssetReference stores one resolved image reference for workflow catalog entities.
type AssetReference struct {
	URL     string
	Status  string
	SetID   string
	AssetID string
}

// Class stores class catalog data used by workflow forms.
type Class struct {
	ID              string
	Name            string
	DomainIDs       []string
	StartingHP      int32
	StartingEvasion int32
	HopeFeature     Feature
	Features        []Feature
	Illustration    AssetReference
	Icon            AssetReference
}

// Subclass stores subclass catalog data used by workflow forms.
type Subclass struct {
	ID             string
	Name           string
	ClassID        string
	SpellcastTrait string
	Foundation     []Feature
	Illustration   AssetReference
}

// Heritage stores ancestry/community catalog data.
type Heritage struct {
	ID           string
	Name         string
	Kind         string
	Features     []Feature
	Illustration AssetReference
}

// Domain stores domain catalog data used by class/domain selection flows.
type Domain struct {
	ID           string
	Name         string
	Illustration AssetReference
	Icon         AssetReference
}

// Weapon stores weapon catalog data used by equipment forms.
type Weapon struct {
	ID           string
	Name         string
	Category     string
	Tier         int32
	Burden       int32
	Trait        string
	Range        string
	Damage       string
	Feature      string
	Illustration AssetReference
}

// Armor stores armor catalog data used by equipment forms.
type Armor struct {
	ID             string
	Name           string
	Tier           int32
	ArmorScore     int32
	BaseThresholds string
	Feature        string
	Illustration   AssetReference
}

// Item stores item catalog data used by equipment forms.
type Item struct {
	ID           string
	Name         string
	Description  string
	Illustration AssetReference
}

// DomainCard stores domain-card catalog data used by forms.
type DomainCard struct {
	ID           string
	Name         string
	DomainID     string
	DomainName   string
	Level        int32
	Type         string
	RecallCost   int32
	FeatureText  string
	Illustration AssetReference
}

// Adversary stores adversary catalog data with image metadata.
type Adversary struct {
	ID           string
	Name         string
	Illustration AssetReference
}

// Environment stores environment catalog data with image metadata.
type Environment struct {
	ID           string
	Name         string
	Illustration AssetReference
}

// Catalog stores the workflow catalog subsets used by system-specific forms.
type Catalog struct {
	AssetTheme   string
	Classes      []Class
	Subclasses   []Subclass
	Heritages    []Heritage
	Domains      []Domain
	Weapons      []Weapon
	Armor        []Armor
	Items        []Item
	DomainCards  []DomainCard
	Adversaries  []Adversary
	Environments []Environment
}

// Experience stores one experience name+modifier pair.
type Experience struct {
	Name     string
	Modifier string
}

// Profile stores selected workflow fields used for filtering options.
type Profile struct {
	CharacterName     string
	ClassID           string
	SubclassID        string
	AncestryID        string
	CommunityID       string
	Agility           string
	Strength          string
	Finesse           string
	Instinct          string
	Presence          string
	Knowledge         string
	PrimaryWeaponID   string
	SecondaryWeaponID string
	ArmorID           string
	PotionItemID      string
	Background        string
	Description       string
	Experiences       []Experience
	DomainCardIDs     []string
	Connections       string
}
