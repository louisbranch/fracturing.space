package app

import "golang.org/x/text/language"

// GameSystem represents a campaign game system as a domain-native value.
// Gateway implementations map these to proto enums at the transport boundary.
type GameSystem string

const (
	GameSystemUnspecified GameSystem = ""
	GameSystemDaggerheart GameSystem = "daggerheart"
)

// GmMode represents a campaign GM mode as a domain-native value.
type GmMode string

const (
	GmModeUnspecified GmMode = ""
	GmModeHuman       GmMode = "human"
	GmModeAI          GmMode = "ai"
	GmModeHybrid      GmMode = "hybrid"
)

// CharacterKind represents the kind of character as a domain-native value.
type CharacterKind string

const (
	CharacterKindUnspecified CharacterKind = ""
	CharacterKindPC          CharacterKind = "pc"
	CharacterKindNPC         CharacterKind = "npc"
)

// CampaignSummary is a transport-safe summary for campaign listings.
type CampaignSummary struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Theme             string `json:"theme"`
	CoverImageURL     string `json:"coverImageUrl"`
	ParticipantCount  string `json:"participantCount"`
	CharacterCount    string `json:"characterCount"`
	CreatedAtUnixNano int64  `json:"createdAtUnixNano"`
	UpdatedAtUnixNano int64  `json:"updatedAtUnixNano"`
}

// CampaignWorkspace stores campaign details used by campaign workspace routes.
type CampaignWorkspace struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Theme            string `json:"theme"`
	System           string `json:"system"`
	GMMode           string `json:"gmMode"`
	Status           string `json:"status"`
	Locale           string `json:"locale"`
	Intent           string `json:"intent"`
	AccessPolicy     string `json:"accessPolicy"`
	ParticipantCount string `json:"participantCount"`
	CharacterCount   string `json:"characterCount"`
	CoverImageURL    string `json:"coverImageUrl"`
}

// CampaignParticipant stores participant details used by campaign participants pages.
type CampaignParticipant struct {
	ID             string `json:"id"`
	UserID         string `json:"userId"`
	Name           string `json:"name"`
	Role           string `json:"role"`
	CampaignAccess string `json:"campaignAccess"`
	Controller     string `json:"controller"`
	Pronouns       string `json:"pronouns"`
	AvatarURL      string `json:"avatarUrl"`
}

// CampaignCharacter stores character details used by campaign characters pages.
type CampaignCharacter struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Kind           string   `json:"kind"`
	Controller     string   `json:"controller"`
	Pronouns       string   `json:"pronouns"`
	Aliases        []string `json:"aliases"`
	AvatarURL      string   `json:"avatarUrl"`
	CanEdit        bool     `json:"canEdit"`
	EditReasonCode string   `json:"editReasonCode"`
}

// CampaignSession stores session details used by campaign sessions pages.
type CampaignSession struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	StartedAt string `json:"startedAt"`
	UpdatedAt string `json:"updatedAt"`
	EndedAt   string `json:"endedAt"`
}

// CampaignInvite stores invite details used by campaign invites pages.
type CampaignInvite struct {
	ID              string `json:"id"`
	ParticipantID   string `json:"participantId"`
	RecipientUserID string `json:"recipientUserId"`
	Status          string `json:"status"`
}

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

// CatalogClass stores class catalog data used by workflow forms.
type CatalogClass struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	DomainIDs []string `json:"domainIds"`
}

// CatalogSubclass stores subclass catalog data used by workflow forms.
type CatalogSubclass struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	ClassID string `json:"classId"`
}

// CatalogHeritage stores ancestry/community catalog data.
type CatalogHeritage struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// CatalogWeapon stores weapon catalog data used by equipment forms.
type CatalogWeapon struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Tier     int32  `json:"tier"`
}

// CatalogArmor stores armor catalog data used by equipment forms.
type CatalogArmor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Tier int32  `json:"tier"`
}

// CatalogItem stores item catalog data used by equipment forms.
type CatalogItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CatalogDomainCard stores domain card catalog data used by forms.
type CatalogDomainCard struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	DomainID string `json:"domainId"`
	Level    int32  `json:"level"`
}

// CampaignCharacterCreationCatalog stores Daggerheart catalog subsets used by workflow forms.
type CampaignCharacterCreationCatalog struct {
	Classes     []CatalogClass      `json:"classes"`
	Subclasses  []CatalogSubclass   `json:"subclasses"`
	Heritages   []CatalogHeritage   `json:"heritages"`
	Weapons     []CatalogWeapon     `json:"weapons"`
	Armor       []CatalogArmor      `json:"armor"`
	Items       []CatalogItem       `json:"items"`
	DomainCards []CatalogDomainCard `json:"domainCards"`
}

// CampaignCharacterCreationProfile stores selected workflow fields used for filtering options.
type CampaignCharacterCreationProfile struct {
	ClassID            string   `json:"classId"`
	SubclassID         string   `json:"subclassId"`
	AncestryID         string   `json:"ancestryId"`
	CommunityID        string   `json:"communityId"`
	Agility            string   `json:"agility"`
	Strength           string   `json:"strength"`
	Finesse            string   `json:"finesse"`
	Instinct           string   `json:"instinct"`
	Presence           string   `json:"presence"`
	Knowledge          string   `json:"knowledge"`
	PrimaryWeaponID    string   `json:"primaryWeaponId"`
	SecondaryWeaponID  string   `json:"secondaryWeaponId"`
	ArmorID            string   `json:"armorId"`
	PotionItemID       string   `json:"potionItemId"`
	Background         string   `json:"background"`
	ExperienceName     string   `json:"experienceName"`
	ExperienceModifier string   `json:"experienceModifier"`
	DomainCardIDs      []string `json:"domainCardIds"`
	Connections        string   `json:"connections"`
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
	ClassID    string `json:"classId"`
	SubclassID string `json:"subclassId"`
}

// CampaignCharacterCreationStepHeritage stores ancestry/community step input.
type CampaignCharacterCreationStepHeritage struct {
	AncestryID  string `json:"ancestryId"`
	CommunityID string `json:"communityId"`
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
type CampaignCharacterCreationStepDetails struct{}

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

// CampaignCharacterCreation stores character-detail workflow UI data.
type CampaignCharacterCreation struct {
	Progress         CampaignCharacterCreationProgress `json:"progress"`
	Profile          CampaignCharacterCreationProfile  `json:"profile"`
	Classes          []CatalogClass                    `json:"classes"`
	Subclasses       []CatalogSubclass                 `json:"subclasses"`
	Ancestries       []CatalogHeritage                 `json:"ancestries"`
	Communities      []CatalogHeritage                 `json:"communities"`
	PrimaryWeapons   []CatalogWeapon                   `json:"primaryWeapons"`
	SecondaryWeapons []CatalogWeapon                   `json:"secondaryWeapons"`
	Armor            []CatalogArmor                    `json:"armor"`
	PotionItems      []CatalogItem                     `json:"potionItems"`
	DomainCards      []CatalogDomainCard               `json:"domainCards"`
}

// CreateCampaignInput stores create-campaign form values.
type CreateCampaignInput struct {
	Name        string
	Locale      language.Tag
	System      GameSystem
	GMMode      GmMode
	ThemePrompt string
}

// CreateCampaignResult stores create-campaign response values.
type CreateCampaignResult struct {
	CampaignID string
}

// CreateCharacterInput stores create-character form values.
type CreateCharacterInput struct {
	Name string
	Kind CharacterKind
}

// CreateCharacterResult stores create-character response values.
type CreateCharacterResult struct {
	CharacterID string
}
