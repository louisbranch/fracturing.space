package web

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

func (h *handler) cachedUserCampaigns(ctx context.Context, userID string) ([]*statev1.Campaign, bool) {
	if h == nil || h.cacheStore == nil {
		return nil, false
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, false
	}
	if ctx == nil {
		ctx = context.Background()
	}

	entry, ok, err := h.cacheStore.GetCacheEntry(ctx, campaignListCacheKey(userID))
	if err != nil || !ok {
		return nil, false
	}
	if entry.Stale || (!entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt)) {
		_ = h.cacheStore.DeleteCacheEntry(ctx, campaignListCacheKey(userID))
		return nil, false
	}
	if len(entry.PayloadBytes) == 0 {
		return nil, false
	}

	var resp statev1.ListCampaignsResponse
	if err := proto.Unmarshal(entry.PayloadBytes, &resp); err != nil {
		_ = h.cacheStore.DeleteCacheEntry(ctx, campaignListCacheKey(userID))
		return nil, false
	}
	return resp.GetCampaigns(), true
}

func (h *handler) setUserCampaignsCache(ctx context.Context, userID string, campaigns []*statev1.Campaign) {
	if h == nil || h.cacheStore == nil {
		return
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	payload, err := proto.Marshal(&statev1.ListCampaignsResponse{Campaigns: campaigns})
	if err != nil {
		return
	}
	now := time.Now().UTC()
	_ = h.cacheStore.PutCacheEntry(ctx, webstorage.CacheEntry{
		CacheKey:     campaignListCacheKey(userID),
		Scope:        cacheScopeCampaignSummary,
		UserID:       userID,
		PayloadBytes: payload,
		Stale:        false,
		CheckedAt:    now,
		RefreshedAt:  now,
		ExpiresAt:    now.Add(campaignListCacheTTL),
	})
}

func (h *handler) cachedCampaign(ctx context.Context, campaignID string) (*statev1.Campaign, bool) {
	if h == nil || h.cacheStore == nil {
		return nil, false
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, false
	}
	if ctx == nil {
		ctx = context.Background()
	}

	key := campaignDetailCacheKey(campaignID)
	entry, ok, err := h.cacheStore.GetCacheEntry(ctx, key)
	if err != nil || !ok {
		return nil, false
	}
	if entry.Stale || (!entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt)) {
		_ = h.cacheStore.DeleteCacheEntry(ctx, key)
		return nil, false
	}
	if len(entry.PayloadBytes) == 0 {
		return nil, false
	}

	var resp statev1.GetCampaignResponse
	if err := proto.Unmarshal(entry.PayloadBytes, &resp); err != nil {
		_ = h.cacheStore.DeleteCacheEntry(ctx, key)
		return nil, false
	}
	if resp.GetCampaign() == nil {
		return nil, false
	}
	return resp.GetCampaign(), true
}

func (h *handler) setCampaignCache(ctx context.Context, campaign *statev1.Campaign) {
	if h == nil || h.cacheStore == nil || campaign == nil {
		return
	}
	campaignID := strings.TrimSpace(campaign.GetId())
	if campaignID == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	payload, err := proto.Marshal(&statev1.GetCampaignResponse{Campaign: campaign})
	if err != nil {
		return
	}
	now := time.Now().UTC()
	_ = h.cacheStore.PutCacheEntry(ctx, webstorage.CacheEntry{
		CacheKey:     campaignDetailCacheKey(campaignID),
		Scope:        cacheScopeCampaignSummary,
		CampaignID:   campaignID,
		PayloadBytes: payload,
		Stale:        false,
		CheckedAt:    now,
		RefreshedAt:  now,
		ExpiresAt:    now.Add(campaignDetailCacheTTL),
	})
}

func (h *handler) cachedCampaignParticipants(ctx context.Context, campaignID string) ([]*statev1.Participant, bool) {
	if h == nil || h.cacheStore == nil {
		return nil, false
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, false
	}
	if ctx == nil {
		ctx = context.Background()
	}

	key := campaignParticipantsCacheKey(campaignID)
	entry, ok, err := h.cacheStore.GetCacheEntry(ctx, key)
	if err != nil || !ok {
		return nil, false
	}
	if entry.Stale || (!entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt)) {
		_ = h.cacheStore.DeleteCacheEntry(ctx, key)
		return nil, false
	}
	if len(entry.PayloadBytes) == 0 {
		return nil, false
	}

	var resp statev1.ListParticipantsResponse
	if err := proto.Unmarshal(entry.PayloadBytes, &resp); err != nil {
		_ = h.cacheStore.DeleteCacheEntry(ctx, key)
		return nil, false
	}
	return resp.GetParticipants(), true
}

