package gateway

import (
	"context"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// CampaignGameSurface returns the game-owned communication context mapped for the web game surface.
func (g gameReadGateway) CampaignGameSurface(ctx context.Context, campaignID string) (campaignapp.CampaignGameSurface, error) {
	if g.read.Communication == nil {
		return campaignapp.CampaignGameSurface{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.campaign_service_client_is_not_configured", "communication service client is not configured")
	}
	resp, err := g.read.Communication.GetCommunicationContext(ctx, &statev1.GetCommunicationContextRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		return campaignapp.CampaignGameSurface{}, err
	}
	contextState := resp.GetContext()
	if contextState == nil {
		return campaignapp.CampaignGameSurface{}, apperrors.E(apperrors.KindNotFound, "communication context not found")
	}

	surface := campaignapp.CampaignGameSurface{
		Participant: campaignapp.CampaignGameParticipant{
			ID:   strings.TrimSpace(contextState.GetParticipant().GetParticipantId()),
			Name: strings.TrimSpace(contextState.GetParticipant().GetName()),
			Role: participantRoleLabel(contextState.GetParticipant().GetRole()),
		},
		DefaultStreamID:  strings.TrimSpace(contextState.GetDefaultStreamId()),
		DefaultPersonaID: strings.TrimSpace(contextState.GetDefaultPersonaId()),
		Streams:          make([]campaignapp.CampaignGameStream, 0, len(contextState.GetStreams())),
		Personas:         make([]campaignapp.CampaignGamePersona, 0, len(contextState.GetPersonas())),
	}
	if sessionState := contextState.GetActiveSession(); sessionState != nil {
		surface.SessionID = strings.TrimSpace(sessionState.GetSessionId())
		surface.SessionName = strings.TrimSpace(sessionState.GetName())
	}
	if gate := contextState.GetActiveSessionGate(); gate != nil {
		surface.ActiveSessionGate = &campaignapp.CampaignGameGate{
			ID:       strings.TrimSpace(gate.GetId()),
			Type:     strings.TrimSpace(gate.GetType()),
			Status:   communicationGateStatusLabel(gate.GetStatus()),
			Reason:   strings.TrimSpace(gate.GetReason()),
			Metadata: structpbMap(gate.GetMetadata()),
			Progress: structpbMap(gate.GetProgress()),
		}
	}
	if spotlight := contextState.GetActiveSessionSpotlight(); spotlight != nil {
		surface.ActiveSessionSpotlight = &campaignapp.CampaignGameSpotlight{
			Type:        communicationSpotlightTypeLabel(spotlight.GetType()),
			CharacterID: strings.TrimSpace(spotlight.GetCharacterId()),
		}
	}
	for _, stream := range contextState.GetStreams() {
		if stream == nil {
			continue
		}
		surface.Streams = append(surface.Streams, campaignapp.CampaignGameStream{
			ID:        strings.TrimSpace(stream.GetStreamId()),
			Kind:      communicationStreamKindLabel(stream.GetKind()),
			Scope:     communicationStreamScopeLabel(stream.GetScope()),
			SessionID: strings.TrimSpace(stream.GetSessionId()),
			SceneID:   strings.TrimSpace(stream.GetSceneId()),
			Label:     strings.TrimSpace(stream.GetLabel()),
		})
	}
	for _, persona := range contextState.GetPersonas() {
		if persona == nil {
			continue
		}
		surface.Personas = append(surface.Personas, campaignapp.CampaignGamePersona{
			ID:            strings.TrimSpace(persona.GetPersonaId()),
			Kind:          communicationPersonaKindLabel(persona.GetKind()),
			ParticipantID: strings.TrimSpace(persona.GetParticipantId()),
			CharacterID:   strings.TrimSpace(persona.GetCharacterId()),
			DisplayName:   strings.TrimSpace(persona.GetDisplayName()),
		})
	}
	return surface, nil
}
