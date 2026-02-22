package daggerheart

import (
	"context"
	"encoding/json"
	"strings"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *DaggerheartService) appendCharacterDeletedEvent(ctx context.Context, campaignID, characterID, reason string) error {
	if s.stores.Campaign == nil {
		return status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Domain == nil {
		return status.Error(codes.Internal, "domain engine is not configured")
	}
	payload := character.DeletePayload{
		CharacterID: characterID,
		Reason:      strings.TrimSpace(reason),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return status.Errorf(codes.Internal, "encode payload: %v", err)
	}
	applier := s.stores.Applier()
	_, err = s.executeAndApplyDomainCommand(ctx, command.Command{
		CampaignID:   campaignID,
		Type:         commandTypeCharacterDelete,
		ActorType:    command.ActorTypeSystem,
		SessionID:    grpcmeta.SessionIDFromContext(ctx),
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		EntityType:   "character",
		EntityID:     characterID,
		PayloadJSON:  payloadJSON,
	}, applier, domainCommandApplyOptions{
		requireEvents:   true,
		missingEventMsg: "character delete did not emit an event",
		applyErrMessage: "apply event",
		executeErrMsg:   "execute domain command",
	})
	if err != nil {
		return err
	}
	return nil
}
