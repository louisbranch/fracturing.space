package campaigns

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
	"google.golang.org/protobuf/types/known/timestamppb"
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

// service implements the campaigns module route service contract.
type service struct {
	base modulehandler.Base
}

var _ Service = (*service)(nil)

// NewService builds a campaigns module-local service implementation.
func NewService(base modulehandler.Base) Service {
	return &service{base: base}
}

// HandleCampaignsPage renders the campaigns page.
func (s *service) HandleCampaignsPage(w http.ResponseWriter, r *http.Request) {
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
func (s *service) HandleCampaignsTable(w http.ResponseWriter, r *http.Request) {
	loc, _ := s.base.Localizer(w, r)
	campaignClient := s.base.CampaignClient()
	if campaignClient == nil {
		s.renderCampaignTable(w, r, nil, loc.Sprintf("error.campaign_service_unavailable"), loc)
		return
	}

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := campaignClient.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
	if err != nil {
		log.Printf("list campaigns: %v", err)
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
func (s *service) HandleCampaignDetail(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := s.base.Localizer(w, r)
	campaignClient := s.base.CampaignClient()
	if campaignClient == nil {
		s.renderCampaignDetail(w, r, templates.CampaignDetail{}, loc.Sprintf("error.campaign_service_unavailable"), lang, loc)
		return
	}

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := campaignClient.GetCampaign(ctx, &statev1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil {
		log.Printf("get campaign: %v", err)
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
func (s *service) HandleCharactersList(w http.ResponseWriter, r *http.Request, campaignID string) {
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
func (s *service) HandleCharactersTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := s.base.Localizer(w, r)
	characterClient := s.base.CharacterClient()
	if characterClient == nil {
		s.renderCharactersTable(w, r, nil, loc.Sprintf("error.character_service_unavailable"), loc)
		return
	}

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := characterClient.ListCharacters(ctx, &statev1.ListCharactersRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		log.Printf("list characters: %v", err)
		s.renderCharactersTable(w, r, nil, loc.Sprintf("error.characters_unavailable"), loc)
		return
	}

	characters := response.GetCharacters()
	if len(characters) == 0 {
		s.renderCharactersTable(w, r, nil, loc.Sprintf("error.no_characters"), loc)
		return
	}

	participantNames := map[string]string{}
	if participantClient := s.base.ParticipantClient(); participantClient != nil {
		participantsResp, err := participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
			CampaignId: campaignID,
		})
		if err != nil {
			log.Printf("list participants for character table: %v", err)
		} else {
			for _, participant := range participantsResp.GetParticipants() {
				if participant != nil {
					participantNames[participant.GetId()] = participant.GetName()
				}
			}
		}
	}

	rows := buildCharacterRows(characters, participantNames, loc)
	s.renderCharactersTable(w, r, rows, "", loc)
}

// HandleCharacterSheet renders a character details page with the info tab active.
func (s *service) HandleCharacterSheet(w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
	s.renderCharacterSheet(w, r, campaignID, characterID, "info")
}

// HandleCharacterActivity renders a character details page with the activity tab active.
func (s *service) HandleCharacterActivity(w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
	s.renderCharacterSheet(w, r, campaignID, characterID, "activity")
}

func (s *service) renderCharacterSheet(w http.ResponseWriter, r *http.Request, campaignID string, characterID string, activePage string) {
	loc, lang := s.base.Localizer(w, r)
	characterClient := s.base.CharacterClient()
	if characterClient == nil {
		http.Error(w, loc.Sprintf("error.character_service_unavailable"), http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := characterClient.GetCharacterSheet(ctx, &statev1.GetCharacterSheetRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		log.Printf("get character sheet: %v", err)
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
	if eventClient := s.base.EventClient(); eventClient != nil {
		eventsResp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
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
						Type:        formatEventType(event.GetType(), loc),
						Timestamp:   formatTimestamp(event.GetTs()),
						Description: formatEventDescription(event, loc),
						PayloadJSON: string(event.GetPayloadJson()),
					})
				}
			}
		}
	}

	controller := loc.Sprintf("label.unassigned")
	participantID := ""
	if character.GetParticipantId() != nil {
		participantID = strings.TrimSpace(character.GetParticipantId().GetValue())
	}
	if participantID != "" {
		if participantClient := s.base.ParticipantClient(); participantClient != nil {
			participantResp, err := participantClient.GetParticipant(ctx, &statev1.GetParticipantRequest{
				CampaignId:    campaignID,
				ParticipantId: participantID,
			})
			if err != nil {
				log.Printf("get participant for character sheet: %v", err)
				controller = loc.Sprintf("label.unknown")
			} else if participant := participantResp.GetParticipant(); participant != nil {
				controller = participant.GetName()
			} else {
				controller = loc.Sprintf("label.unknown")
			}
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
func (s *service) HandleParticipantsList(w http.ResponseWriter, r *http.Request, campaignID string) {
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
func (s *service) HandleParticipantsTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := s.base.Localizer(w, r)
	participantClient := s.base.ParticipantClient()
	if participantClient == nil {
		s.renderParticipantsTable(w, r, nil, loc.Sprintf("error.participant_service_unavailable"), loc)
		return
	}

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		log.Printf("list participants: %v", err)
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
func (s *service) HandleInvitesList(w http.ResponseWriter, r *http.Request, campaignID string) {
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
func (s *service) HandleInvitesTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := s.base.Localizer(w, r)
	inviteClient := s.base.InviteClient()
	if inviteClient == nil {
		s.renderInvitesTable(w, r, nil, loc.Sprintf("error.invite_service_unavailable"), loc)
		return
	}

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := inviteClient.ListInvites(ctx, &statev1.ListInvitesRequest{
		CampaignId: campaignID,
		PageSize:   inviteListPageSize,
	})
	if err != nil {
		log.Printf("list invites: %v", err)
		s.renderInvitesTable(w, r, nil, loc.Sprintf("error.invites_unavailable"), loc)
		return
	}

	invites := response.GetInvites()
	if len(invites) == 0 {
		s.renderInvitesTable(w, r, nil, loc.Sprintf("error.no_invites"), loc)
		return
	}

	participantNames := map[string]string{}
	if participantClient := s.base.ParticipantClient(); participantClient != nil {
		participantsResp, err := participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
			CampaignId: campaignID,
		})
		if err != nil {
			log.Printf("list participants for invites: %v", err)
		} else {
			for _, participant := range participantsResp.GetParticipants() {
				if participant != nil {
					participantNames[participant.GetId()] = participant.GetName()
				}
			}
		}
	}

	recipientNames := map[string]string{}
	if authClient := s.base.AuthClient(); authClient != nil {
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
			userResp, err := authClient.GetUser(ctx, &authv1.GetUserRequest{UserId: recipientID})
			if err != nil {
				log.Printf("get invite recipient: %v", err)
				recipientNames[recipientID] = ""
				continue
			}
			if user := userResp.GetUser(); user != nil {
				recipientNames[recipientID] = user.GetEmail()
			}
		}
	}

	rows := buildInviteRows(invites, participantNames, recipientNames, loc)
	s.renderInvitesTable(w, r, rows, "", loc)
}

// HandleSessionsList renders the sessions list page.
func (s *service) HandleSessionsList(w http.ResponseWriter, r *http.Request, campaignID string) {
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
func (s *service) HandleSessionsTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := s.base.Localizer(w, r)
	sessionClient := s.base.SessionClient()
	if sessionClient == nil {
		s.renderCampaignSessions(w, r, nil, loc.Sprintf("error.session_service_unavailable"), loc)
		return
	}

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := sessionClient.ListSessions(ctx, &statev1.ListSessionsRequest{
		CampaignId: campaignID,
		PageSize:   sessionListPageSize,
	})
	if err != nil {
		log.Printf("list sessions: %v", err)
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
func (s *service) HandleSessionDetail(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	loc, lang := s.base.Localizer(w, r)
	sessionClient := s.base.SessionClient()
	if sessionClient == nil {
		http.Error(w, loc.Sprintf("error.session_service_unavailable"), http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := sessionClient.GetSession(ctx, &statev1.GetSessionRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err != nil {
		log.Printf("get session: %v", err)
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
	if eventClient := s.base.EventClient(); eventClient != nil {
		eventsResp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   1,
			Filter:     "session_id = \"" + sessionID + "\"",
		})
		if err == nil && eventsResp != nil {
			eventCount = eventsResp.GetTotalSize()
		}
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
func (s *service) HandleSessionEvents(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
	loc, _ := s.base.Localizer(w, r)
	eventClient := s.base.EventClient()
	if eventClient == nil {
		templ.Handler(templates.EmptyState(loc.Sprintf("error.event_service_unavailable"))).ServeHTTP(w, r)
		return
	}

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	pageToken := r.URL.Query().Get("page_token")

	eventsResp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   eventListPageSize,
		PageToken:  pageToken,
		OrderBy:    "seq desc",
		Filter:     "session_id = \"" + sessionID + "\"",
	})
	if err != nil {
		log.Printf("list session events: %v", err)
		templ.Handler(templates.EmptyState(loc.Sprintf("error.events_unavailable"))).ServeHTTP(w, r)
		return
	}

	campaignName := s.getCampaignName(r, campaignID, loc)
	sessionName := s.getSessionName(r, campaignID, sessionID, loc)

	events := buildEventRows(eventsResp.GetEvents(), loc)
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
func (s *service) HandleEventLog(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := s.base.Localizer(w, r)
	campaignName := s.getCampaignName(r, campaignID, loc)
	filters := parseEventFilters(r)

	var events []templates.EventRow
	var totalCount int32
	var nextToken, prevToken string

	if eventClient := s.base.EventClient(); eventClient != nil {
		ctx, cancel := s.base.GameGRPCCallContext(r.Context())
		defer cancel()

		filterExpr := buildEventFilterExpression(filters)

		eventsResp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
			CampaignId: campaignID,
			PageSize:   eventListPageSize,
			OrderBy:    "seq desc",
			Filter:     filterExpr,
		})
		if err == nil && eventsResp != nil {
			events = buildEventRows(eventsResp.GetEvents(), loc)
			totalCount = eventsResp.GetTotalSize()
			nextToken = eventsResp.GetNextPageToken()
			prevToken = eventsResp.GetPreviousPageToken()
		}
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
func (s *service) HandleEventLogTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := s.base.Localizer(w, r)
	eventClient := s.base.EventClient()
	if eventClient == nil {
		templ.Handler(templates.EmptyState(loc.Sprintf("error.event_service_unavailable"))).ServeHTTP(w, r)
		return
	}

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	filters := parseEventFilters(r)
	filterExpr := buildEventFilterExpression(filters)
	pageToken := r.URL.Query().Get("page_token")

	eventsResp, err := eventClient.ListEvents(ctx, &statev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   eventListPageSize,
		PageToken:  pageToken,
		OrderBy:    "seq desc",
		Filter:     filterExpr,
	})
	if err != nil {
		log.Printf("list events: %v", err)
		templ.Handler(templates.EmptyState(loc.Sprintf("error.events_unavailable"))).ServeHTTP(w, r)
		return
	}

	campaignName := s.getCampaignName(r, campaignID, loc)
	events := buildEventRows(eventsResp.GetEvents(), loc)

	view := templates.EventLogView{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		Events:       events,
		Filters:      filters,
		NextToken:    eventsResp.GetNextPageToken(),
		PrevToken:    eventsResp.GetPreviousPageToken(),
		TotalCount:   eventsResp.GetTotalSize(),
	}

	if pushURL := eventFilterPushURL(routepath.CampaignEvents(campaignID), filters, pageToken); pushURL != "" {
		w.Header().Set("HX-Push-Url", pushURL)
	}

	templ.Handler(templates.EventLogTableContent(view, loc)).ServeHTTP(w, r)
}

func (s *service) renderCampaignTable(w http.ResponseWriter, r *http.Request, rows []templates.CampaignRow, message string, loc *message.Printer) {
	templ.Handler(templates.CampaignsTable(rows, message, loc)).ServeHTTP(w, r)
}

func (s *service) renderCampaignDetail(w http.ResponseWriter, r *http.Request, detail templates.CampaignDetail, message string, lang string, loc *message.Printer) {
	pageCtx := s.base.PageContext(lang, loc, r)
	s.base.RenderPage(
		w,
		r,
		templates.CampaignDetailPage(detail, message, loc),
		templates.CampaignDetailFullPage(detail, message, pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.campaign", templates.AppName()),
	)
}

func (s *service) renderCampaignSessions(w http.ResponseWriter, r *http.Request, rows []templates.CampaignSessionRow, message string, loc *message.Printer) {
	templ.Handler(templates.CampaignSessionsList(rows, message, loc)).ServeHTTP(w, r)
}

func (s *service) renderCharactersTable(w http.ResponseWriter, r *http.Request, rows []templates.CharacterRow, message string, loc *message.Printer) {
	templ.Handler(templates.CharactersTable(rows, message, loc)).ServeHTTP(w, r)
}

func (s *service) renderParticipantsTable(w http.ResponseWriter, r *http.Request, rows []templates.ParticipantRow, message string, loc *message.Printer) {
	templ.Handler(templates.ParticipantsTable(rows, message, loc)).ServeHTTP(w, r)
}

func (s *service) renderInvitesTable(w http.ResponseWriter, r *http.Request, rows []templates.InviteRow, message string, loc *message.Printer) {
	templ.Handler(templates.InvitesTable(rows, message, loc)).ServeHTTP(w, r)
}

func (s *service) getCampaignName(r *http.Request, campaignID string, loc *message.Printer) string {
	campaignClient := s.base.CampaignClient()
	if campaignClient == nil {
		return loc.Sprintf("label.campaign")
	}

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := campaignClient.GetCampaign(ctx, &statev1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil || response == nil || response.GetCampaign() == nil {
		return loc.Sprintf("label.campaign")
	}

	return response.GetCampaign().GetName()
}

func (s *service) getSessionName(r *http.Request, campaignID string, sessionID string, loc *message.Printer) string {
	sessionClient := s.base.SessionClient()
	if sessionClient == nil {
		return loc.Sprintf("label.session")
	}

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	response, err := sessionClient.GetSession(ctx, &statev1.GetSessionRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err != nil || response == nil || response.GetSession() == nil {
		return loc.Sprintf("label.session")
	}

	return response.GetSession().GetName()
}

func buildCampaignRows(campaigns []*statev1.Campaign, loc *message.Printer) []templates.CampaignRow {
	rows := make([]templates.CampaignRow, 0, len(campaigns))
	for _, campaign := range campaigns {
		if campaign == nil {
			continue
		}
		rows = append(rows, templates.CampaignRow{
			ID:               campaign.GetId(),
			Name:             campaign.GetName(),
			System:           formatGameSystem(campaign.GetSystem(), loc),
			GMMode:           formatGmMode(campaign.GetGmMode(), loc),
			ParticipantCount: strconv.FormatInt(int64(campaign.GetParticipantCount()), 10),
			CharacterCount:   strconv.FormatInt(int64(campaign.GetCharacterCount()), 10),
			ThemePrompt:      truncateText(campaign.GetThemePrompt(), campaignThemePromptLimit),
			CreatedDate:      formatCreatedDate(campaign.GetCreatedAt()),
		})
	}
	return rows
}

func buildCampaignDetail(campaign *statev1.Campaign, loc *message.Printer) templates.CampaignDetail {
	if campaign == nil {
		return templates.CampaignDetail{}
	}
	return templates.CampaignDetail{
		ID:               campaign.GetId(),
		Name:             campaign.GetName(),
		System:           formatGameSystem(campaign.GetSystem(), loc),
		GMMode:           formatGmMode(campaign.GetGmMode(), loc),
		ParticipantCount: strconv.FormatInt(int64(campaign.GetParticipantCount()), 10),
		CharacterCount:   strconv.FormatInt(int64(campaign.GetCharacterCount()), 10),
		ThemePrompt:      campaign.GetThemePrompt(),
		CreatedAt:        formatTimestamp(campaign.GetCreatedAt()),
		UpdatedAt:        formatTimestamp(campaign.GetUpdatedAt()),
	}
}

func buildCampaignSessionRows(sessions []*statev1.Session, loc *message.Printer) []templates.CampaignSessionRow {
	rows := make([]templates.CampaignSessionRow, 0, len(sessions))
	for _, session := range sessions {
		if session == nil {
			continue
		}
		statusBadge := "secondary"
		if session.GetStatus() == statev1.SessionStatus_SESSION_ACTIVE {
			statusBadge = "success"
		}
		row := templates.CampaignSessionRow{
			ID:          session.GetId(),
			CampaignID:  session.GetCampaignId(),
			Name:        session.GetName(),
			Status:      formatSessionStatus(session.GetStatus(), loc),
			StatusBadge: statusBadge,
			StartedAt:   formatTimestamp(session.GetStartedAt()),
		}
		if session.GetEndedAt() != nil {
			row.EndedAt = formatTimestamp(session.GetEndedAt())
		}
		rows = append(rows, row)
	}
	return rows
}

func buildCharacterRows(characters []*statev1.Character, participantNames map[string]string, loc *message.Printer) []templates.CharacterRow {
	rows := make([]templates.CharacterRow, 0, len(characters))
	for _, character := range characters {
		if character == nil {
			continue
		}

		controller := formatCharacterController(character, participantNames, loc)

		rows = append(rows, templates.CharacterRow{
			ID:         character.GetId(),
			CampaignID: character.GetCampaignId(),
			Name:       character.GetName(),
			Kind:       formatCharacterKind(character.GetKind(), loc),
			Controller: controller,
		})
	}
	return rows
}

func buildCharacterSheet(campaignID string, campaignName string, character *statev1.Character, recentEvents []templates.EventRow, controller string, loc *message.Printer) templates.CharacterSheetView {
	return templates.CharacterSheetView{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		Character:    character,
		Controller:   controller,
		CreatedAt:    formatTimestamp(character.GetCreatedAt()),
		UpdatedAt:    formatTimestamp(character.GetUpdatedAt()),
		RecentEvents: recentEvents,
	}
}

func buildInviteRows(invites []*statev1.Invite, participantNames map[string]string, recipientNames map[string]string, loc *message.Printer) []templates.InviteRow {
	rows := make([]templates.InviteRow, 0, len(invites))
	for _, inv := range invites {
		if inv == nil {
			continue
		}

		participantLabel := participantNames[inv.GetParticipantId()]
		if participantLabel == "" {
			participantLabel = loc.Sprintf("label.unknown")
		}

		recipientLabel := loc.Sprintf("label.unassigned")
		recipientID := strings.TrimSpace(inv.GetRecipientUserId())
		if recipientID != "" {
			recipientLabel = recipientNames[recipientID]
			if recipientLabel == "" {
				recipientLabel = recipientID
			}
		}

		statusLabel, statusVariant := formatInviteStatus(inv.GetStatus(), loc)

		rows = append(rows, templates.InviteRow{
			ID:            inv.GetId(),
			CampaignID:    inv.GetCampaignId(),
			Participant:   participantLabel,
			Recipient:     recipientLabel,
			Status:        statusLabel,
			StatusVariant: statusVariant,
			CreatedAt:     formatTimestamp(inv.GetCreatedAt()),
			UpdatedAt:     formatTimestamp(inv.GetUpdatedAt()),
		})
	}
	return rows
}

func buildParticipantRows(participants []*statev1.Participant, loc *message.Printer) []templates.ParticipantRow {
	rows := make([]templates.ParticipantRow, 0, len(participants))
	for _, participant := range participants {
		if participant == nil {
			continue
		}

		role, roleVariant := formatParticipantRole(participant.GetRole(), loc)
		access, accessVariant := formatParticipantAccess(participant.GetCampaignAccess(), loc)
		controller, controllerVariant := formatParticipantController(participant.GetController(), loc)

		rows = append(rows, templates.ParticipantRow{
			ID:                participant.GetId(),
			Name:              participant.GetName(),
			Role:              role,
			RoleVariant:       roleVariant,
			Access:            access,
			AccessVariant:     accessVariant,
			Controller:        controller,
			ControllerVariant: controllerVariant,
			CreatedDate:       formatCreatedDate(participant.GetCreatedAt()),
		})
	}
	return rows
}

func buildSessionDetail(campaignID string, campaignName string, session *statev1.Session, eventCount int32, loc *message.Printer) templates.SessionDetail {
	if session == nil {
		return templates.SessionDetail{}
	}

	status := formatSessionStatus(session.GetStatus(), loc)
	statusBadge := "secondary"
	if session.GetStatus() == statev1.SessionStatus_SESSION_ACTIVE {
		statusBadge = "success"
	}

	detail := templates.SessionDetail{
		CampaignID:   campaignID,
		CampaignName: campaignName,
		ID:           session.GetId(),
		Name:         session.GetName(),
		Status:       status,
		StatusBadge:  statusBadge,
		StartedAt:    formatTimestamp(session.GetStartedAt()),
		EventCount:   eventCount,
	}

	if session.GetEndedAt() != nil {
		detail.EndedAt = formatTimestamp(session.GetEndedAt())
	}

	return detail
}

func buildEventRows(events []*statev1.Event, loc *message.Printer) []templates.EventRow {
	rows := make([]templates.EventRow, 0, len(events))
	for _, event := range events {
		if event == nil {
			continue
		}
		rows = append(rows, templates.EventRow{
			CampaignID:       event.GetCampaignId(),
			Seq:              event.GetSeq(),
			Hash:             event.GetHash(),
			Type:             event.GetType(),
			TypeDisplay:      formatEventType(event.GetType(), loc),
			Timestamp:        formatTimestamp(event.GetTs()),
			SessionID:        event.GetSessionId(),
			ActorType:        event.GetActorType(),
			ActorTypeDisplay: formatActorType(event.GetActorType(), loc),
			ActorName:        "",
			EntityType:       event.GetEntityType(),
			EntityID:         event.GetEntityId(),
			EntityName:       event.GetEntityId(),
			Description:      formatEventDescription(event, loc),
			PayloadJSON:      string(event.GetPayloadJson()),
		})
	}
	return rows
}

func parseEventFilters(r *http.Request) templates.EventFilterOptions {
	return templates.EventFilterOptions{
		SessionID:  r.URL.Query().Get("session_id"),
		EventType:  r.URL.Query().Get("event_type"),
		ActorType:  r.URL.Query().Get("actor_type"),
		EntityType: r.URL.Query().Get("entity_type"),
		StartDate:  r.URL.Query().Get("start_date"),
		EndDate:    r.URL.Query().Get("end_date"),
	}
}

func eventFilterPushURL(basePath string, filters templates.EventFilterOptions, pageToken string) string {
	pushURL := templates.EventFilterBaseURL(basePath, filters)
	if pageToken != "" {
		return templates.AppendQueryParam(pushURL, "page_token", pageToken)
	}
	return pushURL
}

func escapeAIP160StringLiteral(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `"`, `\"`)
	return v
}

func buildEventFilterExpression(filters templates.EventFilterOptions) string {
	var parts []string

	if filters.SessionID != "" {
		parts = append(parts, "session_id = \""+escapeAIP160StringLiteral(filters.SessionID)+"\"")
	}
	if filters.EventType != "" {
		parts = append(parts, "type = \""+escapeAIP160StringLiteral(filters.EventType)+"\"")
	}
	if filters.ActorType != "" {
		parts = append(parts, "actor_type = \""+escapeAIP160StringLiteral(filters.ActorType)+"\"")
	}
	if filters.EntityType != "" {
		parts = append(parts, "entity_type = \""+escapeAIP160StringLiteral(filters.EntityType)+"\"")
	}
	if filters.StartDate != "" {
		parts = append(parts, "ts >= timestamp(\""+escapeAIP160StringLiteral(filters.StartDate)+"T00:00:00Z\")")
	}
	if filters.EndDate != "" {
		parts = append(parts, "ts <= timestamp(\""+escapeAIP160StringLiteral(filters.EndDate)+"T23:59:59Z\")")
	}

	return strings.Join(parts, " AND ")
}

func formatGmMode(mode statev1.GmMode, loc *message.Printer) string {
	switch mode {
	case statev1.GmMode_HUMAN:
		return loc.Sprintf("label.human")
	case statev1.GmMode_AI:
		return loc.Sprintf("label.ai")
	case statev1.GmMode_HYBRID:
		return loc.Sprintf("label.hybrid")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatGameSystem(system commonv1.GameSystem, loc *message.Printer) string {
	switch system {
	case commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART:
		return loc.Sprintf("label.daggerheart")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatSessionStatus(status statev1.SessionStatus, loc *message.Printer) string {
	switch status {
	case statev1.SessionStatus_SESSION_ACTIVE:
		return loc.Sprintf("label.active")
	case statev1.SessionStatus_SESSION_ENDED:
		return loc.Sprintf("label.ended")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatInviteStatus(status statev1.InviteStatus, loc *message.Printer) (string, string) {
	switch status {
	case statev1.InviteStatus_PENDING:
		return loc.Sprintf("label.invite_pending"), "warning"
	case statev1.InviteStatus_CLAIMED:
		return loc.Sprintf("label.invite_claimed"), "success"
	case statev1.InviteStatus_REVOKED:
		return loc.Sprintf("label.invite_revoked"), "error"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

func formatCreatedDate(createdAt *timestamppb.Timestamp) string {
	if createdAt == nil {
		return ""
	}
	return createdAt.AsTime().Format("2006-01-02")
}

func formatTimestamp(value *timestamppb.Timestamp) string {
	if value == nil {
		return ""
	}
	return value.AsTime().Format("2006-01-02 15:04:05")
}

func truncateText(text string, limit int) string {
	if limit <= 0 || text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "..."
}

func formatParticipantRole(role statev1.ParticipantRole, loc *message.Printer) (string, string) {
	switch role {
	case statev1.ParticipantRole_GM:
		return loc.Sprintf("label.gm"), "info"
	case statev1.ParticipantRole_PLAYER:
		return loc.Sprintf("label.player"), "success"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

func formatParticipantController(controller statev1.Controller, loc *message.Printer) (string, string) {
	switch controller {
	case statev1.Controller_CONTROLLER_HUMAN:
		return loc.Sprintf("label.human"), "success"
	case statev1.Controller_CONTROLLER_AI:
		return loc.Sprintf("label.ai"), "info"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

func formatParticipantAccess(access statev1.CampaignAccess, loc *message.Printer) (string, string) {
	switch access {
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER:
		return loc.Sprintf("label.member"), "secondary"
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER:
		return loc.Sprintf("label.manager"), "info"
	case statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER:
		return loc.Sprintf("label.owner"), "warning"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

func formatCharacterController(character *statev1.Character, participantNames map[string]string, loc *message.Printer) string {
	if character == nil {
		return loc.Sprintf("label.unassigned")
	}
	participantID := ""
	if character.GetParticipantId() != nil {
		participantID = strings.TrimSpace(character.GetParticipantId().GetValue())
	}
	if participantID == "" {
		return loc.Sprintf("label.unassigned")
	}
	if name, ok := participantNames[participantID]; ok {
		return name
	}
	return loc.Sprintf("label.unknown")
}

func formatCharacterKind(kind statev1.CharacterKind, loc *message.Printer) string {
	switch kind {
	case statev1.CharacterKind_PC:
		return loc.Sprintf("label.pc")
	case statev1.CharacterKind_NPC:
		return loc.Sprintf("label.npc")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatEventType(eventType string, loc *message.Printer) string {
	switch eventType {
	case "campaign.created":
		return loc.Sprintf("event.campaign_created")
	case "campaign.forked":
		return loc.Sprintf("event.campaign_forked")
	case "campaign.updated":
		return loc.Sprintf("event.campaign_updated")
	case "participant.joined":
		return loc.Sprintf("event.participant_joined")
	case "participant.left":
		return loc.Sprintf("event.participant_left")
	case "participant.updated":
		return loc.Sprintf("event.participant_updated")
	case "character.created":
		return loc.Sprintf("event.character_created")
	case "character.deleted":
		return loc.Sprintf("event.character_deleted")
	case "character.updated":
		return loc.Sprintf("event.character_updated")
	case "character.profile_updated":
		return loc.Sprintf("event.character_profile_updated")
	case "session.started":
		return loc.Sprintf("event.session_started")
	case "session.ended":
		return loc.Sprintf("event.session_ended")
	case "session.gate_opened":
		return loc.Sprintf("event.session_gate_opened")
	case "session.gate_resolved":
		return loc.Sprintf("event.session_gate_resolved")
	case "session.gate_abandoned":
		return loc.Sprintf("event.session_gate_abandoned")
	case "session.spotlight_set":
		return loc.Sprintf("event.session_spotlight_set")
	case "session.spotlight_cleared":
		return loc.Sprintf("event.session_spotlight_cleared")
	case "invite.created":
		return loc.Sprintf("event.invite_created")
	case "invite.updated":
		return loc.Sprintf("event.invite_updated")
	case "action.roll_resolved":
		return loc.Sprintf("event.action_roll_resolved")
	case "action.outcome_applied":
		return loc.Sprintf("event.action_outcome_applied")
	case "action.outcome_rejected":
		return loc.Sprintf("event.action_outcome_rejected")
	case "action.note_added":
		return loc.Sprintf("event.action_note_added")
	case "action.character_state_patched":
		return loc.Sprintf("event.action_character_state_patched")
	case "action.gm_fear_changed":
		return loc.Sprintf("event.action_gm_fear_changed")
	case "action.death_move_resolved":
		return loc.Sprintf("event.action_death_move_resolved")
	case "action.blaze_of_glory_resolved":
		return loc.Sprintf("event.action_blaze_of_glory_resolved")
	case "action.attack_resolved":
		return loc.Sprintf("event.action_attack_resolved")
	case "action.reaction_resolved":
		return loc.Sprintf("event.action_reaction_resolved")
	case "action.damage_roll_resolved":
		return loc.Sprintf("event.action_damage_roll_resolved")
	case "action.adversary_action_resolved":
		return loc.Sprintf("event.action_adversary_action_resolved")
	default:
		parts := strings.Split(eventType, ".")
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			if len(last) > 0 {
				formatted := strings.ReplaceAll(last, "_", " ")
				return strings.ToUpper(formatted[:1]) + formatted[1:]
			}
		}
		return eventType
	}
}

func formatActorType(actorType string, loc *message.Printer) string {
	if actorType == "" {
		return ""
	}
	switch actorType {
	case "system":
		return loc.Sprintf("filter.actor.system")
	case "participant":
		return loc.Sprintf("filter.actor.participant")
	case "gm":
		return loc.Sprintf("filter.actor.gm")
	default:
		return actorType
	}
}

func formatEventDescription(event *statev1.Event, loc *message.Printer) string {
	if event == nil {
		return ""
	}
	return formatEventType(event.GetType(), loc)
}
