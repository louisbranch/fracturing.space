package campaign

import (
	"context"
	"net/http"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type AppCampaignDependencies struct {
	EnsureCampaignClients   func(context.Context) error
	CampaignClientReady     func() bool
	SessionClientReady      func() bool
	ParticipantClientReady  func() bool
	CharacterClientReady    func() bool
	InviteClientReady       func() bool
	CanManageCampaignAccess func(statev1.CampaignAccess) bool

	CampaignSessionPresent      func(http.ResponseWriter, *http.Request) bool
	CampaignListUserID          func(*http.Request) (string, bool)
	CampaignReadContext         func(http.ResponseWriter, *http.Request, string) (context.Context, string, bool)
	RequireCampaignActor        func(http.ResponseWriter, *http.Request, string) (*statev1.Participant, bool)
	CampaignParticipantByUserID func(context.Context, string, string) (*statev1.Participant, error)

	PageContext            func(http.ResponseWriter, *http.Request) templates.PageContext
	PageContextForCampaign func(http.ResponseWriter, *http.Request, string) templates.PageContext
	SessionDisplayName     func(*http.Request) string

	ListCampaigns     func(context.Context, *statev1.ListCampaignsRequest) (*statev1.ListCampaignsResponse, error)
	CreateCampaign    func(context.Context, *statev1.CreateCampaignRequest) (*statev1.CreateCampaignResponse, error)
	GetCampaign       func(context.Context, *statev1.GetCampaignRequest) (*statev1.GetCampaignResponse, error)
	ListSessions      func(context.Context, *statev1.ListSessionsRequest) (*statev1.ListSessionsResponse, error)
	GetSession        func(context.Context, *statev1.GetSessionRequest) (*statev1.GetSessionResponse, error)
	StartSession      func(context.Context, *statev1.StartSessionRequest) (*statev1.StartSessionResponse, error)
	EndSession        func(context.Context, *statev1.EndSessionRequest) (*statev1.EndSessionResponse, error)
	ListParticipants  func(context.Context, *statev1.ListParticipantsRequest) (*statev1.ListParticipantsResponse, error)
	UpdateParticipant func(context.Context, *statev1.UpdateParticipantRequest) (*statev1.UpdateParticipantResponse, error)
	ListCharacters    func(context.Context, *statev1.ListCharactersRequest) (*statev1.ListCharactersResponse, error)
	GetCharacterSheet func(context.Context, *statev1.GetCharacterSheetRequest) (*statev1.GetCharacterSheetResponse, error)
	CreateCharacter   func(context.Context, *statev1.CreateCharacterRequest) (*statev1.CreateCharacterResponse, error)
	UpdateCharacter   func(context.Context, *statev1.UpdateCharacterRequest) (*statev1.UpdateCharacterResponse, error)
	SetDefaultControl func(context.Context, *statev1.SetDefaultControlRequest) (*statev1.SetDefaultControlResponse, error)
	ListInvites       func(context.Context, *statev1.ListInvitesRequest) (*statev1.ListInvitesResponse, error)
	CreateInvite      func(context.Context, *statev1.CreateInviteRequest) (*statev1.CreateInviteResponse, error)
	RevokeInvite      func(context.Context, *statev1.RevokeInviteRequest) (*statev1.RevokeInviteResponse, error)

	CachedUserCampaigns          func(context.Context, string) ([]*statev1.Campaign, bool)
	SetUserCampaignsCache        func(context.Context, string, []*statev1.Campaign)
	ExpireUserCampaignsCache     func(context.Context, string)
	SetCampaignCache             func(context.Context, *statev1.Campaign)
	CachedCampaignSessions       func(context.Context, string) ([]*statev1.Session, bool)
	SetCampaignSessionsCache     func(context.Context, string, []*statev1.Session)
	CachedCampaignParticipants   func(context.Context, string) ([]*statev1.Participant, bool)
	SetCampaignParticipantsCache func(context.Context, string, []*statev1.Participant)
	CachedCampaignCharacters     func(context.Context, string) ([]*statev1.Character, bool)
	SetCampaignCharactersCache   func(context.Context, string, []*statev1.Character)
	CachedCampaignInvites        func(context.Context, string, string) ([]*statev1.Invite, bool)
	SetCampaignInvitesCache      func(context.Context, string, string, []*statev1.Invite)

	ListInviteContactOptions          func(context.Context, string, string, []*statev1.Invite) []templates.CampaignInviteContactOption
	LookupInviteRecipientVerification func(context.Context, string) (templates.CampaignInviteVerification, error)
	ResolveInviteRecipientUserID      func(context.Context, string) (string, error)
	RenderInviteRecipientLookupError  func(http.ResponseWriter, *http.Request, error)

	LocalizeError   func(http.ResponseWriter, *http.Request, int, string, ...any)
	RenderErrorPage func(http.ResponseWriter, *http.Request, int, string, string)
	GRPCErrorStatus func(error, int) int

	RenderCampaignsListPage               func(http.ResponseWriter, *http.Request, templates.PageContext, []*statev1.Campaign)
	RenderCampaignCreatePage              func(http.ResponseWriter, *http.Request, templates.PageContext)
	RenderCampaignPage                    func(http.ResponseWriter, *http.Request, string)
	RenderCampaignSessionsPage            func(http.ResponseWriter, *http.Request, templates.PageContext, string, []*statev1.Session, bool)
	RenderCampaignSessionDetailPage       func(http.ResponseWriter, *http.Request, templates.PageContext, string, *statev1.Session)
	RenderCampaignParticipantsPage        func(http.ResponseWriter, *http.Request, templates.PageContext, string, []*statev1.Participant, bool)
	RenderCampaignCharactersPage          func(http.ResponseWriter, *http.Request, templates.PageContext, string, []*statev1.Character, bool, []*statev1.Participant)
	RenderCampaignCharacterDetailPage     func(http.ResponseWriter, *http.Request, templates.PageContext, string, *statev1.Character)
	RenderCampaignInvitesPage             func(http.ResponseWriter, *http.Request, templates.PageContext, string, []*statev1.Invite, []templates.CampaignInviteContactOption, bool)
	RenderCampaignInvitesVerificationPage func(http.ResponseWriter, *http.Request, string, string, bool, templates.CampaignInviteVerification)
}

func HandleAppCampaigns(d AppCampaignDependencies, w http.ResponseWriter, r *http.Request) {
	if d.LocalizeError == nil {
		http.NotFound(w, r)
		return
	}
	if d.CampaignSessionPresent == nil || d.CampaignListUserID == nil || d.PageContext == nil || d.RenderCampaignsListPage == nil || d.ListCampaigns == nil || d.CachedUserCampaigns == nil || d.SetUserCampaignsCache == nil || d.EnsureCampaignClients == nil || d.CampaignClientReady == nil {
		http.NotFound(w, r)
		return
	}
	if !d.CampaignSessionPresent(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	readCtx := r.Context()
	userID, ok := d.CampaignListUserID(r)
	if !ok || strings.TrimSpace(userID) == "" {
		d.RenderCampaignsListPage(w, r, d.PageContext(w, r), nil)
		return
	}

	if err := d.EnsureCampaignClients(readCtx); err != nil {
		d.RenderCampaignsListPage(w, r, d.PageContext(w, r), nil)
		return
	}
	if !d.CampaignClientReady() {
		d.RenderCampaignsListPage(w, r, d.PageContext(w, r), nil)
		return
	}

	readCtx = grpcauthctx.WithUserID(readCtx, userID)
	if cachedCampaigns, ok := d.CachedUserCampaigns(readCtx, userID); ok {
		d.RenderCampaignsListPage(w, r, d.PageContext(w, r), cachedCampaigns)
		return
	}

	resp, err := d.ListCampaigns(readCtx, &statev1.ListCampaignsRequest{})
	if err != nil {
		d.RenderErrorPage(w, r, grpcStatusFromError(d, err, http.StatusBadGateway), "Campaigns unavailable", "failed to list campaigns")
		return
	}

	campaigns := resp.GetCampaigns()
	d.SetUserCampaignsCache(readCtx, userID, campaigns)
	d.RenderCampaignsListPage(w, r, d.PageContext(w, r), campaigns)
}

func HandleAppCampaignCreate(d AppCampaignDependencies, w http.ResponseWriter, r *http.Request) {
	if d.LocalizeError == nil {
		http.NotFound(w, r)
		return
	}
	if d.CampaignSessionPresent == nil || d.PageContext == nil || d.RenderCampaignCreatePage == nil || d.CampaignReadContext == nil || d.EnsureCampaignClients == nil || d.CampaignClientReady == nil || d.CreateCampaign == nil || d.ExpireUserCampaignsCache == nil || d.SessionDisplayName == nil {
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodGet {
		if !d.CampaignSessionPresent(w, r) {
			return
		}
		d.RenderCampaignCreatePage(w, r, d.PageContext(w, r))
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	readCtx, userID, ok := d.CampaignReadContext(w, r, "Campaign create unavailable")
	if !ok {
		return
	}
	readReq := r.WithContext(readCtx)
	if err := d.EnsureCampaignClients(readCtx); err != nil {
		d.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Campaign create unavailable", "campaign service client is not configured")
		return
	}
	if !d.CampaignClientReady() {
		d.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Campaign create unavailable", "campaign service client is not configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Campaign create unavailable", "failed to parse campaign create form")
		return
	}

	campaignName := strings.TrimSpace(r.FormValue("name"))
	if campaignName == "" {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Campaign create unavailable", "campaign name is required")
		return
	}
	systemValue := strings.TrimSpace(r.FormValue("system"))
	if systemValue == "" {
		systemValue = "daggerheart"
	}
	system, ok := parseAppGameSystem(systemValue)
	if !ok {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Campaign create unavailable", "campaign system is invalid")
		return
	}
	gmModeValue := strings.TrimSpace(r.FormValue("gm_mode"))
	if gmModeValue == "" {
		gmModeValue = "human"
	}
	gmMode, ok := parseAppGmMode(gmModeValue)
	if !ok {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Campaign create unavailable", "campaign gm mode is invalid")
		return
	}
	themePrompt := strings.TrimSpace(r.FormValue("theme_prompt"))
	creatorDisplayName := strings.TrimSpace(r.FormValue("creator_display_name"))
	if creatorDisplayName == "" {
		creatorDisplayName = strings.TrimSpace(d.SessionDisplayName(r))
	}

	resp, err := d.CreateCampaign(readReq.Context(), &statev1.CreateCampaignRequest{
		Name:               campaignName,
		Locale:             commonv1.Locale_LOCALE_EN_US,
		System:             system,
		GmMode:             gmMode,
		ThemePrompt:        themePrompt,
		CreatorDisplayName: creatorDisplayName,
	})
	if err != nil {
		d.RenderErrorPage(w, r, http.StatusBadGateway, "Campaign create unavailable", "failed to create campaign")
		return
	}

	campaignID := strings.TrimSpace(resp.GetCampaign().GetId())
	if campaignID == "" {
		d.RenderErrorPage(w, r, http.StatusBadGateway, "Campaign create unavailable", "created campaign id was empty")
		return
	}
	d.ExpireUserCampaignsCache(readCtx, userID)
	http.Redirect(w, r, routepath.Campaign(campaignID), http.StatusFound)
}

func HandleAppCampaignDetail(d AppCampaignDependencies, w http.ResponseWriter, r *http.Request) {
	HandleCampaignDetailPath(w, r, NewService(buildCampaignHandlers(d)))
}

func HandleAppCampaignOverview(d AppCampaignDependencies, w http.ResponseWriter, r *http.Request, campaignID string) {
	if d.LocalizeError == nil {
		http.NotFound(w, r)
		return
	}
	if d.PageContext == nil || d.CampaignReadContext == nil || d.PageContextForCampaign == nil || d.RenderCampaignPage == nil || d.SetCampaignCache == nil || d.GetCampaign == nil || d.CampaignClientReady == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	readCtx, _, ok := d.CampaignReadContext(w, r, "Campaign unavailable")
	if !ok {
		return
	}
	readReq := r.WithContext(readCtx)
	if d.CampaignClientReady() {
		resp, err := d.GetCampaign(readCtx, &statev1.GetCampaignRequest{CampaignId: campaignID})
		if err != nil {
			d.RenderErrorPage(w, r, grpcStatusFromError(d, err, http.StatusBadGateway), "Campaign unavailable", "failed to load campaign")
			return
		}
		if resp == nil || resp.GetCampaign() == nil {
			d.RenderErrorPage(w, r, http.StatusNotFound, "Campaign unavailable", "campaign not found")
			return
		}
		d.SetCampaignCache(readCtx, resp.GetCampaign())
	}
	d.RenderCampaignPage(w, readReq, campaignID)
}

func HandleAppCampaignSessions(d AppCampaignDependencies, w http.ResponseWriter, r *http.Request, campaignID string) {
	if d.LocalizeError == nil {
		http.NotFound(w, r)
		return
	}
	if d.CampaignReadContext == nil || d.PageContextForCampaign == nil || d.ListSessions == nil || d.CachedCampaignSessions == nil || d.SetCampaignSessionsCache == nil || d.CampaignParticipantByUserID == nil || d.RenderCampaignSessionsPage == nil || d.SessionClientReady == nil || d.CachedCampaignParticipants == nil {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	readCtx, userID, ok := d.CampaignReadContext(w, r, "Sessions unavailable")
	if !ok {
		return
	}
	readReq := r.WithContext(readCtx)
	if !d.SessionClientReady() {
		d.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Sessions unavailable", "session service client is not configured")
		return
	}
	participant, err := d.CampaignParticipantByUserID(readCtx, campaignID, userID)
	if err != nil {
		d.RenderErrorPage(w, r, http.StatusBadGateway, "Sessions unavailable", "failed to resolve campaign participant")
		return
	}
	canManageSessions := participant != nil && d.CanManageCampaignAccess != nil && d.CanManageCampaignAccess(participant.GetCampaignAccess())

	if cachedSessions, ok := d.CachedCampaignSessions(readCtx, campaignID); ok {
		d.RenderCampaignSessionsPage(w, readReq, d.PageContextForCampaign(w, readReq, campaignID), campaignID, cachedSessions, canManageSessions)
		return
	}

	resp, err := d.ListSessions(readCtx, &statev1.ListSessionsRequest{CampaignId: campaignID, PageSize: 10})
	if err != nil {
		d.RenderErrorPage(w, r, grpcStatusFromError(d, err, http.StatusBadGateway), "Sessions unavailable", "failed to list sessions")
		return
	}
	sessions := resp.GetSessions()
	d.SetCampaignSessionsCache(readCtx, campaignID, sessions)
	d.RenderCampaignSessionsPage(w, readReq, d.PageContextForCampaign(w, readReq, campaignID), campaignID, sessions, canManageSessions)
}

func HandleAppCampaignSessionDetail(d AppCampaignDependencies, w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	if d.LocalizeError == nil {
		http.NotFound(w, r)
		return
	}
	if d.CampaignReadContext == nil || d.PageContextForCampaign == nil || d.GetSession == nil || d.RenderCampaignSessionDetailPage == nil || d.SessionClientReady == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	readCtx, _, ok := d.CampaignReadContext(w, r, "Session unavailable")
	if !ok {
		return
	}
	if !d.SessionClientReady() {
		d.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Session unavailable", "session service client is not configured")
		return
	}
	readReq := r.WithContext(readCtx)

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Session unavailable", "session id is required")
		return
	}

	resp, err := d.GetSession(readCtx, &statev1.GetSessionRequest{CampaignId: campaignID, SessionId: sessionID})
	if err != nil {
		d.RenderErrorPage(w, r, grpcStatusFromError(d, err, http.StatusBadGateway), "Session unavailable", "failed to load session")
		return
	}
	if resp.GetSession() == nil {
		d.RenderErrorPage(w, r, http.StatusNotFound, "Session unavailable", "session not found")
		return
	}
	d.RenderCampaignSessionDetailPage(w, readReq, d.PageContextForCampaign(w, readReq, campaignID), campaignID, resp.GetSession())
}

func HandleAppCampaignSessionStart(d AppCampaignDependencies, w http.ResponseWriter, r *http.Request, campaignID string) {
	if d.LocalizeError == nil {
		http.NotFound(w, r)
		return
	}
	if d.RequireCampaignActor == nil || d.StartSession == nil || d.SessionClientReady == nil || d.CanManageCampaignAccess == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	actor, ok := d.RequireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	if !d.SessionClientReady() {
		d.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Session action unavailable", "session service client is not configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Session action unavailable", "failed to parse session start form")
		return
	}
	sessionName := strings.TrimSpace(r.FormValue("name"))
	if sessionName == "" {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Session action unavailable", "session name is required")
		return
	}
	if !d.CanManageCampaignAccess(actor.GetCampaignAccess()) {
		d.RenderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for session action")
		return
	}

	ctx := grpcauthctx.WithParticipantID(r.Context(), strings.TrimSpace(actor.GetId()))
	if _, err := d.StartSession(ctx, &statev1.StartSessionRequest{CampaignId: campaignID, Name: sessionName}); err != nil {
		d.RenderErrorPage(w, r, grpcStatusFromError(d, err, http.StatusBadGateway), "Session action unavailable", "failed to start session")
		return
	}
	http.Redirect(w, r, routepath.CampaignSessions(campaignID), http.StatusFound)
}

func HandleAppCampaignSessionEnd(d AppCampaignDependencies, w http.ResponseWriter, r *http.Request, campaignID string) {
	if d.LocalizeError == nil {
		http.NotFound(w, r)
		return
	}
	if d.RequireCampaignActor == nil || d.EndSession == nil || d.SessionClientReady == nil || d.CanManageCampaignAccess == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	actor, ok := d.RequireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	if !d.SessionClientReady() {
		d.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Session action unavailable", "session service client is not configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Session action unavailable", "failed to parse session end form")
		return
	}
	sessionID := strings.TrimSpace(r.FormValue("session_id"))
	if sessionID == "" {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Session action unavailable", "session id is required")
		return
	}
	if !d.CanManageCampaignAccess(actor.GetCampaignAccess()) {
		d.RenderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for session action")
		return
	}

	ctx := grpcauthctx.WithParticipantID(r.Context(), strings.TrimSpace(actor.GetId()))
	if _, err := d.EndSession(ctx, &statev1.EndSessionRequest{CampaignId: campaignID, SessionId: sessionID}); err != nil {
		d.RenderErrorPage(w, r, grpcStatusFromError(d, err, http.StatusBadGateway), "Session action unavailable", "failed to end session")
		return
	}
	http.Redirect(w, r, routepath.CampaignSessions(campaignID), http.StatusFound)
}

func HandleAppCampaignParticipants(d AppCampaignDependencies, w http.ResponseWriter, r *http.Request, campaignID string) {
	if d.LocalizeError == nil {
		http.NotFound(w, r)
		return
	}
	if d.CampaignReadContext == nil || d.PageContextForCampaign == nil || d.ListParticipants == nil || d.CachedCampaignParticipants == nil || d.SetCampaignParticipantsCache == nil || d.RenderCampaignParticipantsPage == nil || d.CampaignParticipantByUserID == nil || d.ParticipantClientReady == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	readCtx, userID, ok := d.CampaignReadContext(w, r, "Participants unavailable")
	if !ok {
		return
	}
	if !d.ParticipantClientReady() {
		d.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Participants unavailable", "participant service client is not configured")
		return
	}
	if cached, ok := d.CachedCampaignParticipants(readCtx, campaignID); ok {
		canManageParticipants := d.CanManageCampaignAccess != nil && d.CanManageCampaignAccess(campaignAccessForUser(cached, userID))
		readReq := r.WithContext(readCtx)
		d.RenderCampaignParticipantsPage(w, readReq, d.PageContextForCampaign(w, readReq, campaignID), campaignID, cached, canManageParticipants)
		return
	}

	resp, err := d.ListParticipants(readCtx, &statev1.ListParticipantsRequest{CampaignId: campaignID, PageSize: 10})
	if err != nil {
		d.RenderErrorPage(w, r, grpcStatusFromError(d, err, http.StatusBadGateway), "Participants unavailable", "failed to list participants")
		return
	}
	participants := resp.GetParticipants()
	d.SetCampaignParticipantsCache(readCtx, campaignID, participants)
	canManageParticipants := d.CanManageCampaignAccess != nil && d.CanManageCampaignAccess(campaignAccessForUser(participants, userID))
	readReq := r.WithContext(readCtx)
	d.RenderCampaignParticipantsPage(w, readReq, d.PageContextForCampaign(w, readReq, campaignID), campaignID, participants, canManageParticipants)
}

func HandleAppCampaignParticipantUpdate(d AppCampaignDependencies, w http.ResponseWriter, r *http.Request, campaignID string) {
	if d.LocalizeError == nil {
		http.NotFound(w, r)
		return
	}
	if d.RequireCampaignActor == nil || d.ParticipantClientReady == nil || d.UpdateParticipant == nil || d.CanManageCampaignAccess == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	actor, ok := d.RequireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	if !d.ParticipantClientReady() {
		d.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Participant action unavailable", "participant service client is not configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Participant action unavailable", "failed to parse participant update form")
		return
	}
	targetParticipantID := strings.TrimSpace(r.FormValue("participant_id"))
	if targetParticipantID == "" {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Participant action unavailable", "participant id is required")
		return
	}

	updateReq := &statev1.UpdateParticipantRequest{
		CampaignId:    campaignID,
		ParticipantId: targetParticipantID,
	}
	hasFieldUpdate := false
	if rawAccess := strings.TrimSpace(r.FormValue("campaign_access")); rawAccess != "" {
		targetAccess, ok := parseCampaignAccessFormValue(rawAccess)
		if !ok {
			d.RenderErrorPage(w, r, http.StatusBadRequest, "Participant action unavailable", "campaign access value is invalid")
			return
		}
		updateReq.CampaignAccess = targetAccess
		hasFieldUpdate = true
	}
	if rawRole := strings.TrimSpace(r.FormValue("role")); rawRole != "" {
		targetRole, ok := parseParticipantRoleFormValue(rawRole)
		if !ok {
			d.RenderErrorPage(w, r, http.StatusBadRequest, "Participant action unavailable", "participant role value is invalid")
			return
		}
		updateReq.Role = targetRole
		hasFieldUpdate = true
	}
	if rawController := strings.TrimSpace(r.FormValue("controller")); rawController != "" {
		targetController, ok := parseParticipantControllerFormValue(rawController)
		if !ok {
			d.RenderErrorPage(w, r, http.StatusBadRequest, "Participant action unavailable", "participant controller value is invalid")
			return
		}
		updateReq.Controller = targetController
		hasFieldUpdate = true
	}
	if !hasFieldUpdate {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Participant action unavailable", "at least one participant field is required")
		return
	}
	if !d.CanManageCampaignAccess(actor.GetCampaignAccess()) {
		d.RenderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for participant action")
		return
	}

	ctx := grpcauthctx.WithParticipantID(r.Context(), strings.TrimSpace(actor.GetId()))
	if _, err := d.UpdateParticipant(ctx, updateReq); err != nil {
		d.RenderErrorPage(w, r, http.StatusBadGateway, "Participant action unavailable", "failed to update participant")
		return
	}
	http.Redirect(w, r, routepath.CampaignParticipants(campaignID), http.StatusFound)
}

func HandleAppCampaignCharacters(d AppCampaignDependencies, w http.ResponseWriter, r *http.Request, campaignID string) {
	if d.LocalizeError == nil {
		http.NotFound(w, r)
		return
	}
	if d.CampaignReadContext == nil || d.PageContextForCampaign == nil || d.ListCharacters == nil || d.CachedCampaignCharacters == nil || d.SetCampaignCharactersCache == nil || d.CachedCampaignParticipants == nil || d.ListParticipants == nil || d.SetCampaignParticipantsCache == nil || d.RenderCampaignCharactersPage == nil || d.CanManageCampaignAccess == nil || d.CampaignParticipantByUserID == nil || d.CharacterClientReady == nil || d.ParticipantClientReady == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	readCtx, userID, ok := d.CampaignReadContext(w, r, "Characters unavailable")
	if !ok {
		return
	}
	if !d.CharacterClientReady() {
		d.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Characters unavailable", "character service client is not configured")
		return
	}
	readReq := r.WithContext(readCtx)

	actingParticipant, err := d.CampaignParticipantByUserID(readCtx, campaignID, userID)
	if err != nil {
		d.RenderErrorPage(w, r, http.StatusBadGateway, "Characters unavailable", "failed to resolve campaign participant")
		return
	}
	canManage := d.CanManageCampaignAccess(actingParticipant.GetCampaignAccess())

	controlParticipants := []*statev1.Participant(nil)
	if canManage {
		if !d.ParticipantClientReady() {
			d.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Characters unavailable", "participant service client is not configured")
			return
		}
		if cachedParticipants, ok := d.CachedCampaignParticipants(readCtx, campaignID); ok {
			controlParticipants = cachedParticipants
		} else {
			resp, err := d.ListParticipants(readCtx, &statev1.ListParticipantsRequest{CampaignId: campaignID, PageSize: 10})
			if err != nil {
				d.RenderErrorPage(w, r, grpcStatusFromError(d, err, http.StatusBadGateway), "Characters unavailable", "failed to list participants")
				return
			}
			controlParticipants = resp.GetParticipants()
			d.SetCampaignParticipantsCache(readCtx, campaignID, controlParticipants)
		}
	}

	if cachedCharacters, ok := d.CachedCampaignCharacters(readCtx, campaignID); ok {
		d.RenderCampaignCharactersPage(w, readReq, d.PageContextForCampaign(w, readReq, campaignID), campaignID, cachedCharacters, canManage, controlParticipants)
		return
	}
	resp, err := d.ListCharacters(readCtx, &statev1.ListCharactersRequest{CampaignId: campaignID, PageSize: 10})
	if err != nil {
		d.RenderErrorPage(w, r, grpcStatusFromError(d, err, http.StatusBadGateway), "Characters unavailable", "failed to list characters")
		return
	}
	characters := resp.GetCharacters()
	d.SetCampaignCharactersCache(readCtx, campaignID, characters)
	d.RenderCampaignCharactersPage(w, readReq, d.PageContextForCampaign(w, readReq, campaignID), campaignID, characters, canManage, controlParticipants)
}

func HandleAppCampaignCharacterDetail(d AppCampaignDependencies, w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
	if d.LocalizeError == nil {
		http.NotFound(w, r)
		return
	}
	if d.CampaignReadContext == nil || d.PageContextForCampaign == nil || d.GetCharacterSheet == nil || d.CharacterClientReady == nil || d.RenderCampaignCharacterDetailPage == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	readCtx, _, ok := d.CampaignReadContext(w, r, "Character unavailable")
	if !ok {
		return
	}
	if !d.CharacterClientReady() {
		d.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Character unavailable", "character service client is not configured")
		return
	}
	readReq := r.WithContext(readCtx)

	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Character unavailable", "character id is required")
		return
	}

	resp, err := d.GetCharacterSheet(readCtx, &statev1.GetCharacterSheetRequest{CampaignId: campaignID, CharacterId: characterID})
	if err != nil {
		d.RenderErrorPage(w, r, grpcStatusFromError(d, err, http.StatusBadGateway), "Character unavailable", "failed to load character")
		return
	}
	if resp.GetCharacter() == nil {
		d.RenderErrorPage(w, r, http.StatusNotFound, "Character unavailable", "character not found")
		return
	}
	d.RenderCampaignCharacterDetailPage(w, readReq, d.PageContextForCampaign(w, readReq, campaignID), campaignID, resp.GetCharacter())
}

func HandleAppCampaignCharacterCreate(d AppCampaignDependencies, w http.ResponseWriter, r *http.Request, campaignID string) {
	if d.LocalizeError == nil {
		http.NotFound(w, r)
		return
	}
	if d.RequireCampaignActor == nil || d.CharacterClientReady == nil || d.CreateCharacter == nil || d.CanManageCampaignAccess == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	actor, ok := d.RequireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	if !d.CharacterClientReady() {
		d.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Character action unavailable", "character service client is not configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Character action unavailable", "failed to parse character create form")
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Character action unavailable", "character name is required")
		return
	}
	kind, ok := parseCharacterKindFormValue(r.FormValue("kind"))
	if !ok {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Character action unavailable", "character kind value is invalid")
		return
	}
	if !d.CanManageCampaignAccess(actor.GetCampaignAccess()) {
		d.RenderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for character action")
		return
	}

	ctx := grpcauthctx.WithParticipantID(r.Context(), strings.TrimSpace(actor.GetId()))
	if _, err := d.CreateCharacter(ctx, &statev1.CreateCharacterRequest{CampaignId: campaignID, Name: name, Kind: kind}); err != nil {
		d.RenderErrorPage(w, r, http.StatusBadGateway, "Character action unavailable", "failed to create character")
		return
	}
	http.Redirect(w, r, routepath.CampaignCharacters(campaignID), http.StatusFound)
}

func HandleAppCampaignCharacterUpdate(d AppCampaignDependencies, w http.ResponseWriter, r *http.Request, campaignID string) {
	if d.LocalizeError == nil {
		http.NotFound(w, r)
		return
	}
	if d.RequireCampaignActor == nil || d.CharacterClientReady == nil || d.UpdateCharacter == nil || d.CanManageCampaignAccess == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	actor, ok := d.RequireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	if !d.CharacterClientReady() {
		d.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Character action unavailable", "character service client is not configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Character action unavailable", "failed to parse character update form")
		return
	}
	characterID := strings.TrimSpace(r.FormValue("character_id"))
	if characterID == "" {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Character action unavailable", "character id is required")
		return
	}
	req := &statev1.UpdateCharacterRequest{CampaignId: campaignID, CharacterId: characterID}
	hasFieldUpdate := false
	if name := strings.TrimSpace(r.FormValue("name")); name != "" {
		req.Name = &wrapperspb.StringValue{Value: name}
		hasFieldUpdate = true
	}
	if rawKind := strings.TrimSpace(r.FormValue("kind")); rawKind != "" {
		kind, ok := parseCharacterKindFormValue(rawKind)
		if !ok {
			d.RenderErrorPage(w, r, http.StatusBadRequest, "Character action unavailable", "character kind value is invalid")
			return
		}
		req.Kind = kind
		hasFieldUpdate = true
	}
	if !hasFieldUpdate {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Character action unavailable", "at least one character field is required")
		return
	}
	if !d.CanManageCampaignAccess(actor.GetCampaignAccess()) {
		d.RenderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for character action")
		return
	}
	ctx := grpcauthctx.WithParticipantID(r.Context(), strings.TrimSpace(actor.GetId()))
	if _, err := d.UpdateCharacter(ctx, req); err != nil {
		d.RenderErrorPage(w, r, http.StatusBadGateway, "Character action unavailable", "failed to update character")
		return
	}
	http.Redirect(w, r, routepath.CampaignCharacters(campaignID), http.StatusFound)
}

func HandleAppCampaignCharacterControl(d AppCampaignDependencies, w http.ResponseWriter, r *http.Request, campaignID string) {
	if d.LocalizeError == nil {
		http.NotFound(w, r)
		return
	}
	if d.RequireCampaignActor == nil || d.CharacterClientReady == nil || d.SetDefaultControl == nil || d.CanManageCampaignAccess == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	actor, ok := d.RequireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	if !d.CharacterClientReady() {
		d.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Character action unavailable", "character service client is not configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Character action unavailable", "failed to parse character controller form")
		return
	}
	characterID := strings.TrimSpace(r.FormValue("character_id"))
	if characterID == "" {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Character action unavailable", "character id is required")
		return
	}
	participantID := strings.TrimSpace(r.FormValue("participant_id"))
	if !d.CanManageCampaignAccess(actor.GetCampaignAccess()) {
		d.RenderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for character action")
		return
	}

	ctx := grpcauthctx.WithParticipantID(r.Context(), strings.TrimSpace(actor.GetId()))
	req := &statev1.SetDefaultControlRequest{
		CampaignId:    campaignID,
		CharacterId:   characterID,
		ParticipantId: wrapperspb.String(participantID),
	}
	if _, err := d.SetDefaultControl(ctx, req); err != nil {
		d.RenderErrorPage(w, r, http.StatusBadGateway, "Character action unavailable", "failed to set character controller")
		return
	}
	http.Redirect(w, r, routepath.CampaignCharacters(campaignID), http.StatusFound)
}

func HandleAppCampaignInvites(d AppCampaignDependencies, w http.ResponseWriter, r *http.Request, campaignID string) {
	if d.LocalizeError == nil {
		http.NotFound(w, r)
		return
	}
	if d.CampaignReadContext == nil || d.PageContextForCampaign == nil || d.ListInvites == nil || d.CachedCampaignInvites == nil || d.SetCampaignInvitesCache == nil || d.ListInviteContactOptions == nil || d.RenderCampaignInvitesPage == nil || d.InviteClientReady == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	readCtx, userID, ok := d.CampaignReadContext(w, r, "Invites unavailable")
	if !ok {
		return
	}
	readReq := r.WithContext(readCtx)
	if !d.InviteClientReady() {
		d.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Invites unavailable", "campaign invite service is not configured")
		return
	}
	invites := []*statev1.Invite(nil)
	if cachedInvites, ok := d.CachedCampaignInvites(readCtx, campaignID, userID); ok {
		invites = cachedInvites
	} else {
		resp, err := d.ListInvites(readCtx, &statev1.ListInvitesRequest{CampaignId: campaignID, PageSize: 10})
		if err != nil {
			d.RenderErrorPage(w, r, grpcStatusFromError(d, err, http.StatusBadGateway), "Invites unavailable", "failed to list campaign invites")
			return
		}
		invites = resp.GetInvites()
		d.SetCampaignInvitesCache(readCtx, campaignID, userID, invites)
	}

	contactOptions := d.ListInviteContactOptions(readCtx, campaignID, userID, invites)
	d.RenderCampaignInvitesPage(w, readReq, d.PageContextForCampaign(w, readReq, campaignID), campaignID, invites, contactOptions, true)
}

func HandleAppCampaignInvitesVerification(d AppCampaignDependencies, w http.ResponseWriter, r *http.Request, campaignID string, userID string, canManageInvites bool, verification templates.CampaignInviteVerification) {
	if d.CachedCampaignInvites == nil || d.ListInvites == nil || d.SetCampaignInvitesCache == nil || d.ListInviteContactOptions == nil || d.PageContextForCampaign == nil || d.RenderErrorPage == nil || d.InviteClientReady == nil {
		http.NotFound(w, r)
		return
	}
	if !d.InviteClientReady() {
		d.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Invites unavailable", "campaign invite service is not configured")
		return
	}

	userID = strings.TrimSpace(userID)
	ctx := grpcauthctx.WithUserID(r.Context(), userID)
	readReq := r.WithContext(ctx)
	invites := []*statev1.Invite(nil)
	if cachedInvites, ok := d.CachedCampaignInvites(ctx, campaignID, userID); ok {
		invites = cachedInvites
	} else {
		resp, err := d.ListInvites(ctx, &statev1.ListInvitesRequest{
			CampaignId: campaignID,
			PageSize:   10,
		})
		if err != nil {
			d.RenderErrorPage(w, r, grpcStatusFromError(d, err, http.StatusBadGateway), "Invites unavailable", "failed to list campaign invites")
			return
		}
		invites = resp.GetInvites()
		d.SetCampaignInvitesCache(ctx, campaignID, userID, invites)
	}
	contacts := d.ListInviteContactOptions(ctx, campaignID, userID, invites)
	page := d.PageContextForCampaign(w, readReq, campaignID)
	RenderCampaignInvitesPageWithContext(w, readReq, page, campaignID, invites, contacts, canManageInvites, verification)
}

func HandleAppCampaignInviteCreate(d AppCampaignDependencies, w http.ResponseWriter, r *http.Request, campaignID string) {
	if d.LocalizeError == nil {
		http.NotFound(w, r)
		return
	}
	if d.RequireCampaignActor == nil || d.ResolveInviteRecipientUserID == nil || d.CreateInvite == nil || d.CanManageCampaignAccess == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	actor, ok := d.RequireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	actorID := strings.TrimSpace(actor.GetId())
	if actorID == "" {
		d.RenderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for invite action")
		return
	}
	if !d.CanManageCampaignAccess(actor.GetCampaignAccess()) {
		d.RenderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for invite action")
		return
	}
	if d.InviteClientReady == nil || !d.InviteClientReady() {
		d.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Invite action unavailable", "campaign invite service is not configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Invite action unavailable", "failed to parse invite create form")
		return
	}
	participantID := strings.TrimSpace(r.FormValue("participant_id"))
	if participantID == "" {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Invite action unavailable", "participant id is required")
		return
	}

	lookupCtx := grpcauthctx.WithUserID(r.Context(), strings.TrimSpace(actor.GetUserId()))
	if strings.EqualFold(strings.TrimSpace(r.FormValue("action")), "verify") {
		verification, err := d.LookupInviteRecipientVerification(lookupCtx, strings.TrimSpace(r.FormValue("recipient_user_id")))
		if err != nil {
			d.RenderInviteRecipientLookupError(w, r, err)
			return
		}
		d.RenderCampaignInvitesVerificationPage(w, r, campaignID, actorID, true, verification)
		return
	}

	recipientUserID, err := d.ResolveInviteRecipientUserID(lookupCtx, strings.TrimSpace(r.FormValue("recipient_user_id")))
	if err != nil {
		d.RenderInviteRecipientLookupError(w, r, err)
		return
	}
	if _, err := d.CreateInvite(
		grpcauthctx.WithParticipantID(r.Context(), actorID),
		&statev1.CreateInviteRequest{
			CampaignId:      campaignID,
			ParticipantId:   participantID,
			RecipientUserId: recipientUserID,
		},
	); err != nil {
		d.RenderErrorPage(w, r, grpcStatusFromError(d, err, http.StatusBadGateway), "Invite action unavailable", "failed to create invite")
		return
	}
	http.Redirect(w, r, routepath.CampaignInvites(campaignID), http.StatusFound)
}

func HandleAppCampaignInviteRevoke(d AppCampaignDependencies, w http.ResponseWriter, r *http.Request, campaignID string) {
	if d.LocalizeError == nil {
		http.NotFound(w, r)
		return
	}
	if d.RequireCampaignActor == nil || d.RevokeInvite == nil || d.CanManageCampaignAccess == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		d.LocalizeError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	actor, ok := d.RequireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	actorID := strings.TrimSpace(actor.GetId())
	if actorID == "" {
		d.RenderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for invite action")
		return
	}
	if !d.CanManageCampaignAccess(actor.GetCampaignAccess()) {
		d.RenderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for invite action")
		return
	}
	if d.InviteClientReady == nil || !d.InviteClientReady() {
		d.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Invite action unavailable", "campaign invite service is not configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Invite action unavailable", "failed to parse invite revoke form")
		return
	}
	inviteID := strings.TrimSpace(r.FormValue("invite_id"))
	if inviteID == "" {
		d.RenderErrorPage(w, r, http.StatusBadRequest, "Invite action unavailable", "invite id is required")
		return
	}
	if _, err := d.RevokeInvite(
		grpcauthctx.WithParticipantID(r.Context(), actorID),
		&statev1.RevokeInviteRequest{InviteId: inviteID},
	); err != nil {
		d.RenderErrorPage(w, r, grpcStatusFromError(d, err, http.StatusBadGateway), "Invite action unavailable", "failed to revoke invite")
		return
	}
	http.Redirect(w, r, routepath.CampaignInvites(campaignID), http.StatusFound)
}

func buildCampaignHandlers(d AppCampaignDependencies) Handlers {
	return Handlers{
		Campaigns:      func(w http.ResponseWriter, r *http.Request) { HandleAppCampaigns(d, w, r) },
		CampaignCreate: func(w http.ResponseWriter, r *http.Request) { HandleAppCampaignCreate(d, w, r) },
		CampaignOverview: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			HandleAppCampaignOverview(d, w, r, campaignID)
		},
		CampaignSessions: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			HandleAppCampaignSessions(d, w, r, campaignID)
		},
		CampaignSessionStart: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			HandleAppCampaignSessionStart(d, w, r, campaignID)
		},
		CampaignSessionEnd: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			HandleAppCampaignSessionEnd(d, w, r, campaignID)
		},
		CampaignSessionDetail: func(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
			HandleAppCampaignSessionDetail(d, w, r, campaignID, sessionID)
		},
		CampaignParticipants: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			HandleAppCampaignParticipants(d, w, r, campaignID)
		},
		CampaignParticipantUpdate: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			HandleAppCampaignParticipantUpdate(d, w, r, campaignID)
		},
		CampaignCharacters: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			HandleAppCampaignCharacters(d, w, r, campaignID)
		},
		CampaignCharacterCreate: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			HandleAppCampaignCharacterCreate(d, w, r, campaignID)
		},
		CampaignCharacterUpdate: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			HandleAppCampaignCharacterUpdate(d, w, r, campaignID)
		},
		CampaignCharacterControl: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			HandleAppCampaignCharacterControl(d, w, r, campaignID)
		},
		CampaignCharacterDetail: func(w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
			HandleAppCampaignCharacterDetail(d, w, r, campaignID, characterID)
		},
		CampaignInvites: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			HandleAppCampaignInvites(d, w, r, campaignID)
		},
		CampaignInviteCreate: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			HandleAppCampaignInviteCreate(d, w, r, campaignID)
		},
		CampaignInviteRevoke: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			HandleAppCampaignInviteRevoke(d, w, r, campaignID)
		},
	}
}

