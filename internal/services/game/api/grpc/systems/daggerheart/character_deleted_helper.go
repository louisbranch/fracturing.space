package daggerheart

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *DaggerheartService) appendCharacterDeletedEvent(ctx context.Context, campaignID, characterID, reason string) error {
	if err := s.requireDependencies(dependencyCampaignStore); err != nil {
		return err
	}
	payload := character.DeletePayload{
		CharacterID: ids.CharacterID(characterID),
		Reason:      strings.TrimSpace(reason),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return status.Errorf(codes.Internal, "encode payload: %v", err)
	}
	applier := s.stores.Applier()
	_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:   ids.CampaignID(campaignID),
		Type:         commandTypeCharacterDelete,
		ActorType:    command.ActorTypeSystem,
		SessionID:    ids.SessionID(grpcmeta.SessionIDFromContext(ctx)),
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		EntityType:   "character",
		EntityID:     characterID,
		PayloadJSON:  payloadJSON,
	}, applier, domainwrite.RequireEventsWithDiagnostics("character delete did not emit an event", "apply event"))
	if err != nil {
		return err
	}
	return nil
}
