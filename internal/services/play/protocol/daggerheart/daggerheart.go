package daggerheart

import (
	"strconv"
	"strings"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	playprotocol "github.com/louisbranch/fracturing.space/internal/services/play/protocol"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// SystemID is the canonical lowercase identifier for the Daggerheart game
// system, used in CharacterInspection.System and related browser contracts.
const SystemID = "daggerheart"

// CharacterCardData matches the browser DaggerheartCharacterCardData contract
// for rendering character portrait cards.
type CharacterCardData struct {
	ID          string                              `json:"id"`
	Name        string                              `json:"name"`
	Portrait    playprotocol.CharacterCardPortrait  `json:"portrait"`
	Identity    *playprotocol.CharacterCardIdentity `json:"identity,omitempty"`
	Daggerheart *CardSystemSection                  `json:"daggerheart,omitempty"`
}

// CardSystemSection holds the daggerheart-specific card summary.
type CardSystemSection struct {
	Summary *CharacterSummary `json:"summary,omitempty"`
	Traits  *CharacterTraits  `json:"traits,omitempty"`
}

// CharacterSummary is the compact class/stat line on the card.
type CharacterSummary struct {
	Level         int32       `json:"level,omitempty"`
	ClassName     string      `json:"className,omitempty"`
	SubclassName  string      `json:"subclassName,omitempty"`
	AncestryName  string      `json:"ancestryName,omitempty"`
	CommunityName string      `json:"communityName,omitempty"`
	HP            *TrackValue `json:"hp,omitempty"`
	Stress        *TrackValue `json:"stress,omitempty"`
	Evasion       *int32      `json:"evasion,omitempty"`
	Armor         *TrackValue `json:"armor,omitempty"`
	Hope          *TrackValue `json:"hope,omitempty"`
	Feature       string      `json:"feature,omitempty"`
}

// TrackValue is a current/max gauge.
type TrackValue struct {
	Current int32 `json:"current"`
	Max     int32 `json:"max"`
}

// CharacterTraits holds the six trait modifiers.
type CharacterTraits struct {
	Agility   string `json:"agility,omitempty"`
	Strength  string `json:"strength,omitempty"`
	Finesse   string `json:"finesse,omitempty"`
	Instinct  string `json:"instinct,omitempty"`
	Presence  string `json:"presence,omitempty"`
	Knowledge string `json:"knowledge,omitempty"`
}

// CharacterSheetData matches the browser DaggerheartCharacterSheetData contract
// for the full character sheet view.
type CharacterSheetData struct {
	ID              string                             `json:"id"`
	Name            string                             `json:"name"`
	Portrait        playprotocol.CharacterCardPortrait `json:"portrait"`
	Pronouns        string                             `json:"pronouns,omitempty"`
	Level           int32                              `json:"level,omitempty"`
	ClassName       string                             `json:"className,omitempty"`
	SubclassName    string                             `json:"subclassName,omitempty"`
	AncestryName    string                             `json:"ancestryName,omitempty"`
	CommunityName   string                             `json:"communityName,omitempty"`
	Proficiency     *int32                             `json:"proficiency,omitempty"`
	Traits          []Trait                            `json:"traits,omitempty"`
	HP              *TrackValue                        `json:"hp,omitempty"`
	Stress          *TrackValue                        `json:"stress,omitempty"`
	MajorThreshold  *int32                             `json:"majorThreshold,omitempty"`
	SevereThreshold *int32                             `json:"severeThreshold,omitempty"`
	Evasion         *int32                             `json:"evasion,omitempty"`
	Armor           *TrackValue                        `json:"armor,omitempty"`
	Hope            *TrackValue                        `json:"hope,omitempty"`
	HopeFeature     string                             `json:"hopeFeature,omitempty"`
	ClassFeature    string                             `json:"classFeature,omitempty"`
	PrimaryWeapon   *Weapon                            `json:"primaryWeapon,omitempty"`
	SecondaryWeapon *Weapon                            `json:"secondaryWeapon,omitempty"`
	ActiveArmor     *Armor                             `json:"activeArmor,omitempty"`
	Experiences     []Experience                       `json:"experiences,omitempty"`
	DomainCards     []DomainCard                       `json:"domainCards,omitempty"`
	Description     string                             `json:"description,omitempty"`
	Background      string                             `json:"background,omitempty"`
	Connections     string                             `json:"connections,omitempty"`
	LifeState       string                             `json:"lifeState,omitempty"`
	Conditions      []string                           `json:"conditions,omitempty"`
	Kind            string                             `json:"kind,omitempty"`
	Controller      string                             `json:"controller,omitempty"`
}

// Trait is one of the six core traits on the sheet.
type Trait struct {
	Name         string   `json:"name"`
	Abbreviation string   `json:"abbreviation"`
	Value        int32    `json:"value"`
	Skills       []string `json:"skills,omitempty"`
}

// Experience is a named experience modifier.
type Experience struct {
	Name     string `json:"name"`
	Modifier int32  `json:"modifier,omitempty"`
}

// DomainCard is a selected domain card reference.
type DomainCard struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Domain      string `json:"domain,omitempty"`
	FeatureText string `json:"featureText,omitempty"`
}

