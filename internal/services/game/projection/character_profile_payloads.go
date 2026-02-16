package projection

type daggerheartProfilePayload struct {
	Level           int                            `json:"level"`
	HpMax           int                            `json:"hp_max"`
	StressMax       int                            `json:"stress_max"`
	Evasion         int                            `json:"evasion"`
	MajorThreshold  int                            `json:"major_threshold"`
	SevereThreshold int                            `json:"severe_threshold"`
	Proficiency     int                            `json:"proficiency"`
	ArmorScore      int                            `json:"armor_score"`
	ArmorMax        int                            `json:"armor_max"`
	Experiences     []daggerheartExperiencePayload `json:"experiences"`
	Agility         int                            `json:"agility"`
	Strength        int                            `json:"strength"`
	Finesse         int                            `json:"finesse"`
	Instinct        int                            `json:"instinct"`
	Presence        int                            `json:"presence"`
	Knowledge       int                            `json:"knowledge"`
}

type daggerheartExperiencePayload struct {
	Name     string `json:"name"`
	Modifier int    `json:"modifier"`
}
