package workflowwrite

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/command"

// DomainCommandInput describes one Daggerheart domain command emitted by
// sibling transport packages that share the workflow write path.
type DomainCommandInput struct {
	CampaignID      string
	CommandType     command.Type
	SessionID       string
	SceneID         string
	RequestID       string
	InvocationID    string
	EntityType      string
	EntityID        string
	PayloadJSON     []byte
	MissingEventMsg string
	ApplyErrMessage string
}

// CoreCommandInput describes one core-domain command emitted by Daggerheart
// transport helpers that still flow through the shared workflow write path.
type CoreCommandInput struct {
	CampaignID      string
	CommandType     command.Type
	SessionID       string
	SceneID         string
	RequestID       string
	InvocationID    string
	CorrelationID   string
	EntityType      string
	EntityID        string
	PayloadJSON     []byte
	MissingEventMsg string
	ApplyErrMessage string
}