// Weapon is one active weapon entry on the full character sheet.
type Weapon struct {
	Name       string `json:"name"`
	Trait      string `json:"trait,omitempty"`
	Range      string `json:"range,omitempty"`
	DamageDice string `json:"damageDice,omitempty"`
	DamageType string `json:"damageType,omitempty"`
	Feature    string `json:"feature,omitempty"`
}

// Armor is the active armor entry on the full character sheet.
type Armor struct {
	Name      string `json:"name"`
	BaseScore *int32 `json:"baseScore,omitempty"`
	Feature   string `json:"feature,omitempty"`
}

// CardFromSheet builds the card data from a GetCharacterSheetResponse.
func CardFromSheet(assetBaseURL string, char *gamev1.Character, profile *daggerheartv1.DaggerheartProfile, state *daggerheartv1.DaggerheartCharacterState) CharacterCardData {
	card := CharacterCardData{
		ID:       strings.TrimSpace(char.GetId()),
		Name:     strings.TrimSpace(char.GetName()),
		Portrait: characterPortrait(assetBaseURL, char),
	}
	card.Identity = characterIdentity(char)

	if profile != nil {
		className, subclassName := resolveClassNames(profile)
		ancestryName, communityName := heritageDisplayNames(profile)
		summary := &CharacterSummary{
			Level:         profile.GetLevel(),
			ClassName:     className,
			SubclassName:  subclassName,
			AncestryName:  ancestryName,
			CommunityName: communityName,
			HP:            profileHPTrack(profile, state),
			Stress:        profileStressTrack(profile, state),
			Evasion:       wrapperInt32Ptr(profile.GetEvasion()),
			Armor:         profileArmorTrack(profile, state),
			Hope:          hopeTrack(state),
		}
		for _, f := range profile.GetActiveClassFeatures() {
			if f.GetHopeFeature() {
				summary.Feature = strings.TrimSpace(f.GetName())
				break
			}
		}

		traits := characterTraits(profile)
		card.Daggerheart = &CardSystemSection{
			Summary: summary,
			Traits:  traits,
		}
	}
	return card
}

