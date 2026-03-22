package domain

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
)

const (
	defaultPreviewLimit = 3
	maxPreviewLimit     = 10

	defaultCacheFreshTTL        = 15 * time.Second
	defaultCacheStaleTTL        = 2 * time.Minute
	sessionStartNudgeStaleAfter = 7 * 24 * time.Hour

	dependencyAuthUser          = "auth.user"
	dependencyGameCampaigns     = "game.campaigns"
	dependencyGameReadiness     = "game.readiness"
	dependencyGameInvites       = "game.invites"
	dependencyGameSessions      = "game.sessions"
	dependencySocialProfile     = "social.profile"
	dependencyNotificationsRead = "notifications.unread"
)

var (
	// ErrServiceNotConfigured indicates the userhub service is nil.
	ErrServiceNotConfigured = errors.New("userhub service is not configured")
	// ErrUserIDRequired indicates caller identity is required.
	ErrUserIDRequired = errors.New("user id is required")
	// ErrAuthGatewayNotConfigured indicates the auth dependency is missing.
	ErrAuthGatewayNotConfigured = errors.New("auth gateway is not configured")
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
	CacheFreshTTL              time.Duration
	CacheStaleTTL              time.Duration
	Clock                      func() time.Time
	CampaignDependencyObserver CampaignDependencyObserver
}

// CampaignDependencyObserver tracks which campaign IDs currently back cached
// dashboard entries. The app layer uses this to retain/release game update
// subscriptions without moving cache ownership out of the domain service.
type CampaignDependencyObserver interface {
	RetainCampaignDependency(campaignID string)
	ReleaseCampaignDependency(campaignID string)
}

// GetDashboardInput identifies one user dashboard summary request.
type GetDashboardInput struct {
	UserID               string
	Locale               string
	CampaignPreviewLimit int
	InvitePreviewLimit   int
}

// InvalidateDashboardsInput identifies cached dashboard entries to remove.
type InvalidateDashboardsInput struct {
	UserIDs     []string
	CampaignIDs []string
	Reason      string
}

// InvalidateDashboardsResult reports how many cached entries were removed.
type InvalidateDashboardsResult struct {
	InvalidatedEntries int
}

// Dashboard is the userhub at-a-glance aggregate view model.
type Dashboard struct {
	Metadata            DashboardMetadata
	User                UserSummary
	Invites             InviteSummary
	Notifications       NotificationSummary
	Campaigns           CampaignSummary
	ActiveSessions      ActiveSessionSummary
	CampaignStartNudges CampaignStartNudgeSummary
	NextActions         []DashboardAction
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
	InviteID        string
	CampaignID      string
	CampaignName    string
	ParticipantID   string
	ParticipantName string
	CreatedAt       time.Time
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
	LatestSessionAt  *time.Time
	CanManageSession bool
}

// ActiveSessionSummary captures active-session join previews for one user.
type ActiveSessionSummary struct {
	Available   bool
	ListedCount int
	HasMore     bool
	Sessions    []ActiveSessionPreview
}

// ActiveSessionPreview captures one campaign/session row for join nudges.
type ActiveSessionPreview struct {
	CampaignID   string
	CampaignName string
	SessionID    string
	SessionName  string
	StartedAt    time.Time
}

// CampaignStartNudgeSummary captures user-actionable campaign start blockers.
type CampaignStartNudgeSummary struct {
	Available   bool
	ListedCount int
	HasMore     bool
	Nudges      []CampaignStartNudge
}

