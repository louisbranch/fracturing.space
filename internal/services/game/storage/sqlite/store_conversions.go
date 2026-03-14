package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

// enumToStorage converts a domain enum to its uppercase storage representation.
func enumToStorage[T ~string](val T) string {
	if val == "" {
		return "UNSPECIFIED"
	}
	return strings.ToUpper(string(val))
}

// enumFromStorage converts an uppercase storage string to a domain enum
// using the domain's existing Normalize function.
func enumFromStorage[T ~string](s string, normalize func(string) (T, bool)) T {
	val, _ := normalize(s)
	return val
}

func boolToInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func intToBool(value int64) bool {
	return value != 0
}

// unmarshalOptionalJSON decodes a JSON string into dest if non-empty.
// Skips silently when raw is blank; returns a labeled error on decode failure.
func unmarshalOptionalJSON[T any](raw string, dest *T, label string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		return fmt.Errorf("decode %s: %w", label, err)
	}
	return nil
}

// Domain conversion helpers

// campaignRowData holds the common fields from campaign row types.
type campaignRowData struct {
	ID               string
	Name             string
	Locale           string
	GameSystem       string
	Status           string
	GmMode           string
	Intent           string
	AccessPolicy     string
	ParticipantCount int64
	CharacterCount   int64
	ThemePrompt      string
	CoverAssetID     string
	CoverSetID       string
	AIAgentID        string
	AIAuthEpoch      int64
	CreatedAt        int64
	UpdatedAt        int64
	CompletedAt      sql.NullInt64
	ArchivedAt       sql.NullInt64
}

func campaignRowDataToDomain(row campaignRowData) (storage.CampaignRecord, error) {
	locale := platformi18n.DefaultLocale()
	if parsed, ok := platformi18n.ParseLocale(row.Locale); ok {
		locale = parsed
	}
	c := storage.CampaignRecord{
		ID:               row.ID,
		Name:             row.Name,
		Locale:           locale,
		System:           enumFromStorage(row.GameSystem, bridge.NormalizeSystemID),
		Status:           enumFromStorage(row.Status, campaign.NormalizeStatus),
		GmMode:           enumFromStorage(row.GmMode, campaign.NormalizeGmMode),
		Intent:           campaign.NormalizeIntent(row.Intent),
		AccessPolicy:     campaign.NormalizeAccessPolicy(row.AccessPolicy),
		ParticipantCount: int(row.ParticipantCount),
		CharacterCount:   int(row.CharacterCount),
		ThemePrompt:      row.ThemePrompt,
		CoverAssetID:     row.CoverAssetID,
		CoverSetID:       row.CoverSetID,
		AIAgentID:        row.AIAgentID,
		AIAuthEpoch:      uint64(row.AIAuthEpoch),
		CreatedAt:        fromMillis(row.CreatedAt),
		UpdatedAt:        fromMillis(row.UpdatedAt),
	}
	c.CompletedAt = fromNullMillis(row.CompletedAt)
	c.ArchivedAt = fromNullMillis(row.ArchivedAt)

	return c, nil
}

func dbGetCampaignRowToDomain(row db.GetCampaignRow) (storage.CampaignRecord, error) {
	return campaignRowDataToDomain(campaignRowData{
		ID:               row.ID,
		Name:             row.Name,
		Locale:           row.Locale,
		GameSystem:       row.GameSystem,
		Status:           row.Status,
		GmMode:           row.GmMode,
		Intent:           row.Intent,
		AccessPolicy:     row.AccessPolicy,
		ParticipantCount: row.ParticipantCount,
		CharacterCount:   row.CharacterCount,
		ThemePrompt:      row.ThemePrompt,
		CoverAssetID:     row.CoverAssetID,
		CoverSetID:       row.CoverSetID,
		AIAgentID:        row.AiAgentID,
		AIAuthEpoch:      row.AiAuthEpoch,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
		CompletedAt:      row.CompletedAt,
		ArchivedAt:       row.ArchivedAt,
	})
}