// SheetFromResponse builds the full sheet data from a
// GetCharacterSheetResponse.
func SheetFromResponse(assetBaseURL string, char *gamev1.Character, profile *daggerheartv1.DaggerheartProfile, state *daggerheartv1.DaggerheartCharacterState, domainCardLookup map[string]DomainCard) CharacterSheetData {
	sheet := CharacterSheetData{
		ID:       strings.TrimSpace(char.GetId()),
		Name:     strings.TrimSpace(char.GetName()),
		Portrait: characterPortrait(assetBaseURL, char),
		Pronouns: playprotocol.PronounsString(char.GetPronouns()),
		Kind:     characterKindString(char.GetKind()),
	}

	if profile != nil {
		className, subclassName := resolveClassNames(profile)
		ancestryName, communityName := heritageDisplayNames(profile)
		sheet.Level = profile.GetLevel()
		sheet.ClassName = className
		sheet.SubclassName = subclassName
		sheet.AncestryName = ancestryName
		sheet.CommunityName = communityName
		sheet.Proficiency = wrapperInt32Ptr(profile.GetProficiency())
		sheet.HP = profileHPTrack(profile, state)
		sheet.Stress = profileStressTrack(profile, state)
		sheet.MajorThreshold = wrapperInt32Ptr(profile.GetMajorThreshold())
		sheet.SevereThreshold = wrapperInt32Ptr(profile.GetSevereThreshold())
		sheet.Evasion = wrapperInt32Ptr(profile.GetEvasion())
		sheet.Armor = profileArmorTrack(profile, state)
		sheet.Hope = hopeTrack(state)
		sheet.Description = strings.TrimSpace(profile.GetDescription())
		sheet.Background = strings.TrimSpace(profile.GetBackground())
		sheet.Connections = strings.TrimSpace(profile.GetConnections())
		sheet.PrimaryWeapon = sheetWeapon(profile.GetPrimaryWeapon())
		sheet.SecondaryWeapon = sheetWeapon(profile.GetSecondaryWeapon())
		sheet.ActiveArmor = sheetArmor(profile.GetActiveArmor())

		for _, f := range profile.GetActiveClassFeatures() {
			if f.GetHopeFeature() {
				sheet.HopeFeature = featureText(f.GetName(), f.GetDescription())
				break
			}
		}
		for _, f := range profile.GetActiveClassFeatures() {
			if !f.GetHopeFeature() {
				sheet.ClassFeature = featureText(f.GetName(), f.GetDescription())
				break
			}
		}

		sheet.Traits = traitSlice(profile)
		sheet.Experiences = experiences(profile)
		sheet.DomainCards = domainCards(profile, domainCardLookup)
	}

	if state != nil {
		sheet.LifeState = lifeStateString(state.GetLifeState())
		sheet.Conditions = conditionLabels(state.GetConditionStates())
	}

	return sheet
}

// DomainCardFromContent converts catalog content into the browser domain-card
// shape used by the play character sheet.
func DomainCardFromContent(card *daggerheartv1.DaggerheartDomainCard) DomainCard {
	if card == nil {
		return DomainCard{}
	}
	fallbackName, fallbackDomain := domainCardLabelFromID(card.GetId())
	name := strings.TrimSpace(card.GetName())
	if name == "" {
		name = fallbackName
	}
	domain := contentLabelFromID(card.GetDomainId())
	if domain == "" {
		domain = fallbackDomain
	}
	return DomainCard{
		ID:          strings.TrimSpace(card.GetId()),
		Name:        name,
		Domain:      domain,
		FeatureText: strings.TrimSpace(card.GetFeatureText()),
	}
}

// --- Helpers ---

func characterPortrait(assetBaseURL string, char *gamev1.Character) playprotocol.CharacterCardPortrait {
	avatarEntityID := strings.TrimSpace(char.GetId())
	if avatarEntityID == "" {
		avatarEntityID = strings.TrimSpace(char.GetCampaignId())
	}
	return playprotocol.CharacterCardPortrait{
		Alt: strings.TrimSpace(char.GetName()),
		Src: websupport.AvatarImageURL(
			assetBaseURL,
			catalog.AvatarRoleCharacter,
			avatarEntityID,
			strings.TrimSpace(char.GetAvatarSetId()),
			strings.TrimSpace(char.GetAvatarAssetId()),
			playprotocol.PlayAvatarDeliveryWidthPX,
		),
	}
}

