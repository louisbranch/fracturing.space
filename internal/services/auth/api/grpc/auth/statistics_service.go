package auth

import (
	"context"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StatisticsService exposes auth-level aggregates for operator and health-oriented views.
type StatisticsService struct {
	authv1.UnimplementedStatisticsServiceServer
	store storage.StatisticsStore
}

// NewStatisticsService builds the statistics facade from a statistics store.
func NewStatisticsService(store storage.StatisticsStore) *StatisticsService {
	return &StatisticsService{store: store}
}

// GetAuthStatistics returns aggregate auth statistics.
func (s *StatisticsService) GetAuthStatistics(ctx context.Context, in *authv1.GetAuthStatisticsRequest) (*authv1.GetAuthStatisticsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get auth statistics request is required")
	}
	if s.store == nil {
		return nil, status.Error(codes.Internal, "statistics store is not configured")
	}

	var since *time.Time
	if ts := in.GetSince(); ts != nil {
		value := ts.AsTime().UTC()
		since = &value
	}

	stats, err := s.store.GetAuthStatistics(ctx, since)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get auth statistics: %v", err)
	}

	return &authv1.GetAuthStatisticsResponse{
		Stats: &authv1.AuthStatistics{UserCount: stats.UserCount},
	}, nil
}