func dbListCampaignsRowToDomain(row db.ListCampaignsRow) (storage.CampaignRecord, error) {
	return campaignRowDataToDomain(campaignRowData{
		ID:               row.ID,
		Name:             row.Name,
		Locale:           row.Locale,
		GameSystem:       row.GameSystem,
		Status:           row.Status,
		GmMode:           row.GmMode,
		Intent:           row.Intent,
		AccessPolicy:     row.AccessPolicy,
		ParticipantCount: row.ParticipantCount,
		CharacterCount:   row.CharacterCount,
		ThemePrompt:      row.ThemePrompt,
		CoverAssetID:     row.CoverAssetID,
		CoverSetID:       row.CoverSetID,
		AIAgentID:        row.AiAgentID,
		AIAuthEpoch:      row.AiAuthEpoch,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
		CompletedAt:      row.CompletedAt,
		ArchivedAt:       row.ArchivedAt,
	})
}

func dbListAllCampaignsRowToDomain(row db.ListAllCampaignsRow) (storage.CampaignRecord, error) {
	return campaignRowDataToDomain(campaignRowData{
		ID:               row.ID,
		Name:             row.Name,
		Locale:           row.Locale,
		GameSystem:       row.GameSystem,
		Status:           row.Status,
		GmMode:           row.GmMode,
		Intent:           row.Intent,
		AccessPolicy:     row.AccessPolicy,
		ParticipantCount: row.ParticipantCount,
		CharacterCount:   row.CharacterCount,
		ThemePrompt:      row.ThemePrompt,
		CoverAssetID:     row.CoverAssetID,
		CoverSetID:       row.CoverSetID,
		AIAgentID:        row.AiAgentID,
		AIAuthEpoch:      row.AiAuthEpoch,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
		CompletedAt:      row.CompletedAt,
		ArchivedAt:       row.ArchivedAt,
	})
}

type participantRowData struct {
	CampaignID     string
	ID             string
	UserID         string
	DisplayName    string
	Role           string
	Controller     string
	CampaignAccess string
	AvatarSetID    string
	AvatarAssetID  string
	Pronouns       string
	CreatedAt      int64
	UpdatedAt      int64
}

func participantRowDataToDomain(row participantRowData) (storage.ParticipantRecord, error) {
	return storage.ParticipantRecord{
		ID:             row.ID,
		CampaignID:     row.CampaignID,
		UserID:         row.UserID,
		Name:           row.DisplayName,
		Role:           enumFromStorage(row.Role, participant.NormalizeRole),
		Controller:     enumFromStorage(row.Controller, participant.NormalizeController),
		CampaignAccess: enumFromStorage(row.CampaignAccess, participant.NormalizeCampaignAccess),
		AvatarSetID:    row.AvatarSetID,
		AvatarAssetID:  row.AvatarAssetID,
		Pronouns:       row.Pronouns,
		CreatedAt:      fromMillis(row.CreatedAt),
		UpdatedAt:      fromMillis(row.UpdatedAt),
	}, nil
}

func dbGetParticipantRowToDomain(row db.GetParticipantRow) (storage.ParticipantRecord, error) {
	return participantRowDataToDomain(participantRowData{
		CampaignID:     row.CampaignID,
		ID:             row.ID,
		UserID:         row.UserID,
		DisplayName:    row.DisplayName,
		Role:           row.Role,
		Controller:     row.Controller,
		CampaignAccess: row.CampaignAccess,
		AvatarSetID:    row.AvatarSetID,
		AvatarAssetID:  row.AvatarAssetID,
		Pronouns:       row.Pronouns,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	})
}

func dbListParticipantsByCampaignRowToDomain(row db.ListParticipantsByCampaignRow) (storage.ParticipantRecord, error) {
	return participantRowDataToDomain(participantRowData{
		CampaignID:     row.CampaignID,
		ID:             row.ID,
		UserID:         row.UserID,
		DisplayName:    row.DisplayName,
		Role:           row.Role,
		Controller:     row.Controller,
		CampaignAccess: row.CampaignAccess,
		AvatarSetID:    row.AvatarSetID,
		AvatarAssetID:  row.AvatarAssetID,
		Pronouns:       row.Pronouns,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	})
}

