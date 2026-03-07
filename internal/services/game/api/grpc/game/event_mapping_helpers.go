package game

import (
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var defaultCampaignProjectionScopes = []string{
	"campaign_summary",
	"campaign_participants",
	"campaign_sessions",
	"campaign_characters",
	"campaign_invites",
	"campaign_scenes",
}

// eventToProto converts a domain event to a proto Event message.
func eventToProto(evt event.Event) *campaignv1.Event {
	return &campaignv1.Event{
		CampaignId:    evt.CampaignID,
		Seq:           evt.Seq,
		Hash:          evt.Hash,
		Ts:            timestamppb.New(evt.Timestamp),
		Type:          string(evt.Type),
		SystemId:      evt.SystemID,
		SystemVersion: evt.SystemVersion,
		SessionId:     evt.SessionID,
		SceneId:       evt.SceneID,
		RequestId:     evt.RequestID,
		InvocationId:  evt.InvocationID,
		ActorType:     string(evt.ActorType),
		ActorId:       evt.ActorID,
		EntityType:    evt.EntityType,
		EntityId:      evt.EntityID,
		PayloadJson:   evt.PayloadJSON,
	}
}

func campaignUpdateEventCommitted(evt event.Event) *campaignv1.CampaignUpdate {
	return &campaignv1.CampaignUpdate{
		CampaignId: evt.CampaignID,
		Seq:        evt.Seq,
		EventType:  string(evt.Type),
		EventTime:  timestamppb.New(evt.Timestamp),
		EntityType: evt.EntityType,
		EntityId:   evt.EntityID,
		Update: &campaignv1.CampaignUpdate_EventCommitted{
			EventCommitted: &campaignv1.EventCommitted{},
		},
	}
}

func campaignUpdateProjectionApplied(evt event.Event, scopes []string) *campaignv1.CampaignUpdate {
	return &campaignv1.CampaignUpdate{
		CampaignId: evt.CampaignID,
		Seq:        evt.Seq,
		EventType:  string(evt.Type),
		EventTime:  timestamppb.New(evt.Timestamp),
		EntityType: evt.EntityType,
		EntityId:   evt.EntityID,
		Update: &campaignv1.CampaignUpdate_ProjectionApplied{
			ProjectionApplied: &campaignv1.ProjectionApplied{
				SourceSeq: evt.Seq,
				Scopes:    append([]string(nil), scopes...),
			},
		},
	}
}

func projectionScopesForEventType(eventType string) []string {
	eventType = strings.TrimSpace(eventType)
	switch {
	case strings.HasPrefix(eventType, "campaign."):
		return []string{"campaign_summary"}
	case strings.HasPrefix(eventType, "participant."):
		return []string{"campaign_participants", "campaign_summary"}
	case strings.HasPrefix(eventType, "session."):
		return []string{"campaign_sessions"}
	case strings.HasPrefix(eventType, "scene."):
		return []string{"campaign_scenes"}
	case strings.HasPrefix(eventType, "character."):
		return []string{"campaign_characters", "campaign_summary"}
	case strings.HasPrefix(eventType, "invite."):
		return []string{"campaign_invites"}
	default:
		return append([]string(nil), defaultCampaignProjectionScopes...)
	}
}

func hasProjectionScopeIntersection(scopes []string, filter map[string]struct{}) bool {
	if len(scopes) == 0 {
		return false
	}
	if len(filter) == 0 {
		return true
	}
	for _, scope := range scopes {
		if _, ok := filter[scope]; ok {
			return true
		}
	}
	return false
}
