package coreprojection

import (
	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

func dbInviteToDomain(row db.Invite) (storage.InviteRecord, error) {
	return storage.InviteRecord{
		ID:                     row.ID,
		CampaignID:             row.CampaignID,
		ParticipantID:          row.ParticipantID,
		RecipientUserID:        row.RecipientUserID,
		Status:                 enumFromStorage(row.Status, invite.NormalizeStatus),
		CreatedByParticipantID: row.CreatedByParticipantID,
		CreatedAt:              sqliteutil.FromMillis(row.CreatedAt),
		UpdatedAt:              sqliteutil.FromMillis(row.UpdatedAt),
	}, nil
}
