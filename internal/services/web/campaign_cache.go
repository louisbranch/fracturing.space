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

func (h *handler) cachedPayload(ctx context.Context, cacheKey string) ([]byte, bool) {
	if h == nil || h.cacheStore == nil {
		return nil, false
	}
	ctx = normalizeCacheContext(ctx)

	entry, ok, err := h.cacheStore.GetCacheEntry(ctx, cacheKey)
	if err != nil || !ok {
		return nil, false
	}
	if entry.Stale || (!entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt)) {
		_ = h.cacheStore.DeleteCacheEntry(ctx, cacheKey)
		return nil, false
	}
	if len(entry.PayloadBytes) == 0 {
		return nil, false
	}
	return entry.PayloadBytes, true
}

func (h *handler) deleteCacheKey(ctx context.Context, cacheKey string) {
	if h == nil || h.cacheStore == nil {
		return
	}
	_ = h.cacheStore.DeleteCacheEntry(normalizeCacheContext(ctx), cacheKey)
}

func (h *handler) putCachePayload(ctx context.Context, request cacheWriteRequest) {
	if h == nil || h.cacheStore == nil || len(request.Payload) == 0 {
		return
	}
	now := time.Now().UTC()
	_ = h.cacheStore.PutCacheEntry(normalizeCacheContext(ctx), webstorage.CacheEntry{
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

func (h *handler) cachedUserCampaigns(ctx context.Context, userID string) ([]*statev1.Campaign, bool) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, false
	}
	key := campaignListCacheKey(userID)
	payload, ok := h.cachedPayload(ctx, key)
	if !ok {
		return nil, false
	}

	var resp statev1.ListCampaignsResponse
	if err := proto.Unmarshal(payload, &resp); err != nil {
		h.deleteCacheKey(ctx, key)
		return nil, false
	}
	return resp.GetCampaigns(), true
}

func (h *handler) setUserCampaignsCache(ctx context.Context, userID string, campaigns []*statev1.Campaign) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}

	payload, err := proto.Marshal(&statev1.ListCampaignsResponse{Campaigns: campaigns})
	if err != nil {
		return
	}
	h.putCachePayload(ctx, cacheWriteRequest{
		CacheKey: campaignListCacheKey(userID),
		Scope:    cacheScopeCampaignSummary,
		UserID:   userID,
		TTL:      campaignListCacheTTL,
		Payload:  payload,
	})
}

func (h *handler) expireUserCampaignsCache(ctx context.Context, userID string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}
	h.deleteCacheKey(ctx, campaignListCacheKey(userID))
}

func (h *handler) cachedCampaign(ctx context.Context, campaignID string) (*statev1.Campaign, bool) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, false
	}
	key := campaignDetailCacheKey(campaignID)
	payload, ok := h.cachedPayload(ctx, key)
	if !ok {
		return nil, false
	}

	var resp statev1.GetCampaignResponse
	if err := proto.Unmarshal(payload, &resp); err != nil {
		h.deleteCacheKey(ctx, key)
		return nil, false
	}
	if resp.GetCampaign() == nil {
		return nil, false
	}
	return resp.GetCampaign(), true
}

func (h *handler) setCampaignCache(ctx context.Context, campaign *statev1.Campaign) {
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
	h.putCachePayload(ctx, cacheWriteRequest{
		CacheKey:   campaignDetailCacheKey(campaignID),
		Scope:      cacheScopeCampaignSummary,
		CampaignID: campaignID,
		TTL:        campaignDetailCacheTTL,
		Payload:    payload,
	})
}

func (h *handler) cachedCampaignParticipants(ctx context.Context, campaignID string) ([]*statev1.Participant, bool) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, false
	}
	key := campaignParticipantsCacheKey(campaignID)
	payload, ok := h.cachedPayload(ctx, key)
	if !ok {
		return nil, false
	}

	var resp statev1.ListParticipantsResponse
	if err := proto.Unmarshal(payload, &resp); err != nil {
		h.deleteCacheKey(ctx, key)
		return nil, false
	}
	return resp.GetParticipants(), true
}

