package domain

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultPreviewLimit = 3
	maxPreviewLimit     = 10

	defaultCacheFreshTTL = 15 * time.Second
	defaultCacheStaleTTL = 2 * time.Minute

	dependencyGameCampaigns     = "game.campaigns"
	dependencyGameInvites       = "game.invites"
	dependencySocialProfile     = "social.profile"
	dependencyNotificationsRead = "notifications.unread"
)

var (
	// ErrServiceNotConfigured indicates the userhub service is nil.
	ErrServiceNotConfigured = errors.New("userhub service is not configured")
	// ErrUserIDRequired indicates caller identity is required.
	ErrUserIDRequired = errors.New("user id is required")
	// ErrGameGatewayNotConfigured indicates the game dependency is missing.
	ErrGameGatewayNotConfigured = errors.New("game gateway is not configured")
	// ErrSocialGatewayNotConfigured indicates the social dependency is missing.
	ErrSocialGatewayNotConfigured = errors.New("social gateway is not configured")
	// ErrNotificationsGatewayNotConfigured indicates notifications dependency is missing.
	ErrNotificationsGatewayNotConfigured = errors.New("notifications gateway is not configured")
	// ErrProfileNotFound indicates no social profile exists for the requested user.
	ErrProfileNotFound = errors.New("user profile not found")
)

// DependencyUnavailableError reports that a critical upstream dependency failed.
//
// Critical dependencies must fail closed when stale cache is unavailable so
// callers do not receive fabricated summary data.
type DependencyUnavailableError struct {
	Dependency string
	Err        error
}

// Error returns the dependency failure message.
func (e *DependencyUnavailableError) Error() string {
	if e == nil {
		return "dependency unavailable"
	}
	if strings.TrimSpace(e.Dependency) == "" {
		return "dependency unavailable"
	}
	if e.Err == nil {
		return e.Dependency + " unavailable"
	}
	return e.Dependency + " unavailable: " + e.Err.Error()
}

