package campaigns

import (
	"net/http"
	"strings"

	"github.com/a-h/templ"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/eventview"
	adminerrors "github.com/louisbranch/fracturing.space/internal/services/admin/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
)

const (
	// campaignThemePromptLimit caps the number of theme prompt characters shown in campaign tables.
	campaignThemePromptLimit = 80
	// sessionListPageSize caps the number of sessions shown in sessions tables.
	sessionListPageSize = 10
	// eventListPageSize caps the number of events shown per page.
	eventListPageSize = 50
	// inviteListPageSize caps the number of invites shown per page.
	inviteListPageSize = 50
)

// handlers implements the campaigns module Handlers contract.
type handlers struct {
	base              modulehandler.Base
	campaignClient    statev1.CampaignServiceClient
	characterClient   statev1.CharacterServiceClient
	participantClient statev1.ParticipantServiceClient
	inviteClient      statev1.InviteServiceClient
	sessionClient     statev1.SessionServiceClient
	eventClient       statev1.EventServiceClient
	authClient        authv1.AuthServiceClient
}

var _ Handlers = (*handlers)(nil)

// NewHandlers builds a campaigns module handler implementation.
func NewHandlers(
	base modulehandler.Base,
	campaignClient statev1.CampaignServiceClient,
	characterClient statev1.CharacterServiceClient,
	participantClient statev1.ParticipantServiceClient,
	inviteClient statev1.InviteServiceClient,
	sessionClient statev1.SessionServiceClient,
	eventClient statev1.EventServiceClient,
	authClient authv1.AuthServiceClient,
) Handlers {
	return &handlers{
		base:              base,
		campaignClient:    campaignClient,
		characterClient:   characterClient,
		participantClient: participantClient,
		inviteClient:      inviteClient,
		sessionClient:     sessionClient,
		eventClient:       eventClient,
		authClient:        authClient,
	}
}