func dbListParticipantsByCampaignPagedFirstRowToDomain(row db.ListParticipantsByCampaignPagedFirstRow) (storage.ParticipantRecord, error) {
	return participantRowDataToDomain(participantRowData{
		CampaignID:     row.CampaignID,
		ID:             row.ID,
		UserID:         row.UserID,
		DisplayName:    row.DisplayName,
		Role:           row.Role,
		Controller:     row.Controller,
		CampaignAccess: row.CampaignAccess,
		AvatarSetID:    row.AvatarSetID,
		AvatarAssetID:  row.AvatarAssetID,
		Pronouns:       row.Pronouns,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	})
}

func dbListParticipantsByCampaignPagedRowToDomain(row db.ListParticipantsByCampaignPagedRow) (storage.ParticipantRecord, error) {
	return participantRowDataToDomain(participantRowData{
		CampaignID:     row.CampaignID,
		ID:             row.ID,
		UserID:         row.UserID,
		DisplayName:    row.DisplayName,
		Role:           row.Role,
		Controller:     row.Controller,
		CampaignAccess: row.CampaignAccess,
		AvatarSetID:    row.AvatarSetID,
		AvatarAssetID:  row.AvatarAssetID,
		Pronouns:       row.Pronouns,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	})
}

func dbInviteToDomain(row db.Invite) (storage.InviteRecord, error) {
	return storage.InviteRecord{
		ID:                     row.ID,
		CampaignID:             row.CampaignID,
		ParticipantID:          row.ParticipantID,
		RecipientUserID:        row.RecipientUserID,
		Status:                 enumFromStorage(row.Status, invite.NormalizeStatus),
		CreatedByParticipantID: row.CreatedByParticipantID,
		CreatedAt:              fromMillis(row.CreatedAt),
		UpdatedAt:              fromMillis(row.UpdatedAt),
	}, nil
}

func dbCharacterToDomain(row db.Character) (storage.CharacterRecord, error) {
	participantID := ""
	if row.ControllerParticipantID.Valid {
		participantID = row.ControllerParticipantID.String
	}
	aliases := make([]string, 0)
	if err := unmarshalOptionalJSON(row.AliasesJson, &aliases, "character aliases"); err != nil {
		return storage.CharacterRecord{}, err
	}
	return storage.CharacterRecord{
		ID:                 row.ID,
		CampaignID:         row.CampaignID,
		OwnerParticipantID: row.OwnerParticipantID,
		ParticipantID:      participantID,
		Name:               row.Name,
		Kind:               enumFromStorage(row.Kind, character.NormalizeKind),
		Notes:              row.Notes,
		AvatarSetID:        row.AvatarSetID,
		AvatarAssetID:      row.AvatarAssetID,
		Pronouns:           row.Pronouns,
		Aliases:            aliases,
		CreatedAt:          fromMillis(row.CreatedAt),
		UpdatedAt:          fromMillis(row.UpdatedAt),
	}, nil
}

func dbSessionToDomain(row db.Session) (storage.SessionRecord, error) {
	sess := storage.SessionRecord{
		ID:         row.ID,
		CampaignID: row.CampaignID,
		Name:       row.Name,
		Status:     enumFromStorage(row.Status, session.NormalizeStatus),
		StartedAt:  fromMillis(row.StartedAt),
		UpdatedAt:  fromMillis(row.UpdatedAt),
	}
	sess.EndedAt = fromNullMillis(row.EndedAt)

	return sess, nil
}

func dbSessionGateToStorage(row db.SessionGate) storage.SessionGate {
	gate := storage.SessionGate{
		CampaignID:         row.CampaignID,
		SessionID:          row.SessionID,
		GateID:             row.GateID,
		GateType:           row.GateType,
		Status:             session.GateStatus(strings.ToLower(strings.TrimSpace(row.Status))),
		Reason:             row.Reason,
		CreatedAt:          fromMillis(row.CreatedAt),
		CreatedByActorType: row.CreatedByActorType,
		CreatedByActorID:   row.CreatedByActorID,
		MetadataJSON:       row.MetadataJson,
		ProgressJSON:       row.ProgressJson,
		ResolutionJSON:     row.ResolutionJson,
	}
	gate.ResolvedAt = fromNullMillis(row.ResolvedAt)
	if row.ResolvedByActorType.Valid {
		gate.ResolvedByActorType = row.ResolvedByActorType.String
	}
	if row.ResolvedByActorID.Valid {
		gate.ResolvedByActorID = row.ResolvedByActorID.String
	}
	return gate
}

