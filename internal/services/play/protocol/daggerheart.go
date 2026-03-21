package protocol

import (
	"strings"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

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
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	Value        int32  `json:"value"`
}

// DaggerheartExperience is a named experience modifier.
type DaggerheartExperience struct {
	Name     string `json:"name"`
	Modifier int32  `json:"modifier,omitempty"`
}

// DaggerheartDomainCard is a selected domain card reference.
type DaggerheartDomainCard struct {
	Name string `json:"name"`
}

// DaggerheartCardFromSheet builds the card data from a GetCharacterSheetResponse.
func DaggerheartCardFromSheet(char *gamev1.Character, profile *daggerheartv1.DaggerheartProfile, state *daggerheartv1.DaggerheartCharacterState) DaggerheartCharacterCardData {
	card := DaggerheartCharacterCardData{
		ID:       strings.TrimSpace(char.GetId()),
		Name:     strings.TrimSpace(char.GetName()),
		Portrait: characterPortrait(char),
	}
	card.Identity = characterIdentity(char)

	if profile != nil {
		className, subclassName := resolveClassNames(profile)
		summary := &DaggerheartCharacterSummary{
			Level:        profile.GetLevel(),
			ClassName:    className,
			SubclassName: subclassName,
			AncestryName: ancestryName(profile),
			HP:           profileHPTrack(profile, state),
			Stress:       profileStressTrack(profile, state),
			Evasion:      wrapperInt32Ptr(profile.GetEvasion()),
			Armor:        profileArmorTrack(profile, state),
			Hope:         hopeTrack(state),
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
func DaggerheartSheetFromResponse(char *gamev1.Character, profile *daggerheartv1.DaggerheartProfile, state *daggerheartv1.DaggerheartCharacterState) DaggerheartCharacterSheetData {
	sheet := DaggerheartCharacterSheetData{
		ID:       strings.TrimSpace(char.GetId()),
		Name:     strings.TrimSpace(char.GetName()),
		Portrait: characterPortrait(char),
		Pronouns: pronounsString(char.GetPronouns()),
		Kind:     characterKindString(char.GetKind()),
	}

	if profile != nil {
		className, subclassName := resolveClassNames(profile)
		sheet.Level = profile.GetLevel()
		sheet.ClassName = className
		sheet.SubclassName = subclassName
		sheet.AncestryName = ancestryName(profile)
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

		// Hope feature from active class features.
		for _, f := range profile.GetActiveClassFeatures() {
			if f.GetHopeFeature() {
				sheet.HopeFeature = strings.TrimSpace(f.GetName())
				break
			}
		}
		// First non-hope class feature as the classFeature display.
		for _, f := range profile.GetActiveClassFeatures() {
			if !f.GetHopeFeature() {
				sheet.ClassFeature = strings.TrimSpace(f.GetName())
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

func characterPortrait(char *gamev1.Character) CharacterCardPortrait {
	return CharacterCardPortrait{
		Alt: strings.TrimSpace(char.GetName()),
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

func controllerString(value gamev1.Controller) string {
	name := strings.TrimSpace(value.String())
	if name == "" || name == gamev1.Controller_CONTROLLER_UNSPECIFIED.String() {
		return ""
	}
	name = strings.TrimPrefix(name, "CONTROLLER_")
	return strings.ToLower(name)
}

// resolveClassNames extracts the class and subclass display names from active
// features already on the profile. The first active class feature at level 1
// carries the class name; the first active subclass feature provides the
// subclass name.
func resolveClassNames(profile *daggerheartv1.DaggerheartProfile) (string, string) {
	var className, subclassName string
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

func ancestryName(profile *daggerheartv1.DaggerheartProfile) string {
	if h := profile.GetHeritage(); h != nil {
		return strings.TrimSpace(h.GetAncestryLabel())
	}
	return ""
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
	maxW := profile.GetArmorMax()
	if maxW == nil {
		return nil
	}
	max := maxW.GetValue()
	current := max
	if state != nil {
		current = state.GetArmor()
	}
	return &TrackValue{Current: current, Max: max}
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
		name string
		abbr string
		get  func() *wrapperspb.Int32Value
	}
	defs := []traitDef{
		{"Agility", "AGI", profile.GetAgility},
		{"Strength", "STR", profile.GetStrength},
		{"Finesse", "FIN", profile.GetFinesse},
		{"Instinct", "INS", profile.GetInstinct},
		{"Presence", "PRE", profile.GetPresence},
		{"Knowledge", "KNO", profile.GetKnowledge},
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
	// Without content service resolution, use IDs as names for MVP.
	cards := make([]DaggerheartDomainCard, 0, len(ids))
	for _, id := range ids {
		if s := strings.TrimSpace(id); s != "" {
			cards = append(cards, DaggerheartDomainCard{Name: s})
		}
	}
	if len(cards) == 0 {
		return nil
	}
	return cards
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