// HandleCampaignsPage renders the campaigns page.
func (s *handlers) HandleCampaignsPage(w http.ResponseWriter, r *http.Request) {
	loc, lang := s.base.Localizer(w, r)
	pageCtx := s.base.PageContext(lang, loc, r)
	s.base.RenderPage(
		w,
		r,
		templates.CampaignsPage(loc),
		templates.CampaignsFullPage(pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.campaigns", templates.AppName()),
	)
}

// HandleCampaignsTable renders the campaigns table fragment.
func (s *handlers) HandleCampaignsTable(w http.ResponseWriter, r *http.Request) {
	loc, _ := s.base.Localizer(w, r)

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := s.campaignClient.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
	if err != nil {
		adminerrors.LogError(r, "list campaigns: %v", err)
		s.renderCampaignTable(w, r, nil, loc.Sprintf("error.campaigns_unavailable"), loc)
		return
	}

	campaigns := response.GetCampaigns()
	if len(campaigns) == 0 {
		s.renderCampaignTable(w, r, nil, loc.Sprintf("error.no_campaigns"), loc)
		return
	}

	rows := buildCampaignRows(campaigns, loc)
	s.renderCampaignTable(w, r, rows, "", loc)
}

// HandleCampaignDetail renders a campaign detail page.
func (s *handlers) HandleCampaignDetail(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := s.base.Localizer(w, r)

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := s.campaignClient.GetCampaign(ctx, &statev1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil {
		adminerrors.LogError(r, "get campaign: %v", err)
		s.renderCampaignDetail(w, r, templates.CampaignDetail{}, loc.Sprintf("error.campaign_unavailable"), lang, loc)
		return
	}

	campaign := response.GetCampaign()
	if campaign == nil {
		s.renderCampaignDetail(w, r, templates.CampaignDetail{}, loc.Sprintf("error.campaign_not_found"), lang, loc)
		return
	}

	detail := buildCampaignDetail(campaign, loc)
	s.renderCampaignDetail(w, r, detail, "", lang, loc)
}

// HandleCharactersList renders the character list page.
func (s *handlers) HandleCharactersList(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := s.base.Localizer(w, r)
	campaignName := s.getCampaignName(r, campaignID, loc)
	pageCtx := s.base.PageContext(lang, loc, r)
	s.base.RenderPage(
		w,
		r,
		templates.CharactersListPage(campaignID, campaignName, loc),
		templates.CharactersListFullPage(campaignID, campaignName, pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.characters", templates.AppName()),
	)
}

// HandleCharactersTable renders the character rows fragment.
func (s *handlers) HandleCharactersTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := s.base.Localizer(w, r)

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := s.characterClient.ListCharacters(ctx, &statev1.ListCharactersRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		adminerrors.LogError(r, "list characters: %v", err)
		s.renderCharactersTable(w, r, nil, loc.Sprintf("error.characters_unavailable"), loc)
		return
	}

	characters := response.GetCharacters()
	if len(characters) == 0 {
		s.renderCharactersTable(w, r, nil, loc.Sprintf("error.no_characters"), loc)
		return
	}

	participantNames := map[string]string{}
	participantsResp, err := s.participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		adminerrors.LogError(r, "list participants for character table: %v", err)
	} else {
		for _, participant := range participantsResp.GetParticipants() {
			if participant != nil {
				participantNames[participant.GetId()] = participant.GetName()
			}
		}
	}

	rows := buildCharacterRows(characters, participantNames, loc)
	s.renderCharactersTable(w, r, rows, "", loc)
}

// HandleCharacterSheet renders a character details page with the info tab active.
func (s *handlers) HandleCharacterSheet(w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
	s.renderCharacterSheet(w, r, campaignID, characterID, "info")
}

// HandleCharacterActivity renders a character details page with the activity tab active.
func (s *handlers) HandleCharacterActivity(w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
	s.renderCharacterSheet(w, r, campaignID, characterID, "activity")
}

func (s *handlers) renderCharacterSheet(w http.ResponseWriter, r *http.Request, campaignID string, characterID string, activePage string) {
	loc, lang := s.base.Localizer(w, r)

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := s.characterClient.GetCharacterSheet(ctx, &statev1.GetCharacterSheetRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		adminerrors.LogError(r, "get character sheet: %v", err)
		http.Error(w, loc.Sprintf("error.character_unavailable"), http.StatusNotFound)
		return
	}

	character := response.GetCharacter()
	if character == nil {
		http.Error(w, loc.Sprintf("error.character_not_found"), http.StatusNotFound)
		return
	}

	campaignName := s.getCampaignName(r, campaignID, loc)

	var recentEvents []templates.EventRow
	eventsResp, err := s.eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   20,
		OrderBy:    "seq desc",
		Filter:     "entity_id = \"" + characterID + "\"",
	})
	if err == nil && eventsResp != nil {
		for _, event := range eventsResp.GetEvents() {
			if event != nil {
				recentEvents = append(recentEvents, templates.EventRow{
					Seq:         event.GetSeq(),
					Type:        eventview.FormatEventType(event.GetType(), loc),
					Timestamp:   eventview.FormatTimestamp(event.GetTs()),
					Description: eventview.FormatEventDescription(event, loc),
					PayloadJSON: string(event.GetPayloadJson()),
				})
			}
		}
	}

	controller := loc.Sprintf("label.unassigned")
	participantID := ""
	if character.GetParticipantId() != nil {
		participantID = strings.TrimSpace(character.GetParticipantId().GetValue())
	}
	if participantID != "" {
		participantResp, err := s.participantClient.GetParticipant(ctx, &statev1.GetParticipantRequest{
			CampaignId:    campaignID,
			ParticipantId: participantID,
		})
		if err != nil {
			adminerrors.LogError(r, "get participant for character sheet: %v", err)
			controller = loc.Sprintf("label.unknown")
		} else if participant := participantResp.GetParticipant(); participant != nil {
			controller = participant.GetName()
		} else {
			controller = loc.Sprintf("label.unknown")
		}
	}

	sheet := buildCharacterSheet(campaignID, campaignName, character, recentEvents, controller, loc)
	pageCtx := s.base.PageContext(lang, loc, r)
	s.base.RenderPage(
		w,
		r,
		templates.CharacterSheetPage(sheet, activePage, loc),
		templates.CharacterSheetFullPage(sheet, activePage, pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.character_sheet", sheet.Character.GetName(), templates.AppName()),
	)
}