func dbSessionSpotlightToStorage(row db.SessionSpotlight) storage.SessionSpotlight {
	return storage.SessionSpotlight{
		CampaignID:         row.CampaignID,
		SessionID:          row.SessionID,
		SpotlightType:      session.SpotlightType(strings.ToLower(strings.TrimSpace(row.SpotlightType))),
		CharacterID:        row.CharacterID,
		UpdatedAt:          fromMillis(row.UpdatedAt),
		UpdatedByActorType: row.UpdatedByActorType,
		UpdatedByActorID:   row.UpdatedByActorID,
	}
}

func dbDaggerheartClassToStorage(row db.DaggerheartClass) (storage.DaggerheartClass, error) {
	class := storage.DaggerheartClass{
		ID:              row.ID,
		Name:            row.Name,
		StartingEvasion: int(row.StartingEvasion),
		StartingHP:      int(row.StartingHp),
		CreatedAt:       fromMillis(row.CreatedAt),
		UpdatedAt:       fromMillis(row.UpdatedAt),
	}
	if err := unmarshalOptionalJSON(row.StartingItemsJson, &class.StartingItems, "daggerheart class starting items"); err != nil {
		return storage.DaggerheartClass{}, err
	}
	if err := unmarshalOptionalJSON(row.FeaturesJson, &class.Features, "daggerheart class features"); err != nil {
		return storage.DaggerheartClass{}, err
	}
	if err := unmarshalOptionalJSON(row.HopeFeatureJson, &class.HopeFeature, "daggerheart class hope feature"); err != nil {
		return storage.DaggerheartClass{}, err
	}
	if err := unmarshalOptionalJSON(row.DomainIdsJson, &class.DomainIDs, "daggerheart class domain ids"); err != nil {
		return storage.DaggerheartClass{}, err
	}
	return class, nil
}

func dbDaggerheartSubclassToStorage(row db.DaggerheartSubclass) (storage.DaggerheartSubclass, error) {
	subclass := storage.DaggerheartSubclass{
		ID:             row.ID,
		Name:           row.Name,
		ClassID:        row.ClassID,
		SpellcastTrait: row.SpellcastTrait,
		CreatedAt:      fromMillis(row.CreatedAt),
		UpdatedAt:      fromMillis(row.UpdatedAt),
	}
	if err := unmarshalOptionalJSON(row.FoundationFeaturesJson, &subclass.FoundationFeatures, "daggerheart subclass foundation features"); err != nil {
		return storage.DaggerheartSubclass{}, err
	}
	if err := unmarshalOptionalJSON(row.SpecializationFeaturesJson, &subclass.SpecializationFeatures, "daggerheart subclass specialization features"); err != nil {
		return storage.DaggerheartSubclass{}, err
	}
	if err := unmarshalOptionalJSON(row.MasteryFeaturesJson, &subclass.MasteryFeatures, "daggerheart subclass mastery features"); err != nil {
		return storage.DaggerheartSubclass{}, err
	}
	return subclass, nil
}

func dbDaggerheartHeritageToStorage(row db.DaggerheartHeritage) (storage.DaggerheartHeritage, error) {
	heritage := storage.DaggerheartHeritage{
		ID:        row.ID,
		Name:      row.Name,
		Kind:      row.Kind,
		CreatedAt: fromMillis(row.CreatedAt),
		UpdatedAt: fromMillis(row.UpdatedAt),
	}
	if err := unmarshalOptionalJSON(row.FeaturesJson, &heritage.Features, "daggerheart heritage features"); err != nil {
		return storage.DaggerheartHeritage{}, err
	}
	return heritage, nil
}