func campaignAccessForUser(participants []*statev1.Participant, userID string) statev1.CampaignAccess {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return statev1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED
	}
	for _, participant := range participants {
		if participant == nil {
			continue
		}
		if strings.TrimSpace(participant.GetUserId()) == userID {
			return participant.GetCampaignAccess()
		}
	}
	return statev1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED
}

func parseCampaignAccessFormValue(raw string) (statev1.CampaignAccess, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "member", "campaign_access_member":
		return statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER, true
	case "manager", "campaign_access_manager":
		return statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER, true
	case "owner", "campaign_access_owner":
		return statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER, true
	default:
		return statev1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED, false
	}
}

func parseParticipantRoleFormValue(raw string) (statev1.ParticipantRole, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "gm", "participant_role_gm":
		return statev1.ParticipantRole_GM, true
	case "player", "participant_role_player":
		return statev1.ParticipantRole_PLAYER, true
	default:
		return statev1.ParticipantRole_ROLE_UNSPECIFIED, false
	}
}

func parseParticipantControllerFormValue(raw string) (statev1.Controller, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "human", "controller_human":
		return statev1.Controller_CONTROLLER_HUMAN, true
	case "ai", "controller_ai":
		return statev1.Controller_CONTROLLER_AI, true
	default:
		return statev1.Controller_CONTROLLER_UNSPECIFIED, false
	}
}

func parseCharacterKindFormValue(raw string) (statev1.CharacterKind, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "pc", "character_kind_pc":
		return statev1.CharacterKind_PC, true
	case "npc", "character_kind_npc":
		return statev1.CharacterKind_NPC, true
	default:
		return statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED, false
	}
}

func parseAppGameSystem(value string) (commonv1.GameSystem, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "daggerheart", "game_system_daggerheart":
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, true
	default:
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, false
	}
}

func parseAppGmMode(value string) (statev1.GmMode, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "human":
		return statev1.GmMode_HUMAN, true
	case "ai":
		return statev1.GmMode_AI, true
	case "hybrid":
		return statev1.GmMode_HYBRID, true
	default:
		return statev1.GmMode_GM_MODE_UNSPECIFIED, false
	}
}

func grpcStatusFromError(d AppCampaignDependencies, err error, defaultStatus int) int {
	if d.GRPCErrorStatus == nil {
		return defaultStatus
	}
	return d.GRPCErrorStatus(err, defaultStatus)
}