// HandleParticipantsList renders the participants list page.
func (s *handlers) HandleParticipantsList(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := s.base.Localizer(w, r)
	campaignName := s.getCampaignName(r, campaignID, loc)
	pageCtx := s.base.PageContext(lang, loc, r)
	s.base.RenderPage(
		w,
		r,
		templates.ParticipantsListPage(campaignID, campaignName, loc),
		templates.ParticipantsListFullPage(campaignID, campaignName, pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.participants", templates.AppName()),
	)
}

// HandleParticipantsTable renders the participant rows fragment.
func (s *handlers) HandleParticipantsTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := s.base.Localizer(w, r)

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := s.participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		adminerrors.LogError(r, "list participants: %v", err)
		s.renderParticipantsTable(w, r, nil, loc.Sprintf("error.participants_unavailable"), loc)
		return
	}

	participants := response.GetParticipants()
	if len(participants) == 0 {
		s.renderParticipantsTable(w, r, nil, loc.Sprintf("error.no_participants"), loc)
		return
	}

	rows := buildParticipantRows(participants, loc)
	s.renderParticipantsTable(w, r, rows, "", loc)
}

// HandleInvitesList renders the invites list page.
func (s *handlers) HandleInvitesList(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := s.base.Localizer(w, r)
	campaignName := s.getCampaignName(r, campaignID, loc)
	pageCtx := s.base.PageContext(lang, loc, r)
	s.base.RenderPage(
		w,
		r,
		templates.InvitesListPage(campaignID, campaignName, loc),
		templates.InvitesListFullPage(campaignID, campaignName, pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.invites", templates.AppName()),
	)
}

// HandleInvitesTable renders the invite rows fragment.
func (s *handlers) HandleInvitesTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := s.base.Localizer(w, r)

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := s.inviteClient.ListInvites(ctx, &statev1.ListInvitesRequest{
		CampaignId: campaignID,
		PageSize:   inviteListPageSize,
	})
	if err != nil {
		adminerrors.LogError(r, "list invites: %v", err)
		s.renderInvitesTable(w, r, nil, loc.Sprintf("error.invites_unavailable"), loc)
		return
	}

	invites := response.GetInvites()
	if len(invites) == 0 {
		s.renderInvitesTable(w, r, nil, loc.Sprintf("error.no_invites"), loc)
		return
	}

	participantNames := map[string]string{}
	participantsResp, err := s.participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		adminerrors.LogError(r, "list participants for invites: %v", err)
	} else {
		for _, participant := range participantsResp.GetParticipants() {
			if participant != nil {
				participantNames[participant.GetId()] = participant.GetName()
			}
		}
	}

	recipientNames := map[string]string{}
	for _, inv := range invites {
		if inv == nil {
			continue
		}
		recipientID := strings.TrimSpace(inv.GetRecipientUserId())
		if recipientID == "" {
			continue
		}
		if _, ok := recipientNames[recipientID]; ok {
			continue
		}
		userResp, err := s.authClient.GetUser(ctx, &authv1.GetUserRequest{UserId: recipientID})
		if err != nil {
			adminerrors.LogError(r, "get invite recipient: %v", err)
			recipientNames[recipientID] = ""
			continue
		}
		if user := userResp.GetUser(); user != nil {
			recipientNames[recipientID] = user.GetEmail()
		}
	}

	rows := buildInviteRows(invites, participantNames, recipientNames, loc)
	s.renderInvitesTable(w, r, rows, "", loc)
}

