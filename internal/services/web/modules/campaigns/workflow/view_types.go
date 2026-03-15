package workflow

// CharacterCreationStepView carries one step status row for the
// character-creation workflow.
type CharacterCreationStepView struct {
	Step     int32
	Key      string
	Complete bool
}

// CreationClassFeatureView carries one feature paragraph for class, heritage,
// and subclass cards.
type CreationClassFeatureView struct {
	Name        string
	Description string
}

// CreationDomainWatermarkView carries class-domain icon metadata for selectable
// cards.
type CreationDomainWatermarkView struct {
	ID      string
	Name    string
	IconURL string
}

// CreationClassView carries one class option card.
type CreationClassView struct {
	ID               string
	Name             string
	ImageURL         string
	StartingHP       int32
	StartingEvasion  int32
	HopeFeature      CreationClassFeatureView
	Features         []CreationClassFeatureView
	DomainNames      []string
	DomainWatermarks []CreationDomainWatermarkView
}

// CreationSubclassView carries one subclass option card.
type CreationSubclassView struct {
	ID             string
	Name           string
	ImageURL       string
	ClassID        string
	SpellcastTrait string
	Foundation     []CreationClassFeatureView
}

// CreationHeritageView carries one ancestry or community option card.
type CreationHeritageView struct {
	ID       string
	Name     string
	ImageURL string
	Features []CreationClassFeatureView
}

// CreationWeaponView carries one weapon choice.
type CreationWeaponView struct {
	ID       string
	Name     string
	ImageURL string
	Burden   int32
	Trait    string
	Range    string
	Damage   string
	Feature  string
}

// CreationArmorView carries one armor choice.
type CreationArmorView struct {
	ID             string
	Name           string
	ImageURL       string
	ArmorScore     int32
	BaseThresholds string
	Feature        string
}

// CreationItemView carries one item choice.
type CreationItemView struct {
	ID          string
	Name        string
	ImageURL    string
	Description string
}

// CreationExperienceView carries one freeform experience row.
type CreationExperienceView struct {
	Name     string
	Modifier string
}

// CreationDomainCardView carries one domain-card choice.
type CreationDomainCardView struct {
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

// CharacterCreationView carries the workflow-owned page model for one
// character-creation state before HTML render adaptation.
type CharacterCreationView struct {
	Ready                       bool
	NextStep                    int32
	UnmetReasons                []string
	ClassID                     string
	SubclassID                  string
	AncestryID                  string
	CommunityID                 string
	Agility                     string
	Strength                    string
	Finesse                     string
	Instinct                    string
	Presence                    string
	Knowledge                   string
	PrimaryWeaponID             string
	SecondaryWeaponID           string
	ArmorID                     string
	PotionItemID                string
	Background                  string
	Description                 string
	Experiences                 []CreationExperienceView
	DomainCardIDs               []string
	Connections                 string
	NextStepPrefetchURLs        []string
	Steps                       []CharacterCreationStepView
	Classes                     []CreationClassView
	Subclasses                  []CreationSubclassView
	Ancestries                  []CreationHeritageView
	Communities                 []CreationHeritageView
	PrimaryWeapons              []CreationWeaponView
	SecondaryWeapons            []CreationWeaponView
	SecondaryWeaponNoneImageURL string
	Armor                       []CreationArmorView
	PotionItems                 []CreationItemView
	DomainCards                 []CreationDomainCardView
}
