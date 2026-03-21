package environmenttransport

import (
	"context"
	"errors"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func environmentEntityToProto(environmentEntity projectionstore.DaggerheartEnvironmentEntity) *pb.DaggerheartEnvironmentEntity {
	return &pb.DaggerheartEnvironmentEntity{
		Id:            environmentEntity.EnvironmentEntityID,
		CampaignId:    environmentEntity.CampaignID,
		EnvironmentId: environmentEntity.EnvironmentID,
		Name:          environmentEntity.Name,
		Type:          environmentEntity.Type,
		Tier:          int32(environmentEntity.Tier),
		Difficulty:    int32(environmentEntity.Difficulty),
		SessionId:     environmentEntity.SessionID,
		SceneId:       environmentEntity.SceneID,
		Notes:         environmentEntity.Notes,
		CreatedAt:     timestamppb.New(environmentEntity.CreatedAt),
		UpdatedAt:     timestamppb.New(environmentEntity.UpdatedAt),
	}
}

func loadEnvironmentEntityForSession(ctx context.Context, store DaggerheartStore, campaignID, sessionID, environmentEntityID string) (projectionstore.DaggerheartEnvironmentEntity, error) {
	if store == nil {
		return projectionstore.DaggerheartEnvironmentEntity{}, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	environmentEntity, err := store.GetDaggerheartEnvironmentEntity(ctx, campaignID, environmentEntityID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return projectionstore.DaggerheartEnvironmentEntity{}, status.Error(codes.NotFound, "environment entity not found")
		}
		return projectionstore.DaggerheartEnvironmentEntity{}, grpcerror.Internal("load environment entity", err)
	}
	if environmentEntity.SessionID != "" && environmentEntity.SessionID != sessionID {
		return projectionstore.DaggerheartEnvironmentEntity{}, status.Error(codes.FailedPrecondition, "environment entity is not in session")
	}
	return environmentEntity, nil
}