// HandleSessionsList renders the sessions list page.
func (s *handlers) HandleSessionsList(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := s.base.Localizer(w, r)
	campaignName := s.getCampaignName(r, campaignID, loc)
	pageCtx := s.base.PageContext(lang, loc, r)
	s.base.RenderPage(
		w,
		r,
		templates.SessionsListPage(campaignID, campaignName, loc),
		templates.SessionsListFullPage(campaignID, campaignName, pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.sessions", templates.AppName()),
	)
}

// HandleSessionsTable renders the session rows fragment.
func (s *handlers) HandleSessionsTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := s.base.Localizer(w, r)

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := s.sessionClient.ListSessions(ctx, &statev1.ListSessionsRequest{
		CampaignId: campaignID,
		PageSize:   sessionListPageSize,
	})
	if err != nil {
		adminerrors.LogError(r, "list sessions: %v", err)
		s.renderCampaignSessions(w, r, nil, loc.Sprintf("error.sessions_unavailable"), loc)
		return
	}

	sessions := response.GetSessions()
	if len(sessions) == 0 {
		s.renderCampaignSessions(w, r, nil, loc.Sprintf("error.no_sessions"), loc)
		return
	}

	rows := buildCampaignSessionRows(sessions, loc)
	s.renderCampaignSessions(w, r, rows, "", loc)
}

// HandleSessionDetail renders a session detail page.
func (s *handlers) HandleSessionDetail(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	loc, lang := s.base.Localizer(w, r)

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := s.sessionClient.GetSession(ctx, &statev1.GetSessionRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err != nil {
		adminerrors.LogError(r, "get session: %v", err)
		http.Error(w, loc.Sprintf("error.session_unavailable"), http.StatusNotFound)
		return
	}

	session := response.GetSession()
	if session == nil {
		http.Error(w, loc.Sprintf("error.session_not_found"), http.StatusNotFound)
		return
	}

	campaignName := s.getCampaignName(r, campaignID, loc)

	var eventCount int32
	eventsResp, err := s.eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   1,
		Filter:     "session_id = \"" + sessionID + "\"",
	})
	if err == nil && eventsResp != nil {
		eventCount = eventsResp.GetTotalSize()
	}

	detail := buildSessionDetail(campaignID, campaignName, session, eventCount, loc)
	pageCtx := s.base.PageContext(lang, loc, r)
	s.base.RenderPage(
		w,
		r,
		templates.SessionDetailPage(detail, loc),
		templates.SessionDetailFullPage(detail, pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.session", detail.Name, templates.AppName()),
	)
}

// HandleSessionEvents renders session events fragment content.
func (s *handlers) HandleSessionEvents(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	loc, _ := s.base.Localizer(w, r)

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	pageToken := r.URL.Query().Get("page_token")

	eventsResp, err := s.eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   eventListPageSize,
		PageToken:  pageToken,
		OrderBy:    "seq desc",
		Filter:     "session_id = \"" + sessionID + "\"",
	})
	if err != nil {
		adminerrors.LogError(r, "list session events: %v", err)
		templ.Handler(templates.EmptyState(loc.Sprintf("error.events_unavailable"))).ServeHTTP(w, r)
		return
	}

	campaignName := s.getCampaignName(r, campaignID, loc)
	sessionName := s.getSessionName(r, campaignID, sessionID, loc)

	events := eventview.BuildEventRows(eventsResp.GetEvents(), loc)
	detail := templates.SessionDetail{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		ID:           sessionID,
		Name:         sessionName,
		Events:       events,
		EventCount:   eventsResp.GetTotalSize(),
		NextToken:    eventsResp.GetNextPageToken(),
		PrevToken:    eventsResp.GetPreviousPageToken(),
	}

	templ.Handler(templates.SessionEventsContent(detail, loc)).ServeHTTP(w, r)
}

// HandleEventLog renders the event log page.
func (s *handlers) HandleEventLog(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := s.base.Localizer(w, r)
	campaignName := s.getCampaignName(r, campaignID, loc)
	filters := eventview.ParseEventFilters(r)

	var events []templates.EventRow
	var totalCount int32
	var nextToken, prevToken string

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	filterExpr := eventview.BuildEventFilterExpression(filters)

	eventsResp, err := s.eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   eventListPageSize,
		OrderBy:    "seq desc",
		Filter:     filterExpr,
	})
	if err == nil && eventsResp != nil {
		events = eventview.BuildEventRows(eventsResp.GetEvents(), loc)
		totalCount = eventsResp.GetTotalSize()
		nextToken = eventsResp.GetNextPageToken()
		prevToken = eventsResp.GetPreviousPageToken()
	}

	view := templates.EventLogView{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		Events:       events,
		Filters:      filters,
		TotalCount:   totalCount,
		NextToken:    nextToken,
		PrevToken:    prevToken,
	}
	pageCtx := s.base.PageContext(lang, loc, r)
	s.base.RenderPage(
		w,
		r,
		templates.EventLogPage(view, loc),
		templates.EventLogFullPage(view, pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.events", templates.AppName()),
	)
}

