package campaign

import (
	"context"
	"strings"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	webstorage "github.com/louisbranch/fracturing.space/internal/services/web/storage"
	"google.golang.org/protobuf/proto"
)

const (
	cacheScopeCampaignSummary      = "campaign_summary"
	cacheScopeCampaignParticipants = "campaign_participants"
	cacheScopeCampaignSessions     = "campaign_sessions"
	cacheScopeCampaignCharacters   = "campaign_characters"
	cacheScopeCampaignInvites      = "campaign_invites"
	campaignListCacheTTL           = 30 * time.Second
	campaignDetailCacheTTL         = 5 * time.Minute
	campaignParticipantsCacheTTL   = 30 * time.Second
	campaignSessionsCacheTTL       = 30 * time.Second
	campaignCharactersCacheTTL     = 30 * time.Second
	campaignInvitesCacheTTL        = 30 * time.Second
)

// CampaignCache owns campaign cache read/write operations for web handler concerns.
type CampaignCache struct {
	store webstorage.Store
}

// NewCampaignCache constructs campaign cache helpers for a given web storage layer.
func NewCampaignCache(store webstorage.Store) *CampaignCache {
	return &CampaignCache{store: store}
}

func campaignListCacheKey(userID string) string {
	return "campaign_list:user:" + strings.TrimSpace(userID)
}

func campaignDetailCacheKey(campaignID string) string {
	return "campaign_detail:id:" + strings.TrimSpace(campaignID)
}

func campaignParticipantsCacheKey(campaignID string) string {
	return "campaign_participants:id:" + strings.TrimSpace(campaignID)
}

func campaignSessionsCacheKey(campaignID string) string {
	return "campaign_sessions:id:" + strings.TrimSpace(campaignID)
}

func campaignCharactersCacheKey(campaignID string) string {
	return "campaign_characters:id:" + strings.TrimSpace(campaignID)
}

func campaignInvitesCacheKey(campaignID, userID string) string {
	return "campaign_invites:id:" + strings.TrimSpace(campaignID) + ":user:" + strings.TrimSpace(userID)
}

type cacheWriteRequest struct {
	CacheKey   string
	Scope      string
	CampaignID string
	UserID     string
	TTL        time.Duration
	Payload    []byte
}

func normalizeCacheContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func (c *CampaignCache) cachedPayload(ctx context.Context, cacheKey string) ([]byte, bool) {
	if c == nil || c.store == nil {
		return nil, false
	}
	ctx = normalizeCacheContext(ctx)

	entry, ok, err := c.store.GetCacheEntry(ctx, cacheKey)
	if err != nil || !ok {
		return nil, false
	}
	if entry.Stale || (!entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt)) {
		_ = c.store.DeleteCacheEntry(ctx, cacheKey)
		return nil, false
	}
	if len(entry.PayloadBytes) == 0 {
		return nil, false
	}
	return entry.PayloadBytes, true
}

func (c *CampaignCache) deleteCacheKey(ctx context.Context, cacheKey string) {
	if c == nil || c.store == nil {
		return
	}
	_ = c.store.DeleteCacheEntry(normalizeCacheContext(ctx), cacheKey)
}

func (c *CampaignCache) putCachePayload(ctx context.Context, request cacheWriteRequest) {
	if c == nil || c.store == nil || len(request.Payload) == 0 {
		return
	}
	now := time.Now().UTC()
	_ = c.store.PutCacheEntry(normalizeCacheContext(ctx), webstorage.CacheEntry{
		CacheKey:     request.CacheKey,
		Scope:        request.Scope,
		CampaignID:   request.CampaignID,
		UserID:       request.UserID,
		PayloadBytes: request.Payload,
		Stale:        false,
		CheckedAt:    now,
		RefreshedAt:  now,
		ExpiresAt:    now.Add(request.TTL),
	})
}