// Unwrap exposes the wrapped dependency failure.
func (e *DependencyUnavailableError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Config controls userhub aggregation and cache behavior.
type Config struct {
	CacheFreshTTL time.Duration
	CacheStaleTTL time.Duration
	Clock         func() time.Time
}

// GetDashboardInput identifies one user dashboard summary request.
type GetDashboardInput struct {
	UserID               string
	Locale               string
	CampaignPreviewLimit int
	InvitePreviewLimit   int
}

// Dashboard is the userhub at-a-glance aggregate view model.
type Dashboard struct {
	Metadata      DashboardMetadata
	User          UserSummary
	Invites       InviteSummary
	Notifications NotificationSummary
	Campaigns     CampaignSummary
	NextActions   []DashboardAction
}

// DashboardMetadata carries freshness and degradation status for one response.
type DashboardMetadata struct {
	Freshness            Freshness
	CacheHit             bool
	Degraded             bool
	DegradedDependencies []string
	GeneratedAt          time.Time
}

// Freshness identifies whether a dashboard response is live or stale.
type Freshness int

const (
	// FreshnessUnspecified indicates unknown freshness state.
	FreshnessUnspecified Freshness = iota
	// FreshnessFresh indicates a live or fresh-cache response.
	FreshnessFresh
	// FreshnessStale indicates a stale-cache fallback response.
	FreshnessStale
)

// UserSummary captures profile/discoverability state for one user.
type UserSummary struct {
	UserID                 string
	Username               string
	Name                   string
	ProfileAvailable       bool
	Discoverable           bool
	NeedsProfileCompletion bool
}

// InviteSummary captures pending-invite visibility for one user.
type InviteSummary struct {
	Available   bool
	ListedCount int
	HasMore     bool
	Pending     []PendingInvite
}

// PendingInvite captures one pending invite preview row.
type PendingInvite struct {
	InviteID      string
	CampaignID    string
	CampaignName  string
	ParticipantID string
	CreatedAt     time.Time
}

// NotificationSummary captures unread-notification status for one user.
type NotificationSummary struct {
	Available   bool
	HasUnread   bool
	UnreadCount int
}

// CampaignSummary captures campaign-at-a-glance preview state for one user.
type CampaignSummary struct {
	Available   bool
	ListedCount int
	ActiveCount int
	HasMore     bool
	Campaigns   []CampaignPreview
}

// CampaignPreview captures one campaign summary row.
type CampaignPreview struct {
	CampaignID       string
	Name             string
	Status           CampaignStatus
	ParticipantCount int
	CharacterCount   int
	UpdatedAt        time.Time
}

// CampaignStatus identifies lifecycle state for one campaign preview.
type CampaignStatus int

const (
	// CampaignStatusUnspecified indicates unknown campaign status.
	CampaignStatusUnspecified CampaignStatus = iota
	// CampaignStatusDraft indicates setup work in progress.
	CampaignStatusDraft
	// CampaignStatusActive indicates an active campaign.
	CampaignStatusActive
	// CampaignStatusCompleted indicates a completed campaign.
	CampaignStatusCompleted
	// CampaignStatusArchived indicates an archived campaign.
	CampaignStatusArchived
)

// DashboardAction identifies one prioritized user action.
type DashboardAction struct {
	ID       DashboardActionID
	Priority int
}

// DashboardActionID identifies one canonical action slot.
type DashboardActionID int

const (
	// DashboardActionUnspecified indicates no action.
	DashboardActionUnspecified DashboardActionID = iota
	// DashboardActionReviewPendingInvites asks the user to review pending invites.
	DashboardActionReviewPendingInvites
	// DashboardActionCompleteProfile asks the user to finish discoverability profile setup.
	DashboardActionCompleteProfile
	// DashboardActionCreateOrJoinCampaign asks the user to create or join a campaign.
	DashboardActionCreateOrJoinCampaign
	// DashboardActionContinueActiveCampaign asks the user to continue an active campaign.
	DashboardActionContinueActiveCampaign
	// DashboardActionReviewNotifications asks the user to review unread notifications.
	DashboardActionReviewNotifications
)

// UserProfile stores profile state resolved from social.
type UserProfile struct {
	Username string
	Name     string
}

// UnreadStatus stores unread-notification status resolved from notifications.
type UnreadStatus struct {
	HasUnread   bool
	UnreadCount int
}

// CampaignPage contains campaign previews plus pagination state.
type CampaignPage struct {
	Campaigns []CampaignPreview
	HasMore   bool
}

// InvitePage contains pending invite previews plus pagination state.
type InvitePage struct {
	Invites []PendingInvite
	HasMore bool
}

// GameGateway resolves user-scoped campaign and invite summaries.
type GameGateway interface {
	ListCampaignPreviews(ctx context.Context, userID string, limit int) (CampaignPage, error)
	ListPendingInvitePreviews(ctx context.Context, userID string, limit int) (InvitePage, error)
}

// SocialGateway resolves user social profile summaries.
type SocialGateway interface {
	GetUserProfile(ctx context.Context, userID string) (UserProfile, error)
}

// NotificationsGateway resolves unread notification summaries.
type NotificationsGateway interface {
	GetUnreadStatus(ctx context.Context, userID string) (UnreadStatus, error)
}

// Service orchestrates userhub dashboard aggregation and cache behavior.
type Service struct {
	game          GameGateway
	social        SocialGateway
	notifications NotificationsGateway
	clock         func() time.Time
	cache         *dashboardCache
}

// NewService builds a userhub service from upstream read gateways.
func NewService(game GameGateway, social SocialGateway, notifications NotificationsGateway, cfg Config) *Service {
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	cache := newDashboardCache(cfg.CacheFreshTTL, cfg.CacheStaleTTL)
	return &Service{
		game:          game,
		social:        social,
		notifications: notifications,
		clock:         clock,
		cache:         cache,
	}
}

// GetDashboard returns one aggregated dashboard view for a user.
func (s *Service) GetDashboard(ctx context.Context, input GetDashboardInput) (Dashboard, error) {
	if s == nil {
		return Dashboard{}, ErrServiceNotConfigured
	}
	if s.game == nil {
		return Dashboard{}, ErrGameGatewayNotConfigured
	}
	if s.social == nil {
		return Dashboard{}, ErrSocialGatewayNotConfigured
	}
	if s.notifications == nil {
		return Dashboard{}, ErrNotificationsGatewayNotConfigured
	}

	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return Dashboard{}, ErrUserIDRequired
	}

	now := s.nowUTC()
	cacheKey := dashboardCacheKey{
		UserID: userID,
		Locale: strings.TrimSpace(input.Locale),
	}

	if cached, ok := s.cache.getFresh(cacheKey, now); ok {
		cached.Metadata.CacheHit = true
		cached.Metadata.Freshness = FreshnessFresh
		return cached, nil
	}
	staleDashboard, staleOK := s.cache.getStale(cacheKey, now)

	campaignLimit := clampPreviewLimit(input.CampaignPreviewLimit)
	inviteLimit := clampPreviewLimit(input.InvitePreviewLimit)

	campaignPage, err := s.game.ListCampaignPreviews(ctx, userID, campaignLimit)
	if err != nil {
		if staleOK {
			return staleFallback(staleDashboard, []string{dependencyGameCampaigns}), nil
		}
		return Dashboard{}, &DependencyUnavailableError{Dependency: dependencyGameCampaigns, Err: err}
	}

	result := Dashboard{
		User: UserSummary{
			UserID: userID,
		},
		Invites: InviteSummary{
			Available: true,
		},
		Notifications: NotificationSummary{
			Available: true,
		},
		Campaigns: CampaignSummary{
			Available:   true,
			ListedCount: len(campaignPage.Campaigns),
			HasMore:     campaignPage.HasMore,
			Campaigns:   cloneCampaignPreviews(campaignPage.Campaigns),
		},
	}
	for _, campaign := range campaignPage.Campaigns {
		if campaign.Status == CampaignStatusActive {
			result.Campaigns.ActiveCount++
		}
	}

	degradedDependencies := make([]string, 0, 3)

	invitePage, err := s.game.ListPendingInvitePreviews(ctx, userID, inviteLimit)
	if err != nil {
		if staleOK {
			return staleFallback(staleDashboard, []string{dependencyGameInvites}), nil
		}
		result.Invites.Available = false
		degradedDependencies = append(degradedDependencies, dependencyGameInvites)
	} else {
		result.Invites.ListedCount = len(invitePage.Invites)
		result.Invites.HasMore = invitePage.HasMore
		result.Invites.Pending = clonePendingInvites(invitePage.Invites)
	}

	profile, err := s.social.GetUserProfile(ctx, userID)
	switch {
	case err == nil:
		result.User.ProfileAvailable = true
		result.User.Username = strings.TrimSpace(profile.Username)
		result.User.Name = strings.TrimSpace(profile.Name)
	case errors.Is(err, ErrProfileNotFound):
		result.User.ProfileAvailable = false
	case err != nil:
		if staleOK {
			return staleFallback(staleDashboard, []string{dependencySocialProfile}), nil
		}
		result.User.ProfileAvailable = false
		degradedDependencies = append(degradedDependencies, dependencySocialProfile)
	}
	result.User.Discoverable = strings.TrimSpace(result.User.Username) != ""
	result.User.NeedsProfileCompletion = !result.User.Discoverable

	unreadStatus, err := s.notifications.GetUnreadStatus(ctx, userID)
	if err != nil {
		if staleOK {
			return staleFallback(staleDashboard, []string{dependencyNotificationsRead}), nil
		}
		result.Notifications.Available = false
		degradedDependencies = append(degradedDependencies, dependencyNotificationsRead)
	} else {
		if unreadStatus.UnreadCount < 0 {
			unreadStatus.UnreadCount = 0
		}
		result.Notifications.HasUnread = unreadStatus.HasUnread
		result.Notifications.UnreadCount = unreadStatus.UnreadCount
	}

	result.NextActions = buildDashboardActions(result)
	result.Metadata = DashboardMetadata{
		Freshness:            FreshnessFresh,
		CacheHit:             false,
		Degraded:             len(degradedDependencies) > 0,
		DegradedDependencies: normalizeDependencies(degradedDependencies),
		GeneratedAt:          now,
	}

	if !result.Metadata.Degraded {
		s.cache.set(cacheKey, result, now)
	}

	return result, nil
}