// HandleEventLogTable renders event log table fragment rows.
func (s *handlers) HandleEventLogTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := s.base.Localizer(w, r)

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	filters := eventview.ParseEventFilters(r)
	filterExpr := eventview.BuildEventFilterExpression(filters)
	pageToken := r.URL.Query().Get("page_token")

	eventsResp, err := s.eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   eventListPageSize,
		PageToken:  pageToken,
		OrderBy:    "seq desc",
		Filter:     filterExpr,
	})
	if err != nil {
		adminerrors.LogError(r, "list events: %v", err)
		templ.Handler(templates.EmptyState(loc.Sprintf("error.events_unavailable"))).ServeHTTP(w, r)
		return
	}

	campaignName := s.getCampaignName(r, campaignID, loc)
	events := eventview.BuildEventRows(eventsResp.GetEvents(), loc)

	view := templates.EventLogView{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		Events:       events,
		Filters:      filters,
		NextToken:    eventsResp.GetNextPageToken(),
		PrevToken:    eventsResp.GetPreviousPageToken(),
		TotalCount:   eventsResp.GetTotalSize(),
	}

	if pushURL := eventview.EventFilterPushURL(routepath.CampaignEvents(campaignID), filters, pageToken); pushURL != "" {
		w.Header().Set("HX-Push-Url", pushURL)
	}

	templ.Handler(templates.EventLogTableContent(view, loc)).ServeHTTP(w, r)
}

func (s *handlers) renderCampaignTable(w http.ResponseWriter, r *http.Request, rows []templates.CampaignRow, message string, loc *message.Printer) {
	templ.Handler(templates.CampaignsTable(rows, message, loc)).ServeHTTP(w, r)
}

func (s *handlers) renderCampaignDetail(w http.ResponseWriter, r *http.Request, detail templates.CampaignDetail, message string, lang string, loc *message.Printer) {
	pageCtx := s.base.PageContext(lang, loc, r)
	s.base.RenderPage(
		w,
		r,
		templates.CampaignDetailPage(detail, message, loc),
		templates.CampaignDetailFullPage(detail, message, pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.campaign", templates.AppName()),
	)
}

func (s *handlers) renderCampaignSessions(w http.ResponseWriter, r *http.Request, rows []templates.CampaignSessionRow, message string, loc *message.Printer) {
	templ.Handler(templates.CampaignSessionsList(rows, message, loc)).ServeHTTP(w, r)
}

func (s *handlers) renderCharactersTable(w http.ResponseWriter, r *http.Request, rows []templates.CharacterRow, message string, loc *message.Printer) {
	templ.Handler(templates.CharactersTable(rows, message, loc)).ServeHTTP(w, r)
}

func (s *handlers) renderParticipantsTable(w http.ResponseWriter, r *http.Request, rows []templates.ParticipantRow, message string, loc *message.Printer) {
	templ.Handler(templates.ParticipantsTable(rows, message, loc)).ServeHTTP(w, r)
}

func (s *handlers) renderInvitesTable(w http.ResponseWriter, r *http.Request, rows []templates.InviteRow, message string, loc *message.Printer) {
	templ.Handler(templates.InvitesTable(rows, message, loc)).ServeHTTP(w, r)
}

func (s *handlers) getCampaignName(r *http.Request, campaignID string, loc *message.Printer) string {
	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := s.campaignClient.GetCampaign(ctx, &statev1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil || response == nil || response.GetCampaign() == nil {
		return loc.Sprintf("label.campaign")
	}

	return response.GetCampaign().GetName()
}

func (s *handlers) getSessionName(r *http.Request, campaignID string, sessionID string, loc *message.Printer) string {
	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := s.sessionClient.GetSession(ctx, &statev1.GetSessionRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err != nil || response == nil || response.GetSession() == nil {
		return loc.Sprintf("label.session")
	}

	return response.GetSession().GetName()
}
