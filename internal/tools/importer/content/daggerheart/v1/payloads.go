package catalogimporter

type classPayload struct {
	SystemID      string        `json:"system_id"`
	SystemVersion string        `json:"system_version"`
	Source        string        `json:"source"`
	Locale        string        `json:"locale"`
	Items         []classRecord `json:"items"`
}

type subclassPayload struct {
	SystemID      string           `json:"system_id"`
	SystemVersion string           `json:"system_version"`
	Source        string           `json:"source"`
	Locale        string           `json:"locale"`
	Items         []subclassRecord `json:"items"`
}

type heritagePayload struct {
	SystemID      string           `json:"system_id"`
	SystemVersion string           `json:"system_version"`
	Source        string           `json:"source"`
	Locale        string           `json:"locale"`
	Items         []heritageRecord `json:"items"`
}

type experiencePayload struct {
	SystemID      string             `json:"system_id"`
	SystemVersion string             `json:"system_version"`
	Source        string             `json:"source"`
	Locale        string             `json:"locale"`
	Items         []experienceRecord `json:"items"`
}

type adversaryPayload struct {
	SystemID      string            `json:"system_id"`
	SystemVersion string            `json:"system_version"`
	Source        string            `json:"source"`
	Locale        string            `json:"locale"`
	Items         []adversaryRecord `json:"items"`
}

type beastformPayload struct {
	SystemID      string            `json:"system_id"`
	SystemVersion string            `json:"system_version"`
	Source        string            `json:"source"`
	Locale        string            `json:"locale"`
	Items         []beastformRecord `json:"items"`
}

type companionExperiencePayload struct {
	SystemID      string                      `json:"system_id"`
	SystemVersion string                      `json:"system_version"`
	Source        string                      `json:"source"`
	Locale        string                      `json:"locale"`
	Items         []companionExperienceRecord `json:"items"`
}

type lootEntryPayload struct {
	SystemID      string            `json:"system_id"`
	SystemVersion string            `json:"system_version"`
	Source        string            `json:"source"`
	Locale        string            `json:"locale"`
	Items         []lootEntryRecord `json:"items"`
}

type damageTypePayload struct {
	SystemID      string             `json:"system_id"`
	SystemVersion string             `json:"system_version"`
	Source        string             `json:"source"`
	Locale        string             `json:"locale"`
	Items         []damageTypeRecord `json:"items"`
}

type domainPayload struct {
	SystemID      string         `json:"system_id"`
	SystemVersion string         `json:"system_version"`
	Source        string         `json:"source"`
	Locale        string         `json:"locale"`
	Items         []domainRecord `json:"items"`
}

type domainCardPayload struct {
	SystemID      string             `json:"system_id"`
	SystemVersion string             `json:"system_version"`
	Source        string             `json:"source"`
	Locale        string             `json:"locale"`
	Items         []domainCardRecord `json:"items"`
}

type weaponPayload struct {
	SystemID      string         `json:"system_id"`
	SystemVersion string         `json:"system_version"`
	Source        string         `json:"source"`
	Locale        string         `json:"locale"`
	Items         []weaponRecord `json:"items"`
}

type armorPayload struct {
	SystemID      string        `json:"system_id"`
	SystemVersion string        `json:"system_version"`
	Source        string        `json:"source"`
	Locale        string        `json:"locale"`
	Items         []armorRecord `json:"items"`
}

type itemPayload struct {
	SystemID      string       `json:"system_id"`
	SystemVersion string       `json:"system_version"`
	Source        string       `json:"source"`
	Locale        string       `json:"locale"`
	Items         []itemRecord `json:"items"`
}

type environmentPayload struct {
	SystemID      string              `json:"system_id"`
	SystemVersion string              `json:"system_version"`
	Source        string              `json:"source"`
	Locale        string              `json:"locale"`
	Items         []environmentRecord `json:"items"`
}

type featureRecord struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Level       int    `json:"level"`
}

type hopeFeatureRecord struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	HopeCost    int    `json:"hope_cost"`
}

type classRecord struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	StartingEvasion int               `json:"starting_evasion"`
	StartingHP      int               `json:"starting_hp"`
	StartingItems   []string          `json:"starting_items"`
	Features        []featureRecord   `json:"features"`
	HopeFeature     hopeFeatureRecord `json:"hope_feature"`
	DomainIDs       []string          `json:"domain_ids"`
}

