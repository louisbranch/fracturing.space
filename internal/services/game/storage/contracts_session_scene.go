package storage

import (
	"context"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

// SessionRecord captures session lifecycle metadata that defines active session boundaries.
type SessionRecord struct {
	ID         string
	CampaignID string
	Name       string
	Status     session.Status
	StartedAt  time.Time
	UpdatedAt  time.Time
	EndedAt    *time.Time
}

// SessionPage describes a page of session records.
type SessionPage struct {
	Sessions      []SessionRecord
	NextPageToken string
}

// SessionReader provides read-only access to session projections.
type SessionReader interface {
	// GetSession retrieves a session by campaign ID and session ID.
	// Returns ErrNotFound if the session does not exist.
	GetSession(ctx context.Context, campaignID, sessionID string) (SessionRecord, error)
	// GetActiveSession retrieves the active session for a campaign, if one exists.
	// Returns ErrNotFound if no active session exists.
	GetActiveSession(ctx context.Context, campaignID string) (SessionRecord, error)
	// ListSessions returns a page of session records for a campaign starting after the page token.
	ListSessions(ctx context.Context, campaignID string, pageSize int, pageToken string) (SessionPage, error)
}

// SessionStore owns active/completed session state used by replay, API, and CLI flows.
// Projection handlers use the full interface; read-only consumers should prefer
// SessionReader.
type SessionStore interface {
	SessionReader
	// PutSession atomically stores a session and sets it as the active session for the campaign.
	// Returns ErrActiveSessionExists if an active session already exists for the campaign.
	PutSession(ctx context.Context, s SessionRecord) error
	// EndSession marks a session as ended and clears it as active for the campaign.
	// The boolean return value reports whether the session transitioned to ENDED.
	EndSession(ctx context.Context, campaignID, sessionID string, endedAt time.Time) (SessionRecord, bool, error)
}

// SessionGate describes one gate and its resolution lifecycle within a session.
type SessionGate struct {
	CampaignID          string
	SessionID           string
	GateID              string
	GateType            string
	Status              session.GateStatus
	Reason              string
	CreatedAt           time.Time
	CreatedByActorType  string
	CreatedByActorID    string
	ResolvedAt          *time.Time
	ResolvedByActorType string
	ResolvedByActorID   string
	Metadata            map[string]any
	Progress            *session.GateProgress
	Resolution          map[string]any
}

// SessionGateStore persists gate state for the same lifecycle rules the game engine enforces.
// SessionGateReader provides read-only access to session gate projections.
type SessionGateReader interface {
	// GetSessionGate retrieves a gate by id.
	// Returns ErrNotFound if the gate does not exist.
	GetSessionGate(ctx context.Context, campaignID, sessionID, gateID string) (SessionGate, error)
	// GetOpenSessionGate retrieves the currently open gate for a session.
	// Returns ErrNotFound if no open gate exists.
	GetOpenSessionGate(ctx context.Context, campaignID, sessionID string) (SessionGate, error)
}

// SessionGateStore owns session gate lifecycle state. Projection handlers use
// the full interface; read-only consumers should prefer SessionGateReader.
type SessionGateStore interface {
	SessionGateReader
	// PutSessionGate stores a gate record.
	PutSessionGate(ctx context.Context, gate SessionGate) error
}

// SessionSpotlight captures spotlight turn ownership so clients can read turn-order intent.
type SessionSpotlight struct {
	CampaignID         string
	SessionID          string
	SpotlightType      session.SpotlightType
	CharacterID        string
	UpdatedAt          time.Time
	UpdatedByActorType string
	UpdatedByActorID   string
}

// SessionSpotlightStore persists current spotlight state for session-facing APIs.
// SessionSpotlightReader provides read-only access to session spotlight projections.
type SessionSpotlightReader interface {
	// GetSessionSpotlight retrieves the current spotlight for a session.
	// Returns ErrNotFound if no spotlight is set.
	GetSessionSpotlight(ctx context.Context, campaignID, sessionID string) (SessionSpotlight, error)
}

// SessionSpotlightStore owns session spotlight turn state. Projection handlers use
// the full interface; read-only consumers should prefer SessionSpotlightReader.
type SessionSpotlightStore interface {
	SessionSpotlightReader
	// PutSessionSpotlight stores the current spotlight for a session.
	PutSessionSpotlight(ctx context.Context, spotlight SessionSpotlight) error
	// ClearSessionSpotlight removes the spotlight for a session.
	ClearSessionSpotlight(ctx context.Context, campaignID, sessionID string) error
}

// SceneRecord captures scene lifecycle metadata for projection reads.
type SceneRecord struct {
	CampaignID  string
	SceneID     string
	SessionID   string
	Name        string
	Description string
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	EndedAt     *time.Time
}

// ScenePage describes a page of scene records.
type ScenePage struct {
	Scenes        []SceneRecord
	NextPageToken string
}

// SceneCharacterRecord captures a character's presence in a scene.
type SceneCharacterRecord struct {
	CampaignID  string
	SceneID     string
	CharacterID string
	AddedAt     time.Time
}

// SceneGate describes one gate and its resolution lifecycle within a scene.
type SceneGate struct {
	CampaignID          string
	SceneID             string
	GateID              string
	GateType            string
	Status              session.GateStatus
	Reason              string
	CreatedAt           time.Time
	CreatedByActorType  string
	CreatedByActorID    string
	ResolvedAt          *time.Time
	ResolvedByActorType string
	ResolvedByActorID   string
	MetadataJSON        []byte
	ResolutionJSON      []byte
}

// SceneSpotlight captures spotlight turn ownership within a scene.
type SceneSpotlight struct {
	CampaignID         string
	SceneID            string
	SpotlightType      scene.SpotlightType
	CharacterID        string
	UpdatedAt          time.Time
	UpdatedByActorType string
	UpdatedByActorID   string
}

// SceneReader provides read-only access to scene projections.
type SceneReader interface {
	// GetScene retrieves a scene by campaign ID and scene ID.
	// Returns ErrNotFound if the scene does not exist.
	GetScene(ctx context.Context, campaignID, sceneID string) (SceneRecord, error)
	// ListScenes returns a page of scene records for a session.
	ListScenes(ctx context.Context, campaignID, sessionID string, pageSize int, pageToken string) (ScenePage, error)
	// ListActiveScenes returns all active scenes for a campaign.
	ListActiveScenes(ctx context.Context, campaignID string) ([]SceneRecord, error)
	// ListVisibleActiveScenesForCharacters returns the active session scenes that
	// contain at least one of the provided character ids.
	ListVisibleActiveScenesForCharacters(ctx context.Context, campaignID, sessionID string, characterIDs []string) ([]SceneRecord, error)
}

// SceneStore owns scene lifecycle read state. Projection handlers use
// the full interface; read-only consumers should prefer SceneReader.
type SceneStore interface {
	SceneReader
	// PutScene stores a scene record.
	PutScene(ctx context.Context, s SceneRecord) error
	// EndScene marks a scene as ended.
	// Returns ErrNotFound if the scene does not exist.
	EndScene(ctx context.Context, campaignID, sceneID string, endedAt time.Time) error
}

// SceneCharacterReader provides read-only access to scene character projections.
type SceneCharacterReader interface {
	// ListSceneCharacters returns all characters in a scene.
	ListSceneCharacters(ctx context.Context, campaignID, sceneID string) ([]SceneCharacterRecord, error)
}

// SceneCharacterStore owns scene character membership. Projection handlers use
// the full interface; read-only consumers should prefer SceneCharacterReader.
type SceneCharacterStore interface {
	SceneCharacterReader
	// PutSceneCharacter adds a character to a scene.
	PutSceneCharacter(ctx context.Context, rec SceneCharacterRecord) error
	// DeleteSceneCharacter removes a character from a scene.
	DeleteSceneCharacter(ctx context.Context, campaignID, sceneID, characterID string) error
}

// SceneGateReader provides read-only access to scene gate projections.
type SceneGateReader interface {
	// GetSceneGate retrieves a gate by id.
	// Returns ErrNotFound if the gate does not exist.
	GetSceneGate(ctx context.Context, campaignID, sceneID, gateID string) (SceneGate, error)
	// GetOpenSceneGate retrieves the currently open gate for a scene.
	// Returns ErrNotFound if no open gate exists.
	GetOpenSceneGate(ctx context.Context, campaignID, sceneID string) (SceneGate, error)
}

// SceneGateStore owns scene gate lifecycle state. Projection handlers use
// the full interface; read-only consumers should prefer SceneGateReader.
type SceneGateStore interface {
	SceneGateReader
	// PutSceneGate stores a gate record.
	PutSceneGate(ctx context.Context, gate SceneGate) error
}

// SceneSpotlightReader provides read-only access to scene spotlight projections.
type SceneSpotlightReader interface {
	// GetSceneSpotlight retrieves the current spotlight for a scene.
	// Returns ErrNotFound if no spotlight is set.
	GetSceneSpotlight(ctx context.Context, campaignID, sceneID string) (SceneSpotlight, error)
}

// SceneSpotlightStore owns scene spotlight turn state. Projection handlers use
// the full interface; read-only consumers should prefer SceneSpotlightReader.
type SceneSpotlightStore interface {
	SceneSpotlightReader
	// PutSceneSpotlight stores the current spotlight for a scene.
	PutSceneSpotlight(ctx context.Context, spotlight SceneSpotlight) error
	// ClearSceneSpotlight removes the spotlight for a scene.
	ClearSceneSpotlight(ctx context.Context, campaignID, sceneID string) error
}