func characterIdentity(char *gamev1.Character) *playprotocol.CharacterCardIdentity {
	kind := characterKindString(char.GetKind())
	pronouns := playprotocol.PronounsString(char.GetPronouns())
	aliases := playprotocol.TrimStringSlice(char.GetAliases())
	if kind == "" && pronouns == "" && len(aliases) == 0 {
		return nil
	}
	return &playprotocol.CharacterCardIdentity{
		Kind:     kind,
		Pronouns: pronouns,
		Aliases:  aliases,
	}
}

func characterKindString(value gamev1.CharacterKind) string {
	return playprotocol.ProtoEnumToLower(value, gamev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED, "")
}

// resolveClassNames extracts the class and subclass display names from active
// features already on the profile.
func resolveClassNames(profile *daggerheartv1.DaggerheartProfile) (string, string) {
	className := contentLabelFromID(profile.GetClassId())
	subclassName := contentLabelFromID(profile.GetSubclassId())
	if className != "" || subclassName != "" {
		return className, subclassName
	}
	if features := profile.GetActiveClassFeatures(); len(features) > 0 {
		for _, f := range features {
			if n := strings.TrimSpace(f.GetName()); n != "" {
				className = n
				break
			}
		}
	}
	for _, track := range profile.GetActiveSubclassFeatures() {
		for _, feats := range [][]*daggerheartv1.DaggerheartActiveSubclassFeature{
			track.GetFoundationFeatures(),
			track.GetSpecializationFeatures(),
			track.GetMasteryFeatures(),
		} {
			for _, f := range feats {
				if n := strings.TrimSpace(f.GetName()); n != "" {
					subclassName = n
					return className, subclassName
				}
			}
		}
	}
	return className, subclassName
}

func heritageDisplayNames(profile *daggerheartv1.DaggerheartProfile) (string, string) {
	if profile == nil {
		return "", ""
	}
	heritage := profile.GetHeritage()
	if heritage == nil {
		return "", ""
	}
	ancestryName := strings.TrimSpace(heritage.GetAncestryName())
	if ancestryName == "" {
		ancestryName = strings.TrimSpace(heritage.GetAncestryLabel())
	}
	return ancestryName, strings.TrimSpace(heritage.GetCommunityName())
}

func profileHPTrack(profile *daggerheartv1.DaggerheartProfile, state *daggerheartv1.DaggerheartCharacterState) *TrackValue {
	max := profile.GetHpMax()
	if max == 0 {
		return nil
	}
	current := max
	if state != nil {
		current = state.GetHp()
	}
	return &TrackValue{Current: current, Max: max}
}

func profileStressTrack(profile *daggerheartv1.DaggerheartProfile, state *daggerheartv1.DaggerheartCharacterState) *TrackValue {
	maxW := profile.GetStressMax()
	if maxW == nil {
		return nil
	}
	max := maxW.GetValue()
	current := int32(0)
	if state != nil {
		current = state.GetStress()
	}
	return &TrackValue{Current: current, Max: max}
}

func profileArmorTrack(profile *daggerheartv1.DaggerheartProfile, state *daggerheartv1.DaggerheartCharacterState) *TrackValue {
	scoreW := profile.GetArmorScore()
	if scoreW == nil {
		return nil
	}
	max := scoreW.GetValue()
	current := max
	if state != nil {
		current = baseArmorSlotsLeft(state)
	}
	if current < 0 {
		current = 0
	}
	if current > max {
		current = max
	}
	return &TrackValue{Current: current, Max: max}
}

func baseArmorSlotsLeft(state *daggerheartv1.DaggerheartCharacterState) int32 {
	if state == nil {
		return 0
	}
	current := state.GetArmor()
	for _, bucket := range state.GetTemporaryArmorBuckets() {
		if amount := bucket.GetAmount(); amount > 0 {
			current -= amount
		}
	}
	if current < 0 {
		return 0
	}
	return current
}

