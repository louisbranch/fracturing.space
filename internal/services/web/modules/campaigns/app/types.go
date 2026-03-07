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
	CanEdit        bool   `json:"canEdit"`
	EditReasonCode string `json:"editReasonCode"`
}

// CampaignParticipantAccessOption stores one campaign-access option state.
type CampaignParticipantAccessOption struct {
	Value   string `json:"value"`
	Allowed bool   `json:"allowed"`
}

// CampaignParticipantEditor stores participant edit page data.
type CampaignParticipantEditor struct {
	Participant    CampaignParticipant               `json:"participant"`
	AccessOptions  []CampaignParticipantAccessOption `json:"accessOptions"`
	AccessReadOnly bool                              `json:"accessReadOnly"`
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

// CampaignSessionReadinessBlocker stores one session-start readiness blocker.
type CampaignSessionReadinessBlocker struct {
	Code     string            `json:"code"`
	Message  string            `json:"message"`
	Metadata map[string]string `json:"metadata"`
}

// CampaignSessionReadiness stores campaign readiness details for session start.
type CampaignSessionReadiness struct {
	Ready    bool                              `json:"ready"`
	Blockers []CampaignSessionReadinessBlocker `json:"blockers"`
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
	ID             string                `json:"id"`
	Name           string                `json:"name"`
	ClassID        string                `json:"classId"`
	SpellcastTrait string                `json:"spellcastTrait"`
	Foundation     []CatalogFeature      `json:"foundation"`
	Illustration   CatalogAssetReference `json:"illustration"`
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
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Tier     int32  `json:"tier"`
	Trait    string `json:"trait"`
	Range    string `json:"range"`
	Damage   string `json:"damage"`
	Feature  string `json:"feature"`
}

// CatalogArmor stores armor catalog data used by equipment forms.
type CatalogArmor struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Tier           int32  `json:"tier"`
	ArmorScore     int32  `json:"armorScore"`
	BaseThresholds string `json:"baseThresholds"`
	Feature        string `json:"feature"`
}

// CatalogItem stores item catalog data used by equipment forms.
type CatalogItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
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
	AssetTheme   string               `json:"assetTheme"`
	Classes      []CatalogClass       `json:"classes"`
	Subclasses   []CatalogSubclass    `json:"subclasses"`
	Heritages    []CatalogHeritage    `json:"heritages"`
	Domains      []CatalogDomain      `json:"domains"`
	Weapons      []CatalogWeapon      `json:"weapons"`
	Armor        []CatalogArmor       `json:"armor"`
	Items        []CatalogItem        `json:"items"`
	DomainCards  []CatalogDomainCard  `json:"domainCards"`
	Adversaries  []CatalogAdversary   `json:"adversaries"`
	Environments []CatalogEnvironment `json:"environments"`
}

// CampaignCharacterCreationExperience stores one experience name+modifier pair.
type CampaignCharacterCreationExperience struct {
	Name     string `json:"name"`
	Modifier string `json:"modifier"`
}

// CampaignCharacterCreationProfile stores selected workflow fields used for filtering options.
type CampaignCharacterCreationProfile struct {
	CharacterName     string                                `json:"characterName"`
	ClassID           string                                `json:"classId"`
	SubclassID        string                                `json:"subclassId"`
	AncestryID        string                                `json:"ancestryId"`
	CommunityID       string                                `json:"communityId"`
	Agility           string                                `json:"agility"`
	Strength          string                                `json:"strength"`
	Finesse           string                                `json:"finesse"`
	Instinct          string                                `json:"instinct"`
	Presence          string                                `json:"presence"`
	Knowledge         string                                `json:"knowledge"`
	PrimaryWeaponID   string                                `json:"primaryWeaponId"`
	SecondaryWeaponID string                                `json:"secondaryWeaponId"`
	ArmorID           string                                `json:"armorId"`
	PotionItemID      string                                `json:"potionItemId"`
	Background        string                                `json:"background"`
	Description       string                                `json:"description"`
	Experiences       []CampaignCharacterCreationExperience `json:"experiences"`
	DomainCardIDs     []string                              `json:"domainCardIds"`
	Connections       string                                `json:"connections"`
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
	Domains          []CatalogDomain                   `json:"domains"`
}

// UpdateCharacterInput stores character update form values.
type UpdateCharacterInput struct {
	Name     string
	Pronouns string
}

// CreateCampaignInput stores create-campaign form values.
type CreateCampaignInput struct {
	Name        string
	Locale      language.Tag
	System      GameSystem
	GMMode      GmMode
	ThemePrompt string
}

// UpdateCampaignInput stores campaign update form values.
type UpdateCampaignInput struct {
	Name        *string
	ThemePrompt *string
	Locale      *string
}

// CreateCampaignResult stores create-campaign response values.
type CreateCampaignResult struct {
	CampaignID string
}

// StartSessionInput stores start-session form values.
type StartSessionInput struct {
	Name string
}

// EndSessionInput stores end-session form values.
type EndSessionInput struct {
	SessionID string
}

// CreateInviteInput stores create-invite form values.
type CreateInviteInput struct {
	ParticipantID   string
	RecipientUserID string
}

// RevokeInviteInput stores revoke-invite form values.
type RevokeInviteInput struct {
	InviteID string
}

// CreateCharacterInput stores create-character form values.
type CreateCharacterInput struct {
	Name string
	Kind CharacterKind
}

// UpdateParticipantInput stores participant update form values.
type UpdateParticipantInput struct {
	ParticipantID  string
	Name           string
	Role           string
	Pronouns       string
	CampaignAccess string
}

// CreateCharacterResult stores create-character response values.
type CreateCharacterResult struct {
	CharacterID string
}
