package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

// Conversion helpers

func gameSystemToString(gs commonv1.GameSystem) string {
	switch gs {
	case commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART:
		return "DAGGERHEART"
	default:
		return "UNSPECIFIED"
	}
}

func stringToGameSystem(s string) commonv1.GameSystem {
	switch s {
	case "DAGGERHEART":
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
	default:
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED
	}
}

func campaignStatusToString(status campaign.Status) string {
	switch strings.ToLower(strings.TrimSpace(string(status))) {
	case "draft":
		return "DRAFT"
	case "active":
		return "ACTIVE"
	case "completed":
		return "COMPLETED"
	case "archived":
		return "ARCHIVED"
	default:
		return "UNSPECIFIED"
	}
}

func stringToCampaignStatus(s string) campaign.Status {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DRAFT":
		return campaign.StatusDraft
	case "ACTIVE":
		return campaign.StatusActive
	case "COMPLETED":
		return campaign.StatusCompleted
	case "ARCHIVED":
		return campaign.StatusArchived
	default:
		return campaign.StatusUnspecified
	}
}

func gmModeToString(gm campaign.GmMode) string {
	switch strings.ToLower(strings.TrimSpace(string(gm))) {
	case "human":
		return "HUMAN"
	case "ai":
		return "AI"
	case "hybrid":
		return "HYBRID"
	default:
		return "UNSPECIFIED"
	}
}

func stringToGmMode(s string) campaign.GmMode {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "HUMAN":
		return campaign.GmModeHuman
	case "AI":
		return campaign.GmModeAI
	case "HYBRID":
		return campaign.GmModeHybrid
	default:
		return campaign.GmModeUnspecified
	}
}

func campaignIntentToString(intent campaign.Intent) string {
	switch strings.ToLower(strings.TrimSpace(string(intent))) {
	case "standard":
		return "STANDARD"
	case "starter":
		return "STARTER"
	case "sandbox":
		return "SANDBOX"
	default:
		return "UNSPECIFIED"
	}
}

func stringToCampaignIntent(s string) campaign.Intent {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "STANDARD":
		return campaign.IntentStandard
	case "STARTER":
		return campaign.IntentStarter
	case "SANDBOX":
		return campaign.IntentSandbox
	default:
		return campaign.IntentUnspecified
	}
}

func campaignAccessPolicyToString(policy campaign.AccessPolicy) string {
	switch strings.ToLower(strings.TrimSpace(string(policy))) {
	case "private":
		return "PRIVATE"
	case "restricted":
		return "RESTRICTED"
	case "public":
		return "PUBLIC"
	default:
		return "UNSPECIFIED"
	}
}

func stringToCampaignAccessPolicy(s string) campaign.AccessPolicy {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "PRIVATE":
		return campaign.AccessPolicyPrivate
	case "RESTRICTED":
		return campaign.AccessPolicyRestricted
	case "PUBLIC":
		return campaign.AccessPolicyPublic
	default:
		return campaign.AccessPolicyUnspecified
	}
}

func inviteStatusToString(status invite.Status) string {
	switch strings.ToLower(strings.TrimSpace(string(status))) {
	case "pending":
		return "PENDING"
	case "claimed":
		return "CLAIMED"
	case "revoked":
		return "REVOKED"
	default:
		return "UNSPECIFIED"
	}
}

func stringToInviteStatus(s string) invite.Status {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "PENDING":
		return invite.StatusPending
	case "CLAIMED":
		return invite.StatusClaimed
	case "REVOKED":
		return invite.StatusRevoked
	default:
		return invite.StatusUnspecified
	}
}

func participantRoleToString(role participant.Role) string {
	switch strings.ToLower(strings.TrimSpace(string(role))) {
	case "gm":
		return "GM"
	case "player":
		return "PLAYER"
	default:
		return "UNSPECIFIED"
	}
}

func stringToParticipantRole(s string) participant.Role {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "GM":
		return participant.RoleGM
	case "PLAYER":
		return participant.RolePlayer
	default:
		return participant.RoleUnspecified
	}
}

func participantControllerToString(controller participant.Controller) string {
	switch strings.ToLower(strings.TrimSpace(string(controller))) {
	case "human":
		return "HUMAN"
	case "ai":
		return "AI"
	default:
		return "UNSPECIFIED"
	}
}

func participantAccessToString(access participant.CampaignAccess) string {
	switch strings.ToLower(strings.TrimSpace(string(access))) {
	case "member":
		return "MEMBER"
	case "manager":
		return "MANAGER"
	case "owner":
		return "OWNER"
	default:
		return "UNSPECIFIED"
	}
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

func stringToParticipantController(s string) participant.Controller {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "HUMAN":
		return participant.ControllerHuman
	case "AI":
		return participant.ControllerAI
	default:
		return participant.ControllerUnspecified
	}
}