func (h *handler) setCampaignParticipantsCache(ctx context.Context, campaignID string, participants []*statev1.Participant) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return
	}

	payload, err := proto.Marshal(&statev1.ListParticipantsResponse{Participants: participants})
	if err != nil {
		return
	}
	h.putCachePayload(ctx, cacheWriteRequest{
		CacheKey:   campaignParticipantsCacheKey(campaignID),
		Scope:      cacheScopeCampaignParticipants,
		CampaignID: campaignID,
		TTL:        campaignParticipantsCacheTTL,
		Payload:    payload,
	})
}

func (h *handler) cachedCampaignSessions(ctx context.Context, campaignID string) ([]*statev1.Session, bool) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, false
	}
	key := campaignSessionsCacheKey(campaignID)
	payload, ok := h.cachedPayload(ctx, key)
	if !ok {
		return nil, false
	}

	var resp statev1.ListSessionsResponse
	if err := proto.Unmarshal(payload, &resp); err != nil {
		h.deleteCacheKey(ctx, key)
		return nil, false
	}
	return resp.GetSessions(), true
}

func (h *handler) setCampaignSessionsCache(ctx context.Context, campaignID string, sessions []*statev1.Session) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return
	}

	payload, err := proto.Marshal(&statev1.ListSessionsResponse{Sessions: sessions})
	if err != nil {
		return
	}
	h.putCachePayload(ctx, cacheWriteRequest{
		CacheKey:   campaignSessionsCacheKey(campaignID),
		Scope:      cacheScopeCampaignSessions,
		CampaignID: campaignID,
		TTL:        campaignSessionsCacheTTL,
		Payload:    payload,
	})
}

func (h *handler) cachedCampaignCharacters(ctx context.Context, campaignID string) ([]*statev1.Character, bool) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, false
	}
	key := campaignCharactersCacheKey(campaignID)
	payload, ok := h.cachedPayload(ctx, key)
	if !ok {
		return nil, false
	}

	var resp statev1.ListCharactersResponse
	if err := proto.Unmarshal(payload, &resp); err != nil {
		h.deleteCacheKey(ctx, key)
		return nil, false
	}
	return resp.GetCharacters(), true
}

func (h *handler) setCampaignCharactersCache(ctx context.Context, campaignID string, characters []*statev1.Character) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return
	}

	payload, err := proto.Marshal(&statev1.ListCharactersResponse{Characters: characters})
	if err != nil {
		return
	}
	h.putCachePayload(ctx, cacheWriteRequest{
		CacheKey:   campaignCharactersCacheKey(campaignID),
		Scope:      cacheScopeCampaignCharacters,
		CampaignID: campaignID,
		TTL:        campaignCharactersCacheTTL,
		Payload:    payload,
	})
}

func (h *handler) cachedCampaignInvites(ctx context.Context, campaignID, userID string) ([]*statev1.Invite, bool) {
	campaignID = strings.TrimSpace(campaignID)
	userID = strings.TrimSpace(userID)
	if campaignID == "" || userID == "" {
		return nil, false
	}
	key := campaignInvitesCacheKey(campaignID, userID)
	payload, ok := h.cachedPayload(ctx, key)
	if !ok {
		return nil, false
	}

	var resp statev1.ListInvitesResponse
	if err := proto.Unmarshal(payload, &resp); err != nil {
		h.deleteCacheKey(ctx, key)
		return nil, false
	}
	return resp.GetInvites(), true
}

func (h *handler) setCampaignInvitesCache(ctx context.Context, campaignID, userID string, invites []*statev1.Invite) {
	campaignID = strings.TrimSpace(campaignID)
	userID = strings.TrimSpace(userID)
	if campaignID == "" || userID == "" {
		return
	}

	payload, err := proto.Marshal(&statev1.ListInvitesResponse{Invites: invites})
	if err != nil {
		return
	}
	h.putCachePayload(ctx, cacheWriteRequest{
		CacheKey:   campaignInvitesCacheKey(campaignID, userID),
		Scope:      cacheScopeCampaignInvites,
		CampaignID: campaignID,
		UserID:     userID,
		TTL:        campaignInvitesCacheTTL,
		Payload:    payload,
	})
}
