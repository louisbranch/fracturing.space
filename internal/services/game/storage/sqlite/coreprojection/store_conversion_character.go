package coreprojection

import (
	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

func dbCharacterToDomain(row db.Character) (storage.CharacterRecord, error) {
	aliases := make([]string, 0)
	if err := unmarshalOptionalJSON(row.AliasesJson, &aliases, "character aliases"); err != nil {
		return storage.CharacterRecord{}, err
	}
	return storage.CharacterRecord{
		ID:                 row.ID,
		CampaignID:         row.CampaignID,
		OwnerParticipantID: row.OwnerParticipantID,
		Name:               row.Name,
		Kind:               enumFromStorage(row.Kind, character.NormalizeKind),
		Notes:              row.Notes,
		AvatarSetID:        row.AvatarSetID,
		AvatarAssetID:      row.AvatarAssetID,
		Pronouns:           row.Pronouns,
		Aliases:            aliases,
		CreatedAt:          sqliteutil.FromMillis(row.CreatedAt),
		UpdatedAt:          sqliteutil.FromMillis(row.UpdatedAt),
	}, nil
}