func stringToParticipantAccess(s string) participant.CampaignAccess {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "MEMBER":
		return participant.CampaignAccessMember
	case "MANAGER":
		return participant.CampaignAccessManager
	case "OWNER":
		return participant.CampaignAccessOwner
	default:
		return participant.CampaignAccessUnspecified
	}
}

func characterKindToString(kind character.Kind) string {
	switch strings.ToLower(strings.TrimSpace(string(kind))) {
	case "pc":
		return "PC"
	case "npc":
		return "NPC"
	default:
		return "UNSPECIFIED"
	}
}

func stringToCharacterKind(s string) character.Kind {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "PC":
		return character.KindPC
	case "NPC":
		return character.KindNPC
	default:
		return character.KindUnspecified
	}
}

func sessionStatusToString(status session.Status) string {
	switch strings.ToLower(strings.TrimSpace(string(status))) {
	case "active":
		return "ACTIVE"
	case "ended":
		return "ENDED"
	default:
		return "UNSPECIFIED"
	}
}

func stringToSessionStatus(s string) session.Status {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "ACTIVE":
		return session.StatusActive
	case "ENDED":
		return session.StatusEnded
	default:
		return session.StatusUnspecified
	}
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
		System:           stringToGameSystem(row.GameSystem),
		Status:           stringToCampaignStatus(row.Status),
		GmMode:           stringToGmMode(row.GmMode),
		Intent:           stringToCampaignIntent(row.Intent),
		AccessPolicy:     stringToCampaignAccessPolicy(row.AccessPolicy),
		ParticipantCount: int(row.ParticipantCount),
		CharacterCount:   int(row.CharacterCount),
		ThemePrompt:      row.ThemePrompt,
		CoverAssetID:     row.CoverAssetID,
		CoverSetID:       row.CoverSetID,
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
		Role:           stringToParticipantRole(row.Role),
		Controller:     stringToParticipantController(row.Controller),
		CampaignAccess: stringToParticipantAccess(row.CampaignAccess),
		AvatarSetID:    row.AvatarSetID,
		AvatarAssetID:  row.AvatarAssetID,
		Pronouns:       row.Pronouns,
		CreatedAt:      fromMillis(row.CreatedAt),
		UpdatedAt:      fromMillis(row.UpdatedAt),
	}, nil
}