func dbDaggerheartExperienceToStorage(row db.DaggerheartExperience) storage.DaggerheartExperienceEntry {
	return storage.DaggerheartExperienceEntry{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		CreatedAt:   fromMillis(row.CreatedAt),
		UpdatedAt:   fromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartAdversaryEntryToStorage(row db.DaggerheartAdversaryEntry) (storage.DaggerheartAdversaryEntry, error) {
	entry := storage.DaggerheartAdversaryEntry{
		ID:              row.ID,
		Name:            row.Name,
		Tier:            int(row.Tier),
		Role:            row.Role,
		Description:     row.Description,
		Motives:         row.Motives,
		Difficulty:      int(row.Difficulty),
		MajorThreshold:  int(row.MajorThreshold),
		SevereThreshold: int(row.SevereThreshold),
		HP:              int(row.Hp),
		Stress:          int(row.Stress),
		Armor:           int(row.Armor),
		AttackModifier:  int(row.AttackModifier),
		CreatedAt:       fromMillis(row.CreatedAt),
		UpdatedAt:       fromMillis(row.UpdatedAt),
	}
	if err := unmarshalOptionalJSON(row.StandardAttackJson, &entry.StandardAttack, "daggerheart adversary standard attack"); err != nil {
		return storage.DaggerheartAdversaryEntry{}, err
	}
	if err := unmarshalOptionalJSON(row.ExperiencesJson, &entry.Experiences, "daggerheart adversary experiences"); err != nil {
		return storage.DaggerheartAdversaryEntry{}, err
	}
	if err := unmarshalOptionalJSON(row.FeaturesJson, &entry.Features, "daggerheart adversary features"); err != nil {
		return storage.DaggerheartAdversaryEntry{}, err
	}
	return entry, nil
}

func dbDaggerheartBeastformToStorage(row db.DaggerheartBeastform) (storage.DaggerheartBeastformEntry, error) {
	entry := storage.DaggerheartBeastformEntry{
		ID:           row.ID,
		Name:         row.Name,
		Tier:         int(row.Tier),
		Examples:     row.Examples,
		Trait:        row.Trait,
		TraitBonus:   int(row.TraitBonus),
		EvasionBonus: int(row.EvasionBonus),
		CreatedAt:    fromMillis(row.CreatedAt),
		UpdatedAt:    fromMillis(row.UpdatedAt),
	}
	if err := unmarshalOptionalJSON(row.AttackJson, &entry.Attack, "daggerheart beastform attack"); err != nil {
		return storage.DaggerheartBeastformEntry{}, err
	}
	if err := unmarshalOptionalJSON(row.AdvantagesJson, &entry.Advantages, "daggerheart beastform advantages"); err != nil {
		return storage.DaggerheartBeastformEntry{}, err
	}
	if err := unmarshalOptionalJSON(row.FeaturesJson, &entry.Features, "daggerheart beastform features"); err != nil {
		return storage.DaggerheartBeastformEntry{}, err
	}
	return entry, nil
}

func dbDaggerheartCompanionExperienceToStorage(row db.DaggerheartCompanionExperience) storage.DaggerheartCompanionExperienceEntry {
	return storage.DaggerheartCompanionExperienceEntry{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		CreatedAt:   fromMillis(row.CreatedAt),
		UpdatedAt:   fromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartLootEntryToStorage(row db.DaggerheartLootEntry) storage.DaggerheartLootEntry {
	return storage.DaggerheartLootEntry{
		ID:          row.ID,
		Name:        row.Name,
		Roll:        int(row.Roll),
		Description: row.Description,
		CreatedAt:   fromMillis(row.CreatedAt),
		UpdatedAt:   fromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartDamageTypeToStorage(row db.DaggerheartDamageType) storage.DaggerheartDamageTypeEntry {
	return storage.DaggerheartDamageTypeEntry{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		CreatedAt:   fromMillis(row.CreatedAt),
		UpdatedAt:   fromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartDomainToStorage(row db.DaggerheartDomain) storage.DaggerheartDomain {
	return storage.DaggerheartDomain{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		CreatedAt:   fromMillis(row.CreatedAt),
		UpdatedAt:   fromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartDomainCardToStorage(row db.DaggerheartDomainCard) storage.DaggerheartDomainCard {
	return storage.DaggerheartDomainCard{
		ID:          row.ID,
		Name:        row.Name,
		DomainID:    row.DomainID,
		Level:       int(row.Level),
		Type:        row.Type,
		RecallCost:  int(row.RecallCost),
		UsageLimit:  row.UsageLimit,
		FeatureText: row.FeatureText,
		CreatedAt:   fromMillis(row.CreatedAt),
		UpdatedAt:   fromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartWeaponToStorage(row db.DaggerheartWeapon) (storage.DaggerheartWeapon, error) {
	weapon := storage.DaggerheartWeapon{
		ID:         row.ID,
		Name:       row.Name,
		Category:   row.Category,
		Tier:       int(row.Tier),
		Trait:      row.Trait,
		Range:      row.Range,
		DamageType: row.DamageType,
		Burden:     int(row.Burden),
		Feature:    row.Feature,
		CreatedAt:  fromMillis(row.CreatedAt),
		UpdatedAt:  fromMillis(row.UpdatedAt),
	}
	if err := unmarshalOptionalJSON(row.DamageDiceJson, &weapon.DamageDice, "daggerheart weapon damage dice"); err != nil {
		return storage.DaggerheartWeapon{}, err
	}
	return weapon, nil
}

func dbDaggerheartArmorToStorage(row db.DaggerheartArmor) storage.DaggerheartArmor {
	return storage.DaggerheartArmor{
		ID:                  row.ID,
		Name:                row.Name,
		Tier:                int(row.Tier),
		BaseMajorThreshold:  int(row.BaseMajorThreshold),
		BaseSevereThreshold: int(row.BaseSevereThreshold),
		ArmorScore:          int(row.ArmorScore),
		Feature:             row.Feature,
		CreatedAt:           fromMillis(row.CreatedAt),
		UpdatedAt:           fromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartItemToStorage(row db.DaggerheartItem) storage.DaggerheartItem {
	return storage.DaggerheartItem{
		ID:          row.ID,
		Name:        row.Name,
		Rarity:      row.Rarity,
		Kind:        row.Kind,
		StackMax:    int(row.StackMax),
		Description: row.Description,
		EffectText:  row.EffectText,
		CreatedAt:   fromMillis(row.CreatedAt),
		UpdatedAt:   fromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartEnvironmentToStorage(row db.DaggerheartEnvironment) (storage.DaggerheartEnvironment, error) {
	env := storage.DaggerheartEnvironment{
		ID:         row.ID,
		Name:       row.Name,
		Tier:       int(row.Tier),
		Type:       row.Type,
		Difficulty: int(row.Difficulty),
		CreatedAt:  fromMillis(row.CreatedAt),
		UpdatedAt:  fromMillis(row.UpdatedAt),
	}
	if err := unmarshalOptionalJSON(row.ImpulsesJson, &env.Impulses, "daggerheart environment impulses"); err != nil {
		return storage.DaggerheartEnvironment{}, err
	}
	if err := unmarshalOptionalJSON(row.PotentialAdversaryIdsJson, &env.PotentialAdversaryIDs, "daggerheart environment adversaries"); err != nil {
		return storage.DaggerheartEnvironment{}, err
	}
	if err := unmarshalOptionalJSON(row.FeaturesJson, &env.Features, "daggerheart environment features"); err != nil {
		return storage.DaggerheartEnvironment{}, err
	}
	if err := unmarshalOptionalJSON(row.PromptsJson, &env.Prompts, "daggerheart environment prompts"); err != nil {
		return storage.DaggerheartEnvironment{}, err
	}
	return env, nil
}

func dbDaggerheartContentStringToStorage(row db.DaggerheartContentString) storage.DaggerheartContentString {
	return storage.DaggerheartContentString{
		ContentID:   row.ContentID,
		ContentType: row.ContentType,
		Field:       row.Field,
		Locale:      row.Locale,
		Text:        row.Text,
		CreatedAt:   fromMillis(row.CreatedAt),
		UpdatedAt:   fromMillis(row.UpdatedAt),
	}
}
