package gametools

import (
	"context"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration/daggerhearttools"
)

// readResource dispatches a resource URI to the correct gRPC reader and
// returns the text content.
func (s *DirectSession) readResource(ctx context.Context, uri string) (string, error) {
	if value, handled, err := daggerhearttools.ReadResource(daggerheartRuntimeAdapter{session: s}, ctx, uri); handled {
		return value, err
	}

	switch {
	case uri == "context://current":
		return s.readContextCurrent()

	case matchCampaignArtifactURI(uri):
		return s.readCampaignArtifact(ctx, uri)

	case strings.HasSuffix(uri, "/interaction"):
		return s.readInteraction(ctx, uri)

	case strings.HasSuffix(uri, "/recap"):
		return s.readSessionRecap(ctx, uri)

	case strings.HasSuffix(uri, "/scenes"):
		return s.readSceneList(ctx, uri)

	case strings.HasSuffix(uri, "/participants"):
		return s.readParticipantList(ctx, uri)

	case strings.HasSuffix(uri, "/characters"):
		return s.readCharacterList(ctx, uri)

	case strings.HasSuffix(uri, "/sessions"):
		return s.readSessionList(ctx, uri)

	case strings.HasPrefix(uri, "campaign://") && !strings.Contains(strings.TrimPrefix(uri, "campaign://"), "/"):
		return s.readCampaign(ctx, uri)

	default:
		return "", fmt.Errorf("unknown resource URI: %s", uri)
	}
}

func (s *DirectSession) readContextCurrent() (string, error) {
	type contextPayload struct {
		Context struct {
			CampaignID    *string `json:"campaign_id"`
			SessionID     *string `json:"session_id"`
			ParticipantID *string `json:"participant_id"`
		} `json:"context"`
	}

	var payload contextPayload
	if s.sc.CampaignID != "" {
		payload.Context.CampaignID = &s.sc.CampaignID
	}
	if s.sc.SessionID != "" {
		payload.Context.SessionID = &s.sc.SessionID
	}
	if s.sc.ParticipantID != "" {
		payload.Context.ParticipantID = &s.sc.ParticipantID
	}
	return marshalIndent(payload)
}
