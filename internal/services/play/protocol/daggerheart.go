package protocol

import (
	"strings"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const playAvatarDeliveryWidthPX = 384

// DaggerheartCharacterCardData matches the browser DaggerheartCharacterCardData
// contract for rendering character portrait cards.
type DaggerheartCharacterCardData struct {
	ID          string                        `json:"id"`
	Name        string                        `json:"name"`
	Portrait    CharacterCardPortrait         `json:"portrait"`
	Identity    *CharacterCardIdentity        `json:"identity,omitempty"`
	Daggerheart *DaggerheartCardSystemSection `json:"daggerheart,omitempty"`
}

// CharacterCardPortrait holds avatar rendering data.
type CharacterCardPortrait struct {
	Alt string `json:"alt"`
	Src string `json:"src,omitempty"`
}

// CharacterCardIdentity holds character kind and pronoun metadata.
type CharacterCardIdentity struct {
	Kind       string   `json:"kind,omitempty"`
	Controller string   `json:"controller,omitempty"`
	Pronouns   string   `json:"pronouns,omitempty"`
	Aliases    []string `json:"aliases,omitempty"`
}

// DaggerheartCardSystemSection holds the daggerheart-specific card summary.
type DaggerheartCardSystemSection struct {
	Summary *DaggerheartCharacterSummary `json:"summary,omitempty"`
	Traits  *DaggerheartCharacterTraits  `json:"traits,omitempty"`
}

// DaggerheartCharacterSummary is the compact class/stat line on the card.
type DaggerheartCharacterSummary struct {
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

// DaggerheartCharacterTraits holds the six trait modifiers.
type DaggerheartCharacterTraits struct {
	Agility   string `json:"agility,omitempty"`
	Strength  string `json:"strength,omitempty"`
	Finesse   string `json:"finesse,omitempty"`
	Instinct  string `json:"instinct,omitempty"`
	Presence  string `json:"presence,omitempty"`
	Knowledge string `json:"knowledge,omitempty"`
}

// DaggerheartCharacterSheetData matches the browser DaggerheartCharacterSheetData
// contract for the full character sheet view.
type DaggerheartCharacterSheetData struct {
	ID              string                  `json:"id"`
	Name            string                  `json:"name"`
	Portrait        CharacterCardPortrait   `json:"portrait"`
	Pronouns        string                  `json:"pronouns,omitempty"`
	Level           int32                   `json:"level,omitempty"`
	ClassName       string                  `json:"className,omitempty"`
	SubclassName    string                  `json:"subclassName,omitempty"`
	AncestryName    string                  `json:"ancestryName,omitempty"`
	CommunityName   string                  `json:"communityName,omitempty"`
	Proficiency     *int32                  `json:"proficiency,omitempty"`
	Traits          []DaggerheartTrait      `json:"traits,omitempty"`
	HP              *TrackValue             `json:"hp,omitempty"`
	Stress          *TrackValue             `json:"stress,omitempty"`
	MajorThreshold  *int32                  `json:"majorThreshold,omitempty"`
	SevereThreshold *int32                  `json:"severeThreshold,omitempty"`
	Evasion         *int32                  `json:"evasion,omitempty"`
	Armor           *TrackValue             `json:"armor,omitempty"`
	Hope            *TrackValue             `json:"hope,omitempty"`
	HopeFeature     string                  `json:"hopeFeature,omitempty"`
	ClassFeature    string                  `json:"classFeature,omitempty"`
	PrimaryWeapon   *DaggerheartWeapon      `json:"primaryWeapon,omitempty"`
	SecondaryWeapon *DaggerheartWeapon      `json:"secondaryWeapon,omitempty"`
	ActiveArmor     *DaggerheartArmor       `json:"activeArmor,omitempty"`
	Experiences     []DaggerheartExperience `json:"experiences,omitempty"`
	DomainCards     []DaggerheartDomainCard `json:"domainCards,omitempty"`
	Description     string                  `json:"description,omitempty"`
	Background      string                  `json:"background,omitempty"`
	Connections     string                  `json:"connections,omitempty"`
	LifeState       string                  `json:"lifeState,omitempty"`
	Conditions      []string                `json:"conditions,omitempty"`
	Kind            string                  `json:"kind,omitempty"`
	Controller      string                  `json:"controller,omitempty"`
}

// DaggerheartTrait is one of the six core traits on the sheet.
type DaggerheartTrait struct {
	Name         string   `json:"name"`
	Abbreviation string   `json:"abbreviation"`
	Value        int32    `json:"value"`
	Skills       []string `json:"skills,omitempty"`
}

// DaggerheartExperience is a named experience modifier.
type DaggerheartExperience struct {
	Name     string `json:"name"`
	Modifier int32  `json:"modifier,omitempty"`
}

// DaggerheartDomainCard is a selected domain card reference.
type DaggerheartDomainCard struct {
	Name   string `json:"name"`
	Domain string `json:"domain,omitempty"`
}

// DaggerheartWeapon is one active weapon entry on the full character sheet.
type DaggerheartWeapon struct {
	Name       string `json:"name"`
	Trait      string `json:"trait,omitempty"`
	Range      string `json:"range,omitempty"`
	DamageDice string `json:"damageDice,omitempty"`
	DamageType string `json:"damageType,omitempty"`
	Feature    string `json:"feature,omitempty"`
}

// DaggerheartArmor is the active armor entry on the full character sheet.
type DaggerheartArmor struct {
	Name      string `json:"name"`
	BaseScore *int32 `json:"baseScore,omitempty"`
	Feature   string `json:"feature,omitempty"`
}

// DaggerheartCardFromSheet builds the card data from a GetCharacterSheetResponse.
func DaggerheartCardFromSheet(assetBaseURL string, char *gamev1.Character, profile *daggerheartv1.DaggerheartProfile, state *daggerheartv1.DaggerheartCharacterState) DaggerheartCharacterCardData {
	card := DaggerheartCharacterCardData{
		ID:       strings.TrimSpace(char.GetId()),
		Name:     strings.TrimSpace(char.GetName()),
		Portrait: characterPortrait(assetBaseURL, char),
	}
	card.Identity = characterIdentity(char)

	if profile != nil {
		className, subclassName := resolveClassNames(profile)
		ancestryName, communityName := heritageDisplayNames(profile)
		summary := &DaggerheartCharacterSummary{
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
		// Find the first hope feature from active class features.
		for _, f := range profile.GetActiveClassFeatures() {
			if f.GetHopeFeature() {
				summary.Feature = strings.TrimSpace(f.GetName())
				break
			}
		}

		traits := daggerheartTraits(profile)
		card.Daggerheart = &DaggerheartCardSystemSection{
			Summary: summary,
			Traits:  traits,
		}
	}
	return card
}

// DaggerheartSheetFromResponse builds the full sheet data from a
// GetCharacterSheetResponse.
func DaggerheartSheetFromResponse(assetBaseURL string, char *gamev1.Character, profile *daggerheartv1.DaggerheartProfile, state *daggerheartv1.DaggerheartCharacterState) DaggerheartCharacterSheetData {
	sheet := DaggerheartCharacterSheetData{
		ID:       strings.TrimSpace(char.GetId()),
		Name:     strings.TrimSpace(char.GetName()),
		Portrait: characterPortrait(assetBaseURL, char),
		Pronouns: pronounsString(char.GetPronouns()),
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
		sheet.PrimaryWeapon = daggerheartSheetWeapon(profile.GetPrimaryWeapon())
		sheet.SecondaryWeapon = daggerheartSheetWeapon(profile.GetSecondaryWeapon())
		sheet.ActiveArmor = daggerheartSheetArmor(profile.GetActiveArmor())

		// Hope feature from active class features.
		for _, f := range profile.GetActiveClassFeatures() {
			if f.GetHopeFeature() {
				sheet.HopeFeature = daggerheartFeatureText(f.GetName(), f.GetDescription())
				break
			}
		}
		// First non-hope class feature as the classFeature display.
		for _, f := range profile.GetActiveClassFeatures() {
			if !f.GetHopeFeature() {
				sheet.ClassFeature = daggerheartFeatureText(f.GetName(), f.GetDescription())
				break
			}
		}

		sheet.Traits = daggerheartTraitSlice(profile)
		sheet.Experiences = daggerheartExperiences(profile)
		sheet.DomainCards = daggerheartDomainCards(profile)
	}

	if state != nil {
		sheet.LifeState = lifeStateString(state.GetLifeState())
		sheet.Conditions = conditionLabels(state.GetConditionStates())
	}

	return sheet
}

// --- Helpers ---

func characterPortrait(assetBaseURL string, char *gamev1.Character) CharacterCardPortrait {
	avatarEntityID := strings.TrimSpace(char.GetId())
	if avatarEntityID == "" {
		avatarEntityID = strings.TrimSpace(char.GetCampaignId())
	}
	return CharacterCardPortrait{
		Alt: strings.TrimSpace(char.GetName()),
		Src: websupport.AvatarImageURL(
			assetBaseURL,
			catalog.AvatarRoleCharacter,
			avatarEntityID,
			strings.TrimSpace(char.GetAvatarSetId()),
			strings.TrimSpace(char.GetAvatarAssetId()),
			playAvatarDeliveryWidthPX,
		),
	}
}

func characterIdentity(char *gamev1.Character) *CharacterCardIdentity {
	kind := characterKindString(char.GetKind())
	pronouns := pronounsString(char.GetPronouns())
	aliases := trimStringSlice(char.GetAliases())
	if kind == "" && pronouns == "" && len(aliases) == 0 {
		return nil
	}
	return &CharacterCardIdentity{
		Kind:     kind,
		Pronouns: pronouns,
		Aliases:  aliases,
	}
}

func characterKindString(value gamev1.CharacterKind) string {
	name := strings.TrimSpace(value.String())
	if name == "" || name == gamev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED.String() {
		return ""
	}
	return strings.ToLower(name)
}

// resolveClassNames extracts the class and subclass display names from active
// features already on the profile. The first active class feature at level 1
// carries the class name; the first active subclass feature provides the
// subclass name.
func resolveClassNames(profile *daggerheartv1.DaggerheartProfile) (string, string) {
	className := contentLabelFromID(profile.GetClassId())
	subclassName := contentLabelFromID(profile.GetSubclassId())
	if className != "" || subclassName != "" {
		return className, subclassName
	}
	if features := profile.GetActiveClassFeatures(); len(features) > 0 {
		// Use the first feature name as class name — the profile features are
		// already derived and ordered by the game service.
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

func daggerheartTraits(profile *daggerheartv1.DaggerheartProfile) *DaggerheartCharacterTraits {
	if profile == nil {
		return nil
	}
	return &DaggerheartCharacterTraits{
		Agility:   traitString(profile.GetAgility()),
		Strength:  traitString(profile.GetStrength()),
		Finesse:   traitString(profile.GetFinesse()),
		Instinct:  traitString(profile.GetInstinct()),
		Presence:  traitString(profile.GetPresence()),
		Knowledge: traitString(profile.GetKnowledge()),
	}
}

func daggerheartTraitSlice(profile *daggerheartv1.DaggerheartProfile) []DaggerheartTrait {
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
	traits := make([]DaggerheartTrait, 0, len(defs))
	for _, d := range defs {
		w := d.get()
		if w == nil {
			continue
		}
		traits = append(traits, DaggerheartTrait{
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
	if v >= 0 {
		return "+" + itoa(v)
	}
	return itoa(v)
}

func itoa(v int32) string {
	if v == 0 {
		return "0"
	}
	negative := v < 0
	if negative {
		v = -v
	}
	var buf [12]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func daggerheartExperiences(profile *daggerheartv1.DaggerheartProfile) []DaggerheartExperience {
	protoExps := profile.GetExperiences()
	if len(protoExps) == 0 {
		return nil
	}
	exps := make([]DaggerheartExperience, 0, len(protoExps))
	for _, e := range protoExps {
		name := strings.TrimSpace(e.GetName())
		if name == "" {
			continue
		}
		exps = append(exps, DaggerheartExperience{
			Name:     name,
			Modifier: e.GetModifier(),
		})
	}
	if len(exps) == 0 {
		return nil
	}
	return exps
}

func daggerheartDomainCards(profile *daggerheartv1.DaggerheartProfile) []DaggerheartDomainCard {
	ids := profile.GetDomainCardIds()
	if len(ids) == 0 {
		return nil
	}
	cards := make([]DaggerheartDomainCard, 0, len(ids))
	for _, id := range ids {
		name, domain := domainCardLabelFromID(id)
		if name != "" {
			cards = append(cards, DaggerheartDomainCard{
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

func daggerheartFeatureText(name, description string) string {
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

func daggerheartSheetWeapon(summary *daggerheartv1.DaggerheartSheetWeaponSummary) *DaggerheartWeapon {
	if summary == nil || strings.TrimSpace(summary.GetName()) == "" {
		return nil
	}
	return &DaggerheartWeapon{
		Name:       strings.TrimSpace(summary.GetName()),
		Trait:      strings.TrimSpace(summary.GetTrait()),
		Range:      strings.TrimSpace(summary.GetRange()),
		DamageDice: strings.TrimSpace(summary.GetDamageDice()),
		DamageType: strings.TrimSpace(summary.GetDamageType()),
		Feature:    strings.TrimSpace(summary.GetFeature()),
	}
}

func daggerheartSheetArmor(summary *daggerheartv1.DaggerheartSheetArmorSummary) *DaggerheartArmor {
	if summary == nil || strings.TrimSpace(summary.GetName()) == "" {
		return nil
	}
	return &DaggerheartArmor{
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
	name := strings.TrimSpace(value.String())
	if name == "" || name == daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED.String() {
		return ""
	}
	name = strings.TrimPrefix(name, "DAGGERHEART_LIFE_STATE_")
	return strings.ToLower(name)
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