// CachedUserCampaigns returns cached campaigns for the given user.
func (c *CampaignCache) CachedUserCampaigns(ctx context.Context, userID string) ([]*statev1.Campaign, bool) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, false
	}
	key := campaignListCacheKey(userID)
	payload, ok := c.cachedPayload(ctx, key)
	if !ok {
		return nil, false
	}

	var resp statev1.ListCampaignsResponse
	if err := proto.Unmarshal(payload, &resp); err != nil {
		c.deleteCacheKey(ctx, key)
		return nil, false
	}
	return resp.GetCampaigns(), true
}

// SetUserCampaignsCache stores campaign list payload for the given user.
func (c *CampaignCache) SetUserCampaignsCache(ctx context.Context, userID string, campaigns []*statev1.Campaign) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}

	payload, err := proto.Marshal(&statev1.ListCampaignsResponse{Campaigns: campaigns})
	if err != nil {
		return
	}
	c.putCachePayload(ctx, cacheWriteRequest{
		CacheKey: campaignListCacheKey(userID),
		Scope:    cacheScopeCampaignSummary,
		UserID:   userID,
		TTL:      campaignListCacheTTL,
		Payload:  payload,
	})
}

// ExpireUserCampaignsCache removes user campaign summary cache.
func (c *CampaignCache) ExpireUserCampaignsCache(ctx context.Context, userID string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}
	c.deleteCacheKey(ctx, campaignListCacheKey(userID))
}

// CachedCampaign returns cached campaign detail.
func (c *CampaignCache) CachedCampaign(ctx context.Context, campaignID string) (*statev1.Campaign, bool) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, false
	}
	key := campaignDetailCacheKey(campaignID)
	payload, ok := c.cachedPayload(ctx, key)
	if !ok {
		return nil, false
	}

	var resp statev1.GetCampaignResponse
	if err := proto.Unmarshal(payload, &resp); err != nil {
		c.deleteCacheKey(ctx, key)
		return nil, false
	}
	if resp.GetCampaign() == nil {
		return nil, false
	}
	return resp.GetCampaign(), true
}

// SetCampaignCache stores campaign detail.
func (c *CampaignCache) SetCampaignCache(ctx context.Context, campaign *statev1.Campaign) {
	if campaign == nil {
		return
	}
	campaignID := strings.TrimSpace(campaign.GetId())
	if campaignID == "" {
		return
	}

	payload, err := proto.Marshal(&statev1.GetCampaignResponse{Campaign: campaign})
	if err != nil {
		return
	}
	c.putCachePayload(ctx, cacheWriteRequest{
		CacheKey:   campaignDetailCacheKey(campaignID),
		Scope:      cacheScopeCampaignSummary,
		CampaignID: campaignID,
		TTL:        campaignDetailCacheTTL,
		Payload:    payload,
	})
}

// CachedCampaignParticipants returns cached participants for the given campaign.
func (c *CampaignCache) CachedCampaignParticipants(ctx context.Context, campaignID string) ([]*statev1.Participant, bool) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, false
	}
	key := campaignParticipantsCacheKey(campaignID)
	payload, ok := c.cachedPayload(ctx, key)
	if !ok {
		return nil, false
	}

	var resp statev1.ListParticipantsResponse
	if err := proto.Unmarshal(payload, &resp); err != nil {
		c.deleteCacheKey(ctx, key)
		return nil, false
	}
	return resp.GetParticipants(), true
}

// SetCampaignParticipantsCache stores campaign participants.
func (c *CampaignCache) SetCampaignParticipantsCache(ctx context.Context, campaignID string, participants []*statev1.Participant) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return
	}

	payload, err := proto.Marshal(&statev1.ListParticipantsResponse{Participants: participants})
	if err != nil {
		return
	}
	c.putCachePayload(ctx, cacheWriteRequest{
		CacheKey:   campaignParticipantsCacheKey(campaignID),
		Scope:      cacheScopeCampaignParticipants,
		CampaignID: campaignID,
		TTL:        campaignParticipantsCacheTTL,
		Payload:    payload,
	})
}

