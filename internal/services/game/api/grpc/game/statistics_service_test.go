package game

import (
	"context"
	"testing"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeStatisticsStore struct {
	lastSince *time.Time
	stats     storage.GameStatistics
	err       error
}

func (f *fakeStatisticsStore) GetGameStatistics(_ context.Context, since *time.Time) (storage.GameStatistics, error) {
	f.lastSince = since
	return f.stats, f.err
}

func TestStatisticsServiceGetGameStatistics(t *testing.T) {
	t.Run("rejects nil request", func(t *testing.T) {
		service := NewStatisticsService(Stores{Statistics: &fakeStatisticsStore{}})
		_, err := service.GetGameStatistics(context.Background(), nil)
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected invalid argument, got %v", err)
		}
	})

	t.Run("rejects missing store", func(t *testing.T) {
		service := NewStatisticsService(Stores{})
		_, err := service.GetGameStatistics(context.Background(), &gamev1.GetGameStatisticsRequest{})
		if status.Code(err) != codes.Internal {
			t.Fatalf("expected internal error, got %v", err)
		}
	})

	t.Run("returns stats", func(t *testing.T) {
		since := time.Date(2025, 12, 1, 10, 30, 0, 0, time.UTC)
		store := &fakeStatisticsStore{
			stats: storage.GameStatistics{
				CampaignCount:    2,
				SessionCount:     5,
				CharacterCount:   12,
				ParticipantCount: 9,
			},
		}
		service := NewStatisticsService(Stores{Statistics: store})
		resp, err := service.GetGameStatistics(context.Background(), &gamev1.GetGameStatisticsRequest{
			Since: timestamppb.New(since),
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp.GetStats().GetCampaignCount() != 2 {
			t.Fatalf("expected campaign count 2, got %d", resp.GetStats().GetCampaignCount())
		}
		if store.lastSince == nil || !store.lastSince.Equal(since) {
			t.Fatalf("expected since %v, got %v", since, store.lastSince)
		}
	})
}