// buildDashboardActions computes one deterministic action list from dashboard state.
func buildDashboardActions(dashboard Dashboard) []DashboardAction {
	actions := make([]DashboardAction, 0, 5)

	if dashboard.Invites.Available && dashboard.Invites.ListedCount > 0 {
		actions = append(actions, DashboardAction{
			ID:       DashboardActionReviewPendingInvites,
			Priority: 100,
		})
	}
	if dashboard.User.NeedsProfileCompletion {
		actions = append(actions, DashboardAction{
			ID:       DashboardActionCompleteProfile,
			Priority: 90,
		})
	}
	if dashboard.Campaigns.Available && dashboard.Campaigns.ListedCount == 0 && !dashboard.Campaigns.HasMore {
		actions = append(actions, DashboardAction{
			ID:       DashboardActionCreateOrJoinCampaign,
			Priority: 80,
		})
	}
	if dashboard.Campaigns.Available && dashboard.Campaigns.ActiveCount > 0 {
		actions = append(actions, DashboardAction{
			ID:       DashboardActionContinueActiveCampaign,
			Priority: 70,
		})
	}
	if dashboard.Notifications.Available && dashboard.Notifications.HasUnread {
		actions = append(actions, DashboardAction{
			ID:       DashboardActionReviewNotifications,
			Priority: 60,
		})
	}

	sort.SliceStable(actions, func(i, j int) bool {
		return actions[i].Priority > actions[j].Priority
	})
	return actions
}

