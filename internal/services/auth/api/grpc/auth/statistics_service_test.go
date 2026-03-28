package auth

import (
	"context"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeStatisticsStore struct {
	lastSince *time.Time
	stats     storage.AuthStatistics
	err       error
}

func (f *fakeStatisticsStore) GetAuthStatistics(_ context.Context, since *time.Time) (storage.AuthStatistics, error) {
	f.lastSince = since
	return f.stats, f.err
}

func TestStatisticsServiceGetAuthStatistics(t *testing.T) {
	t.Run("rejects nil request", func(t *testing.T) {
		service := NewStatisticsService(&fakeStatisticsStore{})
		_, err := service.GetAuthStatistics(context.Background(), nil)
		grpcassert.StatusCode(t, err, codes.InvalidArgument)
	})

	t.Run("rejects missing store", func(t *testing.T) {
		service := NewStatisticsService(nil)
		_, err := service.GetAuthStatistics(context.Background(), &authv1.GetAuthStatisticsRequest{})
		grpcassert.StatusCode(t, err, codes.Internal)
	})

	t.Run("returns stats", func(t *testing.T) {
		since := time.Date(2026, 2, 1, 10, 30, 0, 0, time.UTC)
		store := &fakeStatisticsStore{
			stats: storage.AuthStatistics{UserCount: 12},
		}
		service := NewStatisticsService(store)
		resp, err := service.GetAuthStatistics(context.Background(), &authv1.GetAuthStatisticsRequest{
			Since: timestamppb.New(since),
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp.GetStats().GetUserCount() != 12 {
			t.Fatalf("expected user count 12, got %d", resp.GetStats().GetUserCount())
		}
		if store.lastSince == nil || !store.lastSince.Equal(since) {
			t.Fatalf("expected since %v, got %v", since, store.lastSince)
		}
	})
}