// CachedCampaignSessions returns cached sessions for the given campaign.
func (c *CampaignCache) CachedCampaignSessions(ctx context.Context, campaignID string) ([]*statev1.Session, bool) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, false
	}
	key := campaignSessionsCacheKey(campaignID)
	payload, ok := c.cachedPayload(ctx, key)
	if !ok {
		return nil, false
	}

	var resp statev1.ListSessionsResponse
	if err := proto.Unmarshal(payload, &resp); err != nil {
		c.deleteCacheKey(ctx, key)
		return nil, false
	}
	return resp.GetSessions(), true
}

// SetCampaignSessionsCache stores campaign sessions.
func (c *CampaignCache) SetCampaignSessionsCache(ctx context.Context, campaignID string, sessions []*statev1.Session) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return
	}

	payload, err := proto.Marshal(&statev1.ListSessionsResponse{Sessions: sessions})
	if err != nil {
		return
	}
	c.putCachePayload(ctx, cacheWriteRequest{
		CacheKey:   campaignSessionsCacheKey(campaignID),
		Scope:      cacheScopeCampaignSessions,
		CampaignID: campaignID,
		TTL:        campaignSessionsCacheTTL,
		Payload:    payload,
	})
}

// CachedCampaignCharacters returns cached characters for the given campaign.
func (c *CampaignCache) CachedCampaignCharacters(ctx context.Context, campaignID string) ([]*statev1.Character, bool) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, false
	}
	key := campaignCharactersCacheKey(campaignID)
	payload, ok := c.cachedPayload(ctx, key)
	if !ok {
		return nil, false
	}

	var resp statev1.ListCharactersResponse
	if err := proto.Unmarshal(payload, &resp); err != nil {
		c.deleteCacheKey(ctx, key)
		return nil, false
	}
	return resp.GetCharacters(), true
}

// SetCampaignCharactersCache stores campaign characters.
func (c *CampaignCache) SetCampaignCharactersCache(ctx context.Context, campaignID string, characters []*statev1.Character) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return
	}

	payload, err := proto.Marshal(&statev1.ListCharactersResponse{Characters: characters})
	if err != nil {
		return
	}
	c.putCachePayload(ctx, cacheWriteRequest{
		CacheKey:   campaignCharactersCacheKey(campaignID),
		Scope:      cacheScopeCampaignCharacters,
		CampaignID: campaignID,
		TTL:        campaignCharactersCacheTTL,
		Payload:    payload,
	})
}

// CachedCampaignInvites returns cached campaign invites scoped by user.
func (c *CampaignCache) CachedCampaignInvites(ctx context.Context, campaignID, userID string) ([]*statev1.Invite, bool) {
	campaignID = strings.TrimSpace(campaignID)
	userID = strings.TrimSpace(userID)
	if campaignID == "" || userID == "" {
		return nil, false
	}
	key := campaignInvitesCacheKey(campaignID, userID)
	payload, ok := c.cachedPayload(ctx, key)
	if !ok {
		return nil, false
	}

	var resp statev1.ListInvitesResponse
	if err := proto.Unmarshal(payload, &resp); err != nil {
		c.deleteCacheKey(ctx, key)
		return nil, false
	}
	return resp.GetInvites(), true
}

// SetCampaignInvitesCache stores campaign invites for the requesting user.
func (c *CampaignCache) SetCampaignInvitesCache(ctx context.Context, campaignID, userID string, invites []*statev1.Invite) {
	campaignID = strings.TrimSpace(campaignID)
	userID = strings.TrimSpace(userID)
	if campaignID == "" || userID == "" {
		return
	}

	payload, err := proto.Marshal(&statev1.ListInvitesResponse{Invites: invites})
	if err != nil {
		return
	}
	c.putCachePayload(ctx, cacheWriteRequest{
		CacheKey:   campaignInvitesCacheKey(campaignID, userID),
		Scope:      cacheScopeCampaignInvites,
		CampaignID: campaignID,
		UserID:     userID,
		TTL:        campaignInvitesCacheTTL,
		Payload:    payload,
	})
}
