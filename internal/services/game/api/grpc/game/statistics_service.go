package game

import (
	"context"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StatisticsService implements the game.v1.StatisticsService gRPC API.
type StatisticsService struct {
	gamev1.UnimplementedStatisticsServiceServer
	statistics storage.StatisticsStore
}

// NewStatisticsService creates a StatisticsService with the provided statistics store.
func NewStatisticsService(statistics storage.StatisticsStore) *StatisticsService {
	return &StatisticsService{statistics: statistics}
}

// GetGameStatistics returns aggregate game statistics.
func (s *StatisticsService) GetGameStatistics(ctx context.Context, in *gamev1.GetGameStatisticsRequest) (*gamev1.GetGameStatisticsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get game statistics request is required")
	}
	var since *time.Time
	if ts := in.GetSince(); ts != nil {
		value := ts.AsTime().UTC()
		since = &value
	}

	stats, err := s.statistics.GetGameStatistics(ctx, since)
	if err != nil {
		return nil, grpcerror.Internal("get game statistics", err)
	}

	return &gamev1.GetGameStatisticsResponse{
		Stats: &gamev1.GameStatistics{
			CampaignCount:    stats.CampaignCount,
			SessionCount:     stats.SessionCount,
			CharacterCount:   stats.CharacterCount,
			ParticipantCount: stats.ParticipantCount,
		},
	}, nil
}