func (h *handler) setCampaignParticipantsCache(ctx context.Context, campaignID string, participants []*statev1.Participant) {
	if h == nil || h.cacheStore == nil {
		return
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	payload, err := proto.Marshal(&statev1.ListParticipantsResponse{Participants: participants})
	if err != nil {
		return
	}
	now := time.Now().UTC()
	_ = h.cacheStore.PutCacheEntry(ctx, webstorage.CacheEntry{
		CacheKey:     campaignParticipantsCacheKey(campaignID),
		Scope:        cacheScopeCampaignParticipants,
		CampaignID:   campaignID,
		PayloadBytes: payload,
		Stale:        false,
		CheckedAt:    now,
		RefreshedAt:  now,
		ExpiresAt:    now.Add(campaignParticipantsCacheTTL),
	})
}

func (h *handler) cachedCampaignSessions(ctx context.Context, campaignID string) ([]*statev1.Session, bool) {
	if h == nil || h.cacheStore == nil {
		return nil, false
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, false
	}
	if ctx == nil {
		ctx = context.Background()
	}

	key := campaignSessionsCacheKey(campaignID)
	entry, ok, err := h.cacheStore.GetCacheEntry(ctx, key)
	if err != nil || !ok {
		return nil, false
	}
	if entry.Stale || (!entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt)) {
		_ = h.cacheStore.DeleteCacheEntry(ctx, key)
		return nil, false
	}
	if len(entry.PayloadBytes) == 0 {
		return nil, false
	}

	var resp statev1.ListSessionsResponse
	if err := proto.Unmarshal(entry.PayloadBytes, &resp); err != nil {
		_ = h.cacheStore.DeleteCacheEntry(ctx, key)
		return nil, false
	}
	return resp.GetSessions(), true
}

func (h *handler) setCampaignSessionsCache(ctx context.Context, campaignID string, sessions []*statev1.Session) {
	if h == nil || h.cacheStore == nil {
		return
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	payload, err := proto.Marshal(&statev1.ListSessionsResponse{Sessions: sessions})
	if err != nil {
		return
	}
	now := time.Now().UTC()
	_ = h.cacheStore.PutCacheEntry(ctx, webstorage.CacheEntry{
		CacheKey:     campaignSessionsCacheKey(campaignID),
		Scope:        cacheScopeCampaignSessions,
		CampaignID:   campaignID,
		PayloadBytes: payload,
		Stale:        false,
		CheckedAt:    now,
		RefreshedAt:  now,
		ExpiresAt:    now.Add(campaignSessionsCacheTTL),
	})
}

func (h *handler) cachedCampaignCharacters(ctx context.Context, campaignID string) ([]*statev1.Character, bool) {
	if h == nil || h.cacheStore == nil {
		return nil, false
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, false
	}
	if ctx == nil {
		ctx = context.Background()
	}

	key := campaignCharactersCacheKey(campaignID)
	entry, ok, err := h.cacheStore.GetCacheEntry(ctx, key)
	if err != nil || !ok {
		return nil, false
	}
	if entry.Stale || (!entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt)) {
		_ = h.cacheStore.DeleteCacheEntry(ctx, key)
		return nil, false
	}
	if len(entry.PayloadBytes) == 0 {
		return nil, false
	}

	var resp statev1.ListCharactersResponse
	if err := proto.Unmarshal(entry.PayloadBytes, &resp); err != nil {
		_ = h.cacheStore.DeleteCacheEntry(ctx, key)
		return nil, false
	}
	return resp.GetCharacters(), true
}

func (h *handler) setCampaignCharactersCache(ctx context.Context, campaignID string, characters []*statev1.Character) {
	if h == nil || h.cacheStore == nil {
		return
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	payload, err := proto.Marshal(&statev1.ListCharactersResponse{Characters: characters})
	if err != nil {
		return
	}
	now := time.Now().UTC()
	_ = h.cacheStore.PutCacheEntry(ctx, webstorage.CacheEntry{
		CacheKey:     campaignCharactersCacheKey(campaignID),
		Scope:        cacheScopeCampaignCharacters,
		CampaignID:   campaignID,
		PayloadBytes: payload,
		Stale:        false,
		CheckedAt:    now,
		RefreshedAt:  now,
		ExpiresAt:    now.Add(campaignCharactersCacheTTL),
	})
}

func (h *handler) cachedCampaignInvites(ctx context.Context, campaignID, userID string) ([]*statev1.Invite, bool) {
	if h == nil || h.cacheStore == nil {
		return nil, false
	}
	campaignID = strings.TrimSpace(campaignID)
	userID = strings.TrimSpace(userID)
	if campaignID == "" || userID == "" {
		return nil, false
	}
	if ctx == nil {
		ctx = context.Background()
	}

	key := campaignInvitesCacheKey(campaignID, userID)
	entry, ok, err := h.cacheStore.GetCacheEntry(ctx, key)
	if err != nil || !ok {
		return nil, false
	}
	if entry.Stale || (!entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt)) {
		_ = h.cacheStore.DeleteCacheEntry(ctx, key)
		return nil, false
	}
	if len(entry.PayloadBytes) == 0 {
		return nil, false
	}

	var resp statev1.ListInvitesResponse
	if err := proto.Unmarshal(entry.PayloadBytes, &resp); err != nil {
		_ = h.cacheStore.DeleteCacheEntry(ctx, key)
		return nil, false
	}
	return resp.GetInvites(), true
}

func (h *handler) setCampaignInvitesCache(ctx context.Context, campaignID, userID string, invites []*statev1.Invite) {
	if h == nil || h.cacheStore == nil {
		return
	}
	campaignID = strings.TrimSpace(campaignID)
	userID = strings.TrimSpace(userID)
	if campaignID == "" || userID == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	payload, err := proto.Marshal(&statev1.ListInvitesResponse{Invites: invites})
	if err != nil {
		return
	}
	now := time.Now().UTC()
	_ = h.cacheStore.PutCacheEntry(ctx, webstorage.CacheEntry{
		CacheKey:     campaignInvitesCacheKey(campaignID, userID),
		Scope:        cacheScopeCampaignInvites,
		CampaignID:   campaignID,
		UserID:       userID,
		PayloadBytes: payload,
		Stale:        false,
		CheckedAt:    now,
		RefreshedAt:  now,
		ExpiresAt:    now.Add(campaignInvitesCacheTTL),
	})
}