func hopeTrack(state *daggerheartv1.DaggerheartCharacterState) *TrackValue {
	if state == nil {
		return nil
	}
	max := state.GetHopeMax()
	if max == 0 {
		return nil
	}
	return &TrackValue{Current: state.GetHope(), Max: max}
}

func wrapperInt32Ptr(w *wrapperspb.Int32Value) *int32 {
	if w == nil {
		return nil
	}
	v := w.GetValue()
	return &v
}

func characterTraits(profile *daggerheartv1.DaggerheartProfile) *CharacterTraits {
	if profile == nil {
		return nil
	}
	return &CharacterTraits{
		Agility:   traitString(profile.GetAgility()),
		Strength:  traitString(profile.GetStrength()),
		Finesse:   traitString(profile.GetFinesse()),
		Instinct:  traitString(profile.GetInstinct()),
		Presence:  traitString(profile.GetPresence()),
		Knowledge: traitString(profile.GetKnowledge()),
	}
}

func traitSlice(profile *daggerheartv1.DaggerheartProfile) []Trait {
	type traitDef struct {
		name   string
		abbr   string
		skills []string
		get    func() *wrapperspb.Int32Value
	}
	defs := []traitDef{
		{"Agility", "AGI", []string{"Sprint", "Leap", "Maneuver"}, profile.GetAgility},
		{"Strength", "STR", []string{"Lift", "Smash", "Grapple"}, profile.GetStrength},
		{"Finesse", "FIN", []string{"Control", "Hide", "Tinker"}, profile.GetFinesse},
		{"Instinct", "INS", []string{"Perceive", "Sense", "Navigate"}, profile.GetInstinct},
		{"Presence", "PRE", []string{"Charm", "Perform", "Deceive"}, profile.GetPresence},
		{"Knowledge", "KNO", []string{"Recall", "Analyze", "Comprehend"}, profile.GetKnowledge},
	}
	traits := make([]Trait, 0, len(defs))
	for _, d := range defs {
		w := d.get()
		if w == nil {
			continue
		}
		traits = append(traits, Trait{
			Name:         d.name,
			Abbreviation: d.abbr,
			Value:        w.GetValue(),
			Skills:       append([]string(nil), d.skills...),
		})
	}
	if len(traits) == 0 {
		return nil
	}
	return traits
}

func traitString(w *wrapperspb.Int32Value) string {
	if w == nil {
		return ""
	}
	v := w.GetValue()
	s := strconv.FormatInt(int64(v), 10)
	if v >= 0 {
		return "+" + s
	}
	return s
}

func experiences(profile *daggerheartv1.DaggerheartProfile) []Experience {
	protoExps := profile.GetExperiences()
	if len(protoExps) == 0 {
		return nil
	}
	exps := make([]Experience, 0, len(protoExps))
	for _, e := range protoExps {
		name := strings.TrimSpace(e.GetName())
		if name == "" {
			continue
		}
		exps = append(exps, Experience{
			Name:     name,
			Modifier: e.GetModifier(),
		})
	}
	if len(exps) == 0 {
		return nil
	}
	return exps
}

func domainCards(profile *daggerheartv1.DaggerheartProfile, lookup map[string]DomainCard) []DomainCard {
	ids := profile.GetDomainCardIds()
	if len(ids) == 0 {
		return nil
	}
	cards := make([]DomainCard, 0, len(ids))
	for _, id := range ids {
		trimmedID := strings.TrimSpace(id)
		if trimmedID == "" {
			continue
		}
		if enriched, ok := lookup[trimmedID]; ok {
			if enriched.ID == "" {
				enriched.ID = trimmedID
			}
			if enriched.Name != "" {
				cards = append(cards, enriched)
				continue
			}
		}
		name, domain := domainCardLabelFromID(trimmedID)
		if name != "" {
			cards = append(cards, DomainCard{
				ID:     trimmedID,
				Name:   name,
				Domain: domain,
			})
		}
	}
	if len(cards) == 0 {
		return nil
	}
	return cards
}