func dbParticipantToDomain(row db.Participant) (storage.ParticipantRecord, error) {
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
		Status:                 stringToInviteStatus(row.Status),
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
	if strings.TrimSpace(row.AliasesJson) != "" {
		if err := json.Unmarshal([]byte(row.AliasesJson), &aliases); err != nil {
			return storage.CharacterRecord{}, fmt.Errorf("decode character aliases: %w", err)
		}
	}
	return storage.CharacterRecord{
		ID:                 row.ID,
		CampaignID:         row.CampaignID,
		OwnerParticipantID: row.OwnerParticipantID,
		ParticipantID:      participantID,
		Name:               row.Name,
		Kind:               stringToCharacterKind(row.Kind),
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
		Status:     stringToSessionStatus(row.Status),
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
	if row.StartingItemsJson != "" {
		if err := json.Unmarshal([]byte(row.StartingItemsJson), &class.StartingItems); err != nil {
			return storage.DaggerheartClass{}, fmt.Errorf("decode daggerheart class starting items: %w", err)
		}
	}
	if row.FeaturesJson != "" {
		if err := json.Unmarshal([]byte(row.FeaturesJson), &class.Features); err != nil {
			return storage.DaggerheartClass{}, fmt.Errorf("decode daggerheart class features: %w", err)
		}
	}
	if row.HopeFeatureJson != "" {
		if err := json.Unmarshal([]byte(row.HopeFeatureJson), &class.HopeFeature); err != nil {
			return storage.DaggerheartClass{}, fmt.Errorf("decode daggerheart class hope feature: %w", err)
		}
	}
	if row.DomainIdsJson != "" {
		if err := json.Unmarshal([]byte(row.DomainIdsJson), &class.DomainIDs); err != nil {
			return storage.DaggerheartClass{}, fmt.Errorf("decode daggerheart class domain ids: %w", err)
		}
	}
	return class, nil
}

func dbDaggerheartSubclassToStorage(row db.DaggerheartSubclass) (storage.DaggerheartSubclass, error) {
	subclass := storage.DaggerheartSubclass{
		ID:             row.ID,
		Name:           row.Name,
		SpellcastTrait: row.SpellcastTrait,
		CreatedAt:      fromMillis(row.CreatedAt),
		UpdatedAt:      fromMillis(row.UpdatedAt),
	}
	if row.FoundationFeaturesJson != "" {
		if err := json.Unmarshal([]byte(row.FoundationFeaturesJson), &subclass.FoundationFeatures); err != nil {
			return storage.DaggerheartSubclass{}, fmt.Errorf("decode daggerheart subclass foundation features: %w", err)
		}
	}
	if row.SpecializationFeaturesJson != "" {
		if err := json.Unmarshal([]byte(row.SpecializationFeaturesJson), &subclass.SpecializationFeatures); err != nil {
			return storage.DaggerheartSubclass{}, fmt.Errorf("decode daggerheart subclass specialization features: %w", err)
		}
	}
	if row.MasteryFeaturesJson != "" {
		if err := json.Unmarshal([]byte(row.MasteryFeaturesJson), &subclass.MasteryFeatures); err != nil {
			return storage.DaggerheartSubclass{}, fmt.Errorf("decode daggerheart subclass mastery features: %w", err)
		}
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
	if row.FeaturesJson != "" {
		if err := json.Unmarshal([]byte(row.FeaturesJson), &heritage.Features); err != nil {
			return storage.DaggerheartHeritage{}, fmt.Errorf("decode daggerheart heritage features: %w", err)
		}
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
	if row.StandardAttackJson != "" {
		if err := json.Unmarshal([]byte(row.StandardAttackJson), &entry.StandardAttack); err != nil {
			return storage.DaggerheartAdversaryEntry{}, fmt.Errorf("decode daggerheart adversary standard attack: %w", err)
		}
	}
	if row.ExperiencesJson != "" {
		if err := json.Unmarshal([]byte(row.ExperiencesJson), &entry.Experiences); err != nil {
			return storage.DaggerheartAdversaryEntry{}, fmt.Errorf("decode daggerheart adversary experiences: %w", err)
		}
	}
	if row.FeaturesJson != "" {
		if err := json.Unmarshal([]byte(row.FeaturesJson), &entry.Features); err != nil {
			return storage.DaggerheartAdversaryEntry{}, fmt.Errorf("decode daggerheart adversary features: %w", err)
		}
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
	if row.AttackJson != "" {
		if err := json.Unmarshal([]byte(row.AttackJson), &entry.Attack); err != nil {
			return storage.DaggerheartBeastformEntry{}, fmt.Errorf("decode daggerheart beastform attack: %w", err)
		}
	}
	if row.AdvantagesJson != "" {
		if err := json.Unmarshal([]byte(row.AdvantagesJson), &entry.Advantages); err != nil {
			return storage.DaggerheartBeastformEntry{}, fmt.Errorf("decode daggerheart beastform advantages: %w", err)
		}
	}
	if row.FeaturesJson != "" {
		if err := json.Unmarshal([]byte(row.FeaturesJson), &entry.Features); err != nil {
			return storage.DaggerheartBeastformEntry{}, fmt.Errorf("decode daggerheart beastform features: %w", err)
		}
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
	if row.DamageDiceJson != "" {
		if err := json.Unmarshal([]byte(row.DamageDiceJson), &weapon.DamageDice); err != nil {
			return storage.DaggerheartWeapon{}, fmt.Errorf("decode daggerheart weapon damage dice: %w", err)
		}
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
	if row.ImpulsesJson != "" {
		if err := json.Unmarshal([]byte(row.ImpulsesJson), &env.Impulses); err != nil {
			return storage.DaggerheartEnvironment{}, fmt.Errorf("decode daggerheart environment impulses: %w", err)
		}
	}
	if row.PotentialAdversaryIdsJson != "" {
		if err := json.Unmarshal([]byte(row.PotentialAdversaryIdsJson), &env.PotentialAdversaryIDs); err != nil {
			return storage.DaggerheartEnvironment{}, fmt.Errorf("decode daggerheart environment adversaries: %w", err)
		}
	}
	if row.FeaturesJson != "" {
		if err := json.Unmarshal([]byte(row.FeaturesJson), &env.Features); err != nil {
			return storage.DaggerheartEnvironment{}, fmt.Errorf("decode daggerheart environment features: %w", err)
		}
	}
	if row.PromptsJson != "" {
		if err := json.Unmarshal([]byte(row.PromptsJson), &env.Prompts); err != nil {
			return storage.DaggerheartEnvironment{}, fmt.Errorf("decode daggerheart environment prompts: %w", err)
		}
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