// CampaignStartNudge captures one user-specific readiness blocker plus action target.
type CampaignStartNudge struct {
	CampaignID          string
	CampaignName        string
	CampaignUpdatedAt   time.Time
	BlockerCode         string
	BlockerMessage      string
	ActionKind          CampaignStartNudgeActionKind
	TargetParticipantID string
	TargetCharacterID   string
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

// CampaignStartNudgeActionKind identifies one stable dashboard CTA mapping.
type CampaignStartNudgeActionKind int

const (
	// CampaignStartNudgeActionUnspecified indicates no stable dashboard CTA exists.
	CampaignStartNudgeActionUnspecified CampaignStartNudgeActionKind = iota
	// CampaignStartNudgeActionCreateCharacter asks the user to create a character.
	CampaignStartNudgeActionCreateCharacter
	// CampaignStartNudgeActionCompleteCharacter asks the user to finish a character.
	CampaignStartNudgeActionCompleteCharacter
	// CampaignStartNudgeActionConfigureAIAgent asks the user to bind an AI agent.
	CampaignStartNudgeActionConfigureAIAgent
	// CampaignStartNudgeActionInvitePlayer asks the user to invite another player.
	CampaignStartNudgeActionInvitePlayer
	// CampaignStartNudgeActionManageParticipants asks the user to manage participant seats.
	CampaignStartNudgeActionManageParticipants
	// CampaignStartNudgeActionStartSession asks the user to start a new session.
	CampaignStartNudgeActionStartSession
)

// CampaignReadiness captures current campaign start blockers for one campaign.
type CampaignReadiness struct {
	Blockers []CampaignReadinessBlocker
}

// CampaignReadinessBlocker captures one localized blocker plus action metadata.
type CampaignReadinessBlocker struct {
	Code                string
	Message             string
	ResponsibleUserIDs  []string
	ActionKind          CampaignStartNudgeActionKind
	TargetParticipantID string
	TargetCharacterID   string
}

// UserProfile stores profile state resolved from social.
type UserProfile struct {
	Name string
}

// UserIdentity stores auth-owned account identity resolved for a user.
type UserIdentity struct {
	Username string
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

// ActiveSessionPage contains active-session previews plus pagination state.
type ActiveSessionPage struct {
	Sessions []ActiveSessionPreview
	HasMore  bool
}

// AuthGateway resolves auth-owned account identity summaries.
type AuthGateway interface {
	GetUserIdentity(ctx context.Context, userID string) (UserIdentity, error)
}

// GameGateway resolves user-scoped campaign and invite summaries.
type GameGateway interface {
	ListCampaignPreviews(ctx context.Context, userID string, limit int) (CampaignPage, error)
	ListReadinessCampaigns(ctx context.Context, userID string) ([]CampaignPreview, error)
	GetCampaignReadiness(ctx context.Context, userID, campaignID string) (CampaignReadiness, error)
	ListPendingInvitePreviews(ctx context.Context, userID string, limit int) (InvitePage, error)
	ListActiveSessionPreviews(ctx context.Context, userID string, limit int) (ActiveSessionPage, error)
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
	auth          AuthGateway
	game          GameGateway
	social        SocialGateway
	notifications NotificationsGateway
	clock         func() time.Time
	cache         *dashboardCache
}

// NewService builds a userhub service from upstream read gateways.
func NewService(auth AuthGateway, game GameGateway, social SocialGateway, notifications NotificationsGateway, cfg Config) *Service {
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	cache := newDashboardCache(cfg.CacheFreshTTL, cfg.CacheStaleTTL, cfg.CampaignDependencyObserver)
	return &Service{
		auth:          auth,
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
	if s.auth == nil {
		return Dashboard{}, ErrAuthGatewayNotConfigured
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
		ActiveSessions: ActiveSessionSummary{
			Available: true,
		},
		CampaignStartNudges: CampaignStartNudgeSummary{
			Available: true,
		},
	}
	for _, campaign := range campaignPage.Campaigns {
		if campaign.Status == CampaignStatusActive {
			result.Campaigns.ActiveCount++
		}
	}

	degradedDependencies := make([]string, 0, 4)

	identity, err := s.auth.GetUserIdentity(ctx, userID)
	if err != nil {
		if staleOK {
			return staleFallback(staleDashboard, []string{dependencyAuthUser}), nil
		}
		return Dashboard{}, &DependencyUnavailableError{Dependency: dependencyAuthUser, Err: err}
	}
	result.User.Username = strings.TrimSpace(identity.Username)
	result.User.Discoverable = strings.TrimSpace(result.User.Username) != ""

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

	activeSessionPage, err := s.game.ListActiveSessionPreviews(ctx, userID, campaignLimit)
	if err != nil {
		if staleOK {
			return staleFallback(staleDashboard, []string{dependencyGameSessions}), nil
		}
		result.ActiveSessions.Available = false
		degradedDependencies = append(degradedDependencies, dependencyGameSessions)
	} else {
		result.ActiveSessions.ListedCount = len(activeSessionPage.Sessions)
		result.ActiveSessions.HasMore = activeSessionPage.HasMore
		result.ActiveSessions.Sessions = cloneActiveSessionPreviews(activeSessionPage.Sessions)
	}

	readinessCampaigns, err := s.game.ListReadinessCampaigns(ctx, userID)
	if err != nil {
		result.CampaignStartNudges.Available = false
		degradedDependencies = append(degradedDependencies, dependencyGameReadiness)
	} else {
		nudges, nudgeHasMore, readinessErr := s.buildCampaignStartNudges(ctx, userID, campaignLimit, readinessCampaigns)
		if readinessErr != nil {
			result.CampaignStartNudges.Available = false
			degradedDependencies = append(degradedDependencies, dependencyGameReadiness)
		} else {
			result.CampaignStartNudges.ListedCount = len(nudges)
			result.CampaignStartNudges.HasMore = nudgeHasMore
			result.CampaignStartNudges.Nudges = nudges
		}
	}

	profile, err := s.social.GetUserProfile(ctx, userID)
	switch {
	case err == nil:
		result.User.ProfileAvailable = true
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
	result.User.NeedsProfileCompletion = !result.User.ProfileAvailable || strings.TrimSpace(result.User.Name) == ""

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

// InvalidateDashboards removes cached dashboard entries for the requested
// users and campaigns. It is intentionally fail-open for empty inputs.
func (s *Service) InvalidateDashboards(_ context.Context, input InvalidateDashboardsInput) (InvalidateDashboardsResult, error) {
	if s == nil {
		return InvalidateDashboardsResult{}, ErrServiceNotConfigured
	}
	if s.cache == nil {
		return InvalidateDashboardsResult{}, nil
	}
	invalidated := s.cache.invalidate(normalizeIDs(input.UserIDs), normalizeIDs(input.CampaignIDs))
	return InvalidateDashboardsResult{InvalidatedEntries: invalidated}, nil
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
	if dashboardHasContinueActiveCampaignAction(dashboard) {
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

func dashboardHasContinueActiveCampaignAction(dashboard Dashboard) bool {
	if dashboard.ActiveSessions.Available && dashboard.ActiveSessions.ListedCount > 0 {
		return true
	}
	return dashboard.Campaigns.Available && dashboard.Campaigns.ActiveCount > 0
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

func normalizeIDs(values []string) []string {
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
	result.ActiveSessions.Sessions = cloneActiveSessionPreviews(input.ActiveSessions.Sessions)
	result.CampaignStartNudges.Nudges = cloneCampaignStartNudges(input.CampaignStartNudges.Nudges)
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

// cloneActiveSessionPreviews copies active-session preview slices.
func cloneActiveSessionPreviews(input []ActiveSessionPreview) []ActiveSessionPreview {
	if len(input) == 0 {
		return nil
	}
	result := make([]ActiveSessionPreview, len(input))
	copy(result, input)
	return result
}

func cloneCampaignStartNudges(input []CampaignStartNudge) []CampaignStartNudge {
	if len(input) == 0 {
		return nil
	}
	result := make([]CampaignStartNudge, len(input))
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

func (s *Service) buildCampaignStartNudges(ctx context.Context, userID string, limit int, campaigns []CampaignPreview) ([]CampaignStartNudge, bool, error) {
	if len(campaigns) == 0 {
		return nil, false, nil
	}
	now := s.nowUTC()
	collected := make([]CampaignStartNudge, 0, len(campaigns))
	for _, campaign := range campaigns {
		readiness, err := s.game.GetCampaignReadiness(ctx, userID, campaign.CampaignID)
		if err != nil {
			return nil, false, err
		}
		blocker, ok := actionableReadinessBlockerForUser(userID, readiness.Blockers)
		if ok {
			collected = append(collected, CampaignStartNudge{
				CampaignID:          campaign.CampaignID,
				CampaignName:        campaign.Name,
				CampaignUpdatedAt:   campaign.UpdatedAt,
				BlockerCode:         blocker.Code,
				BlockerMessage:      blocker.Message,
				ActionKind:          blocker.ActionKind,
				TargetParticipantID: blocker.TargetParticipantID,
				TargetCharacterID:   blocker.TargetCharacterID,
			})
			continue
		}
		if campaignSessionStartNudgeReady(now, campaign, readiness) {
			collected = append(collected, CampaignStartNudge{
				CampaignID:        campaign.CampaignID,
				CampaignName:      campaign.Name,
				CampaignUpdatedAt: campaign.UpdatedAt,
				BlockerCode:       "START_SESSION_STALE",
				BlockerMessage:    "Start a new session for this campaign.",
				ActionKind:        CampaignStartNudgeActionStartSession,
			})
		}
	}
	if len(collected) == 0 {
		return nil, false, nil
	}
	sort.SliceStable(collected, func(i, j int) bool {
		left := collected[i].CampaignUpdatedAt.UTC()
		right := collected[j].CampaignUpdatedAt.UTC()
		if left.Equal(right) {
			return collected[i].CampaignID < collected[j].CampaignID
		}
		return left.After(right)
	})
	if limit <= 0 || len(collected) <= limit {
		return collected, false, nil
	}
	return append([]CampaignStartNudge{}, collected[:limit]...), true, nil
}

func campaignSessionStartNudgeReady(now time.Time, campaign CampaignPreview, readiness CampaignReadiness) bool {
	if len(readiness.Blockers) > 0 {
		return false
	}
	if !campaign.CanManageSession {
		return false
	}
	if campaign.LatestSessionAt == nil {
		return true
	}
	return !campaign.LatestSessionAt.UTC().After(now.UTC().Add(-sessionStartNudgeStaleAfter))
}

func actionableReadinessBlockerForUser(userID string, blockers []CampaignReadinessBlocker) (CampaignReadinessBlocker, bool) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return CampaignReadinessBlocker{}, false
	}
	for _, blocker := range blockers {
		if blocker.ActionKind == CampaignStartNudgeActionUnspecified {
			continue
		}
		if containsNormalizedString(blocker.ResponsibleUserIDs, userID) {
			return blocker, true
		}
	}
	return CampaignReadinessBlocker{}, false
}

func containsNormalizedString(values []string, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return false
	}
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), want) {
			return true
		}
	}
	return false
}