func featureText(name, description string) string {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	switch {
	case name != "" && description != "":
		return name + ": " + description
	case name != "":
		return name
	default:
		return description
	}
}

func sheetWeapon(summary *daggerheartv1.DaggerheartSheetWeaponSummary) *Weapon {
	if summary == nil || strings.TrimSpace(summary.GetName()) == "" {
		return nil
	}
	return &Weapon{
		Name:       strings.TrimSpace(summary.GetName()),
		Trait:      strings.TrimSpace(summary.GetTrait()),
		Range:      strings.TrimSpace(summary.GetRange()),
		DamageDice: strings.TrimSpace(summary.GetDamageDice()),
		DamageType: strings.TrimSpace(summary.GetDamageType()),
		Feature:    strings.TrimSpace(summary.GetFeature()),
	}
}

func sheetArmor(summary *daggerheartv1.DaggerheartSheetArmorSummary) *Armor {
	if summary == nil || strings.TrimSpace(summary.GetName()) == "" {
		return nil
	}
	return &Armor{
		Name:      strings.TrimSpace(summary.GetName()),
		BaseScore: int32PtrIfNonZero(summary.GetBaseScore()),
		Feature:   strings.TrimSpace(summary.GetFeature()),
	}
}

func contentLabelFromID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if idx := strings.IndexAny(value, ".:"); idx >= 0 && idx < len(value)-1 {
		value = value[idx+1:]
	}
	return humanizeContentSlug(value)
}

func int32PtrIfNonZero(v int32) *int32 {
	if v == 0 {
		return nil
	}
	return &v
}

// domainCardLabelFromID extracts a human-readable (name, domain) pair from a
// composite domain card ID. The expected format after stripping any prefix up
// to the first "." or ":" separator is "{domain}-{name-words}", where the first
// "-" separates the domain slug from the card name slug.
//
// This is a best-effort heuristic: hyphenated domain names (e.g.,
// "bone-harvest-ancient-flame") parse the domain as "Bone" rather than "Bone
// Harvest". Prefer enriched content from the catalog service when available.
func domainCardLabelFromID(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}
	if idx := strings.IndexAny(value, ".:"); idx >= 0 && idx < len(value)-1 {
		value = value[idx+1:]
	}
	domain := ""
	if idx := strings.Index(value, "-"); idx >= 0 && idx < len(value)-1 {
		domain = humanizeContentSlug(value[:idx])
		value = value[idx+1:]
	}
	return humanizeContentSlug(value), domain
}

// humanizeContentSlug converts a hyphen/underscore-separated slug to
// title-cased display text (e.g., "ancient-flame" → "Ancient Flame"). This is
// a best-effort heuristic that title-cases every word; it does not handle
// acronyms or proper-noun casing.
func humanizeContentSlug(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.NewReplacer("-", " ", "_", " ").Replace(value)
	parts := strings.Fields(value)
	if len(parts) == 0 {
		return ""
	}
	for i, part := range parts {
		lower := strings.ToLower(part)
		parts[i] = strings.ToUpper(lower[:1]) + lower[1:]
	}
	return strings.Join(parts, " ")
}

func lifeStateString(value daggerheartv1.DaggerheartLifeState) string {
	return playprotocol.ProtoEnumToLower(value, daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED, "DAGGERHEART_LIFE_STATE_")
}

func conditionLabels(conditions []*daggerheartv1.DaggerheartConditionState) []string {
	if len(conditions) == 0 {
		return nil
	}
	labels := make([]string, 0, len(conditions))
	for _, c := range conditions {
		label := strings.TrimSpace(c.GetLabel())
		if label == "" {
			label = strings.TrimSpace(c.GetCode())
		}
		if label != "" {
			labels = append(labels, label)
		}
	}
	if len(labels) == 0 {
		return nil
	}
	return labels
}