// clampPreviewLimit normalizes preview limit request values.
func clampPreviewLimit(limit int) int {
	switch {
	case limit <= 0:
		return defaultPreviewLimit
	case limit > maxPreviewLimit:
		return maxPreviewLimit
	default:
		return limit
	}
}

// staleFallback returns one stale-cache dashboard with current degradation notes.
func staleFallback(stale Dashboard, degradedDependencies []string) Dashboard {
	result := cloneDashboard(stale)
	deps := append([]string{}, result.Metadata.DegradedDependencies...)
	deps = append(deps, degradedDependencies...)
	result.Metadata.CacheHit = true
	result.Metadata.Freshness = FreshnessStale
	result.Metadata.Degraded = true
	result.Metadata.DegradedDependencies = normalizeDependencies(deps)
	return result
}

// normalizeDependencies returns deterministic dependency names for responses.
func normalizeDependencies(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(set))
	for value := range set {
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	return normalized
}

// cloneDashboard protects cached state from caller mutation.
func cloneDashboard(input Dashboard) Dashboard {
	result := input
	result.Metadata.DegradedDependencies = append([]string{}, input.Metadata.DegradedDependencies...)
	result.Invites.Pending = clonePendingInvites(input.Invites.Pending)
	result.Campaigns.Campaigns = cloneCampaignPreviews(input.Campaigns.Campaigns)
	result.NextActions = append([]DashboardAction{}, input.NextActions...)
	return result
}

// clonePendingInvites copies pending invite slices.
func clonePendingInvites(input []PendingInvite) []PendingInvite {
	if len(input) == 0 {
		return nil
	}
	result := make([]PendingInvite, len(input))
	copy(result, input)
	return result
}

// cloneCampaignPreviews copies campaign preview slices.
func cloneCampaignPreviews(input []CampaignPreview) []CampaignPreview {
	if len(input) == 0 {
		return nil
	}
	result := make([]CampaignPreview, len(input))
	copy(result, input)
	return result
}

// nowUTC resolves current time for deterministic cache behavior.
func (s *Service) nowUTC() time.Time {
	if s == nil || s.clock == nil {
		return time.Now().UTC()
	}
	return s.clock().UTC()
}

// dashboardCacheKey identifies one cached dashboard snapshot.
type dashboardCacheKey struct {
	UserID string
	Locale string
}

// dashboardCacheEntry stores one cached dashboard snapshot.
type dashboardCacheEntry struct {
	Dashboard Dashboard
	CachedAt  time.Time
}

// dashboardCache provides in-memory per-user dashboard snapshots.
type dashboardCache struct {
	mu       sync.RWMutex
	freshTTL time.Duration
	staleTTL time.Duration
	entries  map[dashboardCacheKey]dashboardCacheEntry
}

// newDashboardCache creates an in-memory cache with normalized TTLs.
func newDashboardCache(freshTTL, staleTTL time.Duration) *dashboardCache {
	if freshTTL <= 0 {
		freshTTL = defaultCacheFreshTTL
	}
	if staleTTL <= 0 {
		staleTTL = defaultCacheStaleTTL
	}
	if staleTTL < freshTTL {
		staleTTL = freshTTL
	}
	return &dashboardCache{
		freshTTL: freshTTL,
		staleTTL: staleTTL,
		entries:  make(map[dashboardCacheKey]dashboardCacheEntry),
	}
}

// getFresh returns a dashboard when cache age is within fresh TTL.
func (c *dashboardCache) getFresh(key dashboardCacheKey, now time.Time) (Dashboard, bool) {
	if c == nil {
		return Dashboard{}, false
	}
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return Dashboard{}, false
	}
	age := now.Sub(entry.CachedAt)
	if age < 0 || age > c.freshTTL {
		return Dashboard{}, false
	}
	return cloneDashboard(entry.Dashboard), true
}

// getStale returns a dashboard when cache age is within stale TTL.
func (c *dashboardCache) getStale(key dashboardCacheKey, now time.Time) (Dashboard, bool) {
	if c == nil {
		return Dashboard{}, false
	}
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return Dashboard{}, false
	}
	age := now.Sub(entry.CachedAt)
	if age < 0 || age > c.staleTTL {
		return Dashboard{}, false
	}
	return cloneDashboard(entry.Dashboard), true
}

// set upserts one cached dashboard snapshot.
func (c *dashboardCache) set(key dashboardCacheKey, dashboard Dashboard, now time.Time) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.entries[key] = dashboardCacheEntry{
		Dashboard: cloneDashboard(dashboard),
		CachedAt:  now,
	}
	c.mu.Unlock()
}