type subclassRecord struct {
	ID                     string          `json:"id"`
	Name                   string          `json:"name"`
	ClassID                string          `json:"class_id"`
	SpellcastTrait         string          `json:"spellcast_trait"`
	FoundationFeatures     []featureRecord `json:"foundation_features"`
	SpecializationFeatures []featureRecord `json:"specialization_features"`
	MasteryFeatures        []featureRecord `json:"mastery_features"`
}

type heritageRecord struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Kind     string          `json:"kind"`
	Features []featureRecord `json:"features"`
}

type experienceRecord struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type adversaryAttackRecord struct {
	Name        string            `json:"name"`
	Range       string            `json:"range"`
	DamageDice  []damageDieRecord `json:"damage_dice"`
	DamageBonus int               `json:"damage_bonus"`
	DamageType  string            `json:"damage_type"`
}

type adversaryExperienceRecord struct {
	Name     string `json:"name"`
	Modifier int    `json:"modifier"`
}

type adversaryFeatureRecord struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	CostType    string `json:"cost_type"`
	Cost        int    `json:"cost"`
}

type adversaryRecord struct {
	ID              string                      `json:"id"`
	Name            string                      `json:"name"`
	Tier            int                         `json:"tier"`
	Role            string                      `json:"role"`
	Description     string                      `json:"description"`
	Motives         string                      `json:"motives"`
	Difficulty      int                         `json:"difficulty"`
	MajorThreshold  int                         `json:"major_threshold"`
	SevereThreshold int                         `json:"severe_threshold"`
	HP              int                         `json:"hp"`
	Stress          int                         `json:"stress"`
	Armor           int                         `json:"armor"`
	AttackModifier  int                         `json:"attack_modifier"`
	StandardAttack  adversaryAttackRecord       `json:"standard_attack"`
	Experiences     []adversaryExperienceRecord `json:"experiences"`
	Features        []adversaryFeatureRecord    `json:"features"`
}

type beastformAttackRecord struct {
	Range       string            `json:"range"`
	Trait       string            `json:"trait"`
	DamageDice  []damageDieRecord `json:"damage_dice"`
	DamageBonus int               `json:"damage_bonus"`
	DamageType  string            `json:"damage_type"`
}

type beastformFeatureRecord struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type beastformRecord struct {
	ID           string                   `json:"id"`
	Name         string                   `json:"name"`
	Tier         int                      `json:"tier"`
	Examples     string                   `json:"examples"`
	Trait        string                   `json:"trait"`
	TraitBonus   int                      `json:"trait_bonus"`
	EvasionBonus int                      `json:"evasion_bonus"`
	Attack       beastformAttackRecord    `json:"attack"`
	Advantages   []string                 `json:"advantages"`
	Features     []beastformFeatureRecord `json:"features"`
}

type companionExperienceRecord struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type lootEntryRecord struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Roll        int    `json:"roll"`
	Description string `json:"description"`
}

type damageTypeRecord struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type domainRecord struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type domainCardRecord struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DomainID    string `json:"domain_id"`
	Level       int    `json:"level"`
	Type        string `json:"type"`
	RecallCost  int    `json:"recall_cost"`
	UsageLimit  string `json:"usage_limit"`
	FeatureText string `json:"feature_text"`
}

type damageDieRecord struct {
	Sides int `json:"sides"`
	Count int `json:"count"`
}

type weaponRecord struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Category   string            `json:"category"`
	Tier       int               `json:"tier"`
	Trait      string            `json:"trait"`
	Range      string            `json:"range"`
	DamageDice []damageDieRecord `json:"damage_dice"`
	DamageType string            `json:"damage_type"`
	Burden     int               `json:"burden"`
	Feature    string            `json:"feature"`
}

type armorRecord struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Tier                int    `json:"tier"`
	BaseMajorThreshold  int    `json:"base_major_threshold"`
	BaseSevereThreshold int    `json:"base_severe_threshold"`
	ArmorScore          int    `json:"armor_score"`
	Feature             string `json:"feature"`
}

type itemRecord struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Rarity      string `json:"rarity"`
	Kind        string `json:"kind"`
	StackMax    int    `json:"stack_max"`
	Description string `json:"description"`
	EffectText  string `json:"effect_text"`
}

type environmentRecord struct {
	ID                    string          `json:"id"`
	Name                  string          `json:"name"`
	Tier                  int             `json:"tier"`
	Type                  string          `json:"type"`
	Difficulty            int             `json:"difficulty"`
	Impulses              []string        `json:"impulses"`
	PotentialAdversaryIDs []string        `json:"potential_adversary_ids"`
	Features              []featureRecord `json:"features"`
	Prompts               []string        `json:"prompts"`
}
