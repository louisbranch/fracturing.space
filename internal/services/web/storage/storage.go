package storage

import (
	"context"
	"time"
)

// CacheEntry stores one web cache payload and freshness metadata.
//
// Cache data is always derived and can be discarded/rebuilt from upstream
// service reads.
type CacheEntry struct {
	CacheKey     string
	Scope        string
	CampaignID   string
	UserID       string
	PayloadBytes []byte
	SourceSeq    uint64
	Stale        bool
	CheckedAt    time.Time
	RefreshedAt  time.Time
	ExpiresAt    time.Time
}

// CampaignEventCursor tracks the latest known campaign event sequence.
//
// The cursor allows web cache invalidation to detect source-of-truth
// progression without re-fetching every cached scope payload first.
type CampaignEventCursor struct {
	CampaignID string
	LatestSeq  uint64
	CheckedAt  time.Time
}

// Store is the minimal contract for web cache persistence lifecycle.
//
// Individual cache read/write contracts are introduced as cache scopes are
// implemented in later milestones.
type Store interface {
	Close() error
	GetCacheEntry(ctx context.Context, cacheKey string) (CacheEntry, bool, error)
	PutCacheEntry(ctx context.Context, entry CacheEntry) error
	DeleteCacheEntry(ctx context.Context, cacheKey string) error
	ListTrackedCampaignIDs(ctx context.Context) ([]string, error)
	GetCampaignEventCursor(ctx context.Context, campaignID string) (CampaignEventCursor, bool, error)
	PutCampaignEventCursor(ctx context.Context, cursor CampaignEventCursor) error
	MarkCampaignScopeStale(ctx context.Context, campaignID, scope string, headSeq uint64, checkedAt time.Time) error
}
