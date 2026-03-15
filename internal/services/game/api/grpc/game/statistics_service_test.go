package game

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestStatisticsServiceGetGameStatistics(t *testing.T) {
	t.Run("rejects nil request", func(t *testing.T) {
		service := NewStatisticsService(&gametest.FakeStatisticsStore{})
		_, err := service.GetGameStatistics(context.Background(), nil)
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected invalid argument, got %v", err)
		}
	})

	t.Run("returns stats", func(t *testing.T) {
		since := time.Date(2025, 12, 1, 10, 30, 0, 0, time.UTC)
		store := &gametest.FakeStatisticsStore{
			Stats: storage.GameStatistics{
				CampaignCount:    2,
				SessionCount:     5,
				CharacterCount:   12,
				ParticipantCount: 9,
			},
		}
		service := NewStatisticsService(store)
		resp, err := service.GetGameStatistics(context.Background(), &gamev1.GetGameStatisticsRequest{
			Since: timestamppb.New(since),
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp.GetStats().GetCampaignCount() != 2 {
			t.Fatalf("expected campaign count 2, got %d", resp.GetStats().GetCampaignCount())
		}
		if store.LastSince == nil || !store.LastSince.Equal(since) {
			t.Fatalf("expected since %v, got %v", since, store.LastSince)
		}
	})
}
