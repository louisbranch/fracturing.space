package coreprojection

import (
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

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
