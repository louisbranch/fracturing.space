package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"google.golang.org/protobuf/types/known/structpb"
)

func newCampaignAuthorizer(
	config Config,
	communicationClient statev1.CommunicationServiceClient,
	participantClient statev1.ParticipantServiceClient,
	sessionClient statev1.SessionServiceClient,
	campaignClient statev1.CampaignServiceClient,
	authSessionClient webSessionAuthClient,
) wsAuthorizer {
	authBaseURL := strings.TrimSpace(config.AuthBaseURL)
	resourceSecret := strings.TrimSpace(config.OAuthResourceSecret)
	if (authBaseURL == "" || resourceSecret == "") && authSessionClient == nil && communicationClient == nil && participantClient == nil {
		return nil
	}
	var httpClient *http.Client
	if authBaseURL != "" && resourceSecret != "" {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}

	return &campaignAuthorizer{
		authBaseURL:         authBaseURL,
		oauthResourceSecret: resourceSecret,
		httpClient:          httpClient,
		authSessionClient:   authSessionClient,
		communicationClient: communicationClient,
		participantClient:   participantClient,
		sessionClient:       sessionClient,
		campaignClient:      campaignClient,
	}
}

func (a *campaignAuthorizer) Authenticate(ctx context.Context, accessToken string) (string, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return "", errors.New("access token is required")
	}
	if strings.HasPrefix(accessToken, webSessionTokenPrefix) {
		return a.authenticateWebSession(ctx, strings.TrimPrefix(accessToken, webSessionTokenPrefix))
	}
	if a == nil || a.httpClient == nil {
		return "", errors.New("auth is not configured")
	}

	endpoint := strings.TrimRight(a.authBaseURL, "/") + "/introspect"
	authCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(authCtx, http.MethodPost, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("build introspection request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Resource-Secret", a.oauthResourceSecret)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call auth introspection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth introspection status %d", resp.StatusCode)
	}

	var payload authIntrospectResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode introspection response: %w", err)
	}
	if !payload.Active {
		return "", errors.New("inactive access token")
	}

	userID := strings.TrimSpace(payload.UserID)
	if userID == "" {
		return "", errors.New("introspection returned empty user id")
	}
	return userID, nil
}

func (a *campaignAuthorizer) authenticateWebSession(ctx context.Context, sessionID string) (string, error) {
	if a == nil || a.authSessionClient == nil {
		return "", errors.New("web session auth is not configured")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", errors.New("web session id is required")
	}
	resp, err := a.authSessionClient.GetWebSession(ctx, &authv1.GetWebSessionRequest{SessionId: sessionID})
	if err != nil {
		return "", fmt.Errorf("lookup web session: %w", err)
	}
	if resp == nil || resp.GetSession() == nil {
		return "", errors.New("web session not found")
	}
	userID := strings.TrimSpace(resp.GetSession().GetUserId())
	if userID == "" {
		return "", errors.New("web session returned empty user id")
	}
	return userID, nil
}

func (a *campaignAuthorizer) IsCampaignParticipant(ctx context.Context, campaignID string, userID string) (bool, error) {
	if a == nil || a.participantClient == nil {
		return false, errors.New("participant client is not configured")
	}

	campaignID = strings.TrimSpace(campaignID)
	userID = strings.TrimSpace(userID)
	if campaignID == "" || userID == "" {
		return false, nil
	}

	participant, err := a.findParticipantByUserID(ctx, campaignID, userID)
	if err != nil {
		return false, err
	}
	return participant != nil, nil
}

func (a *campaignAuthorizer) ResolveJoinWelcome(ctx context.Context, campaignID string, userID string) (joinWelcome, error) {
	if a.communicationClient != nil {
		resolved, err := a.ResolveCommunicationContext(ctx, campaignID, userID)
		if err != nil {
			return joinWelcome{}, err
		}
		return resolved.Welcome, nil
	}

	campaignID = strings.TrimSpace(campaignID)
	userID = strings.TrimSpace(userID)
	if campaignID == "" {
		return joinWelcome{}, errors.New("campaign id is required")
	}

	var activeSession *statev1.Session
	if a.sessionClient != nil {
		var err error
		activeSession, err = a.findActiveSession(ctx, campaignID, userID)
		if err != nil && !errors.Is(err, errCampaignSessionInactive) {
			return joinWelcome{}, err
		}
	}

	participant, err := a.findParticipantByUserID(ctx, campaignID, userID)
	if err != nil {
		return joinWelcome{}, err
	}
	if participant == nil {
		return joinWelcome{}, errCampaignParticipantRequired
	}

	participantName := userID
	if strings.TrimSpace(participant.GetName()) != "" {
		participantName = strings.TrimSpace(participant.GetName())
	}

	campaignName := campaignID
	gmMode := ""
	aiAgentID := ""
	locale := commonv1.Locale_LOCALE_EN_US
	if a.campaignClient != nil {
		callCtx, cancel := context.WithTimeout(grpcauthctx.WithUserID(ctx, userID), 3*time.Second)
		resp, err := a.campaignClient.GetCampaign(callCtx, &statev1.GetCampaignRequest{CampaignId: campaignID})
		cancel()
		if err != nil {
			return joinWelcome{}, fmt.Errorf("get campaign: %w", err)
		}
		if campaign := resp.GetCampaign(); campaign != nil {
			if name := strings.TrimSpace(campaign.GetName()); name != "" {
				campaignName = name
			}
			gmMode = campaign.GetGmMode().String()
			aiAgentID = strings.TrimSpace(campaign.GetAiAgentId())
			if campaign.GetLocale() != commonv1.Locale_LOCALE_UNSPECIFIED {
				locale = campaign.GetLocale()
			}
		}
	}

	sessionID := ""
	sessionName := ""
	if activeSession != nil {
		sessionID = strings.TrimSpace(activeSession.GetId())
		sessionName = strings.TrimSpace(activeSession.GetName())
		if sessionName == "" {
			sessionName = sessionID
		}
	}

	return joinWelcome{
		ParticipantName: strings.TrimSpace(participantName),
		CampaignName:    campaignName,
		SessionID:       sessionID,
		SessionName:     sessionName,
		GmMode:          gmMode,
		AIAgentID:       aiAgentID,
		Locale:          locale,
	}, nil
}

func (a *campaignAuthorizer) ResolveCommunicationContext(ctx context.Context, campaignID string, userID string) (communicationContext, error) {
	if a == nil || a.communicationClient == nil {
		return communicationContext{}, errors.New("communication client is not configured")
	}

	campaignID = strings.TrimSpace(campaignID)
	userID = strings.TrimSpace(userID)
	if campaignID == "" {
		return communicationContext{}, errors.New("campaign id is required")
	}

	callCtx, cancel := context.WithTimeout(grpcauthctx.WithUserID(ctx, userID), 3*time.Second)
	defer cancel()

	resp, err := a.communicationClient.GetCommunicationContext(callCtx, &statev1.GetCommunicationContextRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		return communicationContext{}, fmt.Errorf("get communication context: %w", err)
	}
	return communicationContextFromProto(campaignID, userID, resp.GetContext())
}

func (a *campaignAuthorizer) OpenCommunicationGate(ctx context.Context, campaignID string, participantID string, gateType string, reason string, controlMetadata map[string]any) (communicationContext, error) {
	if a == nil || a.communicationClient == nil {
		return communicationContext{}, errors.New("communication client is not configured")
	}
	requestMetadata, err := mapToStruct(controlMetadata)
	if err != nil {
		return communicationContext{}, fmt.Errorf("encode communication gate metadata: %w", err)
	}
	callCtx, cancel := context.WithTimeout(grpcauthctx.WithParticipantID(ctx, participantID), 3*time.Second)
	defer cancel()

	resp, err := a.communicationClient.OpenCommunicationGate(callCtx, &statev1.OpenCommunicationGateRequest{
		CampaignId: campaignID,
		GateType:   strings.TrimSpace(gateType),
		Reason:     strings.TrimSpace(reason),
		Metadata:   requestMetadata,
	})
	if err != nil {
		return communicationContext{}, fmt.Errorf("open communication gate: %w", err)
	}
	return communicationContextFromProto(campaignID, grpcauthctx.UserIDFromOutgoingContext(callCtx), resp.GetContext())
}

func (a *campaignAuthorizer) ResolveCommunicationGate(ctx context.Context, campaignID string, participantID string, decision string, resolution map[string]any) (communicationContext, error) {
	if a == nil || a.communicationClient == nil {
		return communicationContext{}, errors.New("communication client is not configured")
	}
	resolutionStruct, err := mapToStruct(resolution)
	if err != nil {
		return communicationContext{}, fmt.Errorf("encode communication gate resolution: %w", err)
	}
	callCtx, cancel := context.WithTimeout(grpcauthctx.WithParticipantID(ctx, participantID), 3*time.Second)
	defer cancel()

	resp, err := a.communicationClient.ResolveCommunicationGate(callCtx, &statev1.ResolveCommunicationGateRequest{
		CampaignId: campaignID,
		Decision:   strings.TrimSpace(decision),
		Resolution: resolutionStruct,
	})
	if err != nil {
		return communicationContext{}, fmt.Errorf("resolve communication gate: %w", err)
	}
	return communicationContextFromProto(campaignID, grpcauthctx.UserIDFromOutgoingContext(callCtx), resp.GetContext())
}

func (a *campaignAuthorizer) RespondToCommunicationGate(ctx context.Context, campaignID string, participantID string, decision string, response map[string]any) (communicationContext, error) {
	if a == nil || a.communicationClient == nil {
		return communicationContext{}, errors.New("communication client is not configured")
	}
	responseStruct, err := mapToStruct(response)
	if err != nil {
		return communicationContext{}, fmt.Errorf("encode communication gate response: %w", err)
	}
	callCtx, cancel := context.WithTimeout(grpcauthctx.WithParticipantID(ctx, participantID), 3*time.Second)
	defer cancel()

	resp, err := a.communicationClient.RespondToCommunicationGate(callCtx, &statev1.RespondToCommunicationGateRequest{
		CampaignId: campaignID,
		Decision:   strings.TrimSpace(decision),
		Response:   responseStruct,
	})
	if err != nil {
		return communicationContext{}, fmt.Errorf("respond to communication gate: %w", err)
	}
	return communicationContextFromProto(campaignID, grpcauthctx.UserIDFromOutgoingContext(callCtx), resp.GetContext())
}

func (a *campaignAuthorizer) AbandonCommunicationGate(ctx context.Context, campaignID string, participantID string, reason string) (communicationContext, error) {
	if a == nil || a.communicationClient == nil {
		return communicationContext{}, errors.New("communication client is not configured")
	}
	callCtx, cancel := context.WithTimeout(grpcauthctx.WithParticipantID(ctx, participantID), 3*time.Second)
	defer cancel()

	resp, err := a.communicationClient.AbandonCommunicationGate(callCtx, &statev1.AbandonCommunicationGateRequest{
		CampaignId: campaignID,
		Reason:     strings.TrimSpace(reason),
	})
	if err != nil {
		return communicationContext{}, fmt.Errorf("abandon communication gate: %w", err)
	}
	return communicationContextFromProto(campaignID, grpcauthctx.UserIDFromOutgoingContext(callCtx), resp.GetContext())
}

func (a *campaignAuthorizer) RequestGMHandoff(ctx context.Context, campaignID string, participantID string, reason string, controlMetadata map[string]any) (communicationContext, error) {
	if a == nil || a.communicationClient == nil {
		return communicationContext{}, errors.New("communication client is not configured")
	}
	requestMetadata, err := mapToStruct(controlMetadata)
	if err != nil {
		return communicationContext{}, fmt.Errorf("encode gm handoff metadata: %w", err)
	}
	callCtx, cancel := context.WithTimeout(grpcauthctx.WithParticipantID(ctx, participantID), 3*time.Second)
	defer cancel()

	resp, err := a.communicationClient.RequestGMHandoff(callCtx, &statev1.RequestGMHandoffRequest{
		CampaignId: campaignID,
		Reason:     strings.TrimSpace(reason),
		Metadata:   requestMetadata,
	})
	if err != nil {
		return communicationContext{}, fmt.Errorf("request gm handoff: %w", err)
	}
	return communicationContextFromProto(campaignID, grpcauthctx.UserIDFromOutgoingContext(callCtx), resp.GetContext())
}

func (a *campaignAuthorizer) ResolveGMHandoff(ctx context.Context, campaignID string, participantID string, decision string, resolution map[string]any) (communicationContext, error) {
	if a == nil || a.communicationClient == nil {
		return communicationContext{}, errors.New("communication client is not configured")
	}
	resolutionStruct, err := mapToStruct(resolution)
	if err != nil {
		return communicationContext{}, fmt.Errorf("encode gm handoff resolution: %w", err)
	}
	callCtx, cancel := context.WithTimeout(grpcauthctx.WithParticipantID(ctx, participantID), 3*time.Second)
	defer cancel()

	resp, err := a.communicationClient.ResolveGMHandoff(callCtx, &statev1.ResolveGMHandoffRequest{
		CampaignId: campaignID,
		Decision:   strings.TrimSpace(decision),
		Resolution: resolutionStruct,
	})
	if err != nil {
		return communicationContext{}, fmt.Errorf("resolve gm handoff: %w", err)
	}
	return communicationContextFromProto(campaignID, grpcauthctx.UserIDFromOutgoingContext(callCtx), resp.GetContext())
}

func (a *campaignAuthorizer) AbandonGMHandoff(ctx context.Context, campaignID string, participantID string, reason string) (communicationContext, error) {
	if a == nil || a.communicationClient == nil {
		return communicationContext{}, errors.New("communication client is not configured")
	}
	callCtx, cancel := context.WithTimeout(grpcauthctx.WithParticipantID(ctx, participantID), 3*time.Second)
	defer cancel()

	resp, err := a.communicationClient.AbandonGMHandoff(callCtx, &statev1.AbandonGMHandoffRequest{
		CampaignId: campaignID,
		Reason:     strings.TrimSpace(reason),
	})
	if err != nil {
		return communicationContext{}, fmt.Errorf("abandon gm handoff: %w", err)
	}
	return communicationContextFromProto(campaignID, grpcauthctx.UserIDFromOutgoingContext(callCtx), resp.GetContext())
}

func communicationContextFromProto(campaignID string, userID string, contextState *statev1.CommunicationContext) (communicationContext, error) {
	if contextState == nil {
		return communicationContext{}, errors.New("communication context response is empty")
	}

	welcome := joinWelcome{
		ParticipantName: strings.TrimSpace(contextState.GetParticipant().GetName()),
		CampaignName:    strings.TrimSpace(contextState.GetCampaignName()),
		GmMode:          contextState.GetGmMode().String(),
		AIAgentID:       strings.TrimSpace(contextState.GetAiAgentId()),
		Locale:          contextState.GetLocale(),
	}
	if sessionState := contextState.GetActiveSession(); sessionState != nil {
		welcome.SessionID = strings.TrimSpace(sessionState.GetSessionId())
		welcome.SessionName = strings.TrimSpace(sessionState.GetName())
	}
	if welcome.ParticipantName == "" {
		welcome.ParticipantName = userID
	}
	if welcome.CampaignName == "" {
		welcome.CampaignName = campaignID
	}
	if welcome.Locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		welcome.Locale = commonv1.Locale_LOCALE_EN_US
	}

	resolved := communicationContext{
		Welcome:                welcome,
		ParticipantID:          strings.TrimSpace(contextState.GetParticipant().GetParticipantId()),
		DefaultStreamID:        strings.TrimSpace(contextState.GetDefaultStreamId()),
		DefaultPersonaID:       strings.TrimSpace(contextState.GetDefaultPersonaId()),
		ActiveSessionGate:      communicationSessionGateJSON(contextState.GetActiveSessionGate()),
		ActiveSessionSpotlight: communicationSessionSpotlightJSON(contextState.GetActiveSessionSpotlight()),
		Streams:                make([]chatStream, 0, len(contextState.GetStreams())),
		Personas:               make([]chatPersona, 0, len(contextState.GetPersonas())),
	}
	for _, stream := range contextState.GetStreams() {
		if stream == nil {
			continue
		}
		resolved.Streams = append(resolved.Streams, chatStream{
			StreamID:  strings.TrimSpace(stream.GetStreamId()),
			Kind:      communicationStreamKindJSON(stream.GetKind()),
			Scope:     communicationStreamScopeJSON(stream.GetScope()),
			SessionID: strings.TrimSpace(stream.GetSessionId()),
			SceneID:   strings.TrimSpace(stream.GetSceneId()),
			Label:     strings.TrimSpace(stream.GetLabel()),
		})
	}
	for _, persona := range contextState.GetPersonas() {
		if persona == nil {
			continue
		}
		resolved.Personas = append(resolved.Personas, chatPersona{
			PersonaID:     strings.TrimSpace(persona.GetPersonaId()),
			Kind:          communicationPersonaKindJSON(persona.GetKind()),
			ParticipantID: strings.TrimSpace(persona.GetParticipantId()),
			CharacterID:   strings.TrimSpace(persona.GetCharacterId()),
			DisplayName:   strings.TrimSpace(persona.GetDisplayName()),
		})
	}
	return resolved, nil
}

func mapToStruct(values map[string]any) (*structpb.Struct, error) {
	if len(values) == 0 {
		return nil, nil
	}
	return structpb.NewStruct(values)
}

func (a *campaignAuthorizer) findActiveSession(ctx context.Context, campaignID string, userID string) (*statev1.Session, error) {
	if a == nil || a.sessionClient == nil {
		return nil, errors.New("session client is not configured")
	}
	pageToken := ""
	for {
		callCtx, cancel := context.WithTimeout(grpcauthctx.WithUserID(ctx, userID), 3*time.Second)
		resp, err := a.sessionClient.ListSessions(callCtx, &statev1.ListSessionsRequest{
			CampaignId: campaignID,
			PageSize:   10,
			PageToken:  pageToken,
		})
		cancel()
		if err != nil {
			return nil, fmt.Errorf("list campaign sessions: %w", err)
		}
		for _, session := range resp.GetSessions() {
			if session.GetStatus() == statev1.SessionStatus_SESSION_ACTIVE {
				return session, nil
			}
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}
	return nil, errCampaignSessionInactive
}

func (a *campaignAuthorizer) findParticipantByUserID(ctx context.Context, campaignID string, userID string) (*statev1.Participant, error) {
	if a == nil || a.participantClient == nil {
		return nil, errors.New("participant client is not configured")
	}
	pageToken := ""
	for {
		callCtx, cancel := context.WithTimeout(grpcauthctx.WithUserID(ctx, userID), 3*time.Second)
		resp, err := a.participantClient.ListParticipants(callCtx, &statev1.ListParticipantsRequest{
			CampaignId: campaignID,
			PageSize:   100,
			PageToken:  pageToken,
		})
		cancel()
		if err != nil {
			return nil, fmt.Errorf("list campaign participants: %w", err)
		}
		for _, participant := range resp.GetParticipants() {
			if strings.TrimSpace(participant.GetUserId()) == userID {
				return participant, nil
			}
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}
	return nil, nil
}

func communicationStreamKindJSON(kind statev1.CommunicationStreamKind) string {
	switch kind {
	case statev1.CommunicationStreamKind_COMMUNICATION_STREAM_KIND_SYSTEM:
		return "system"
	case statev1.CommunicationStreamKind_COMMUNICATION_STREAM_KIND_CHARACTER:
		return "character"
	case statev1.CommunicationStreamKind_COMMUNICATION_STREAM_KIND_CONTROL:
		return "control"
	default:
		return "table"
	}
}

func communicationStreamScopeJSON(scope statev1.CommunicationStreamScope) string {
	switch scope {
	case statev1.CommunicationStreamScope_COMMUNICATION_STREAM_SCOPE_CAMPAIGN:
		return "campaign"
	case statev1.CommunicationStreamScope_COMMUNICATION_STREAM_SCOPE_SCENE:
		return "scene"
	default:
		return "session"
	}
}

func communicationSessionGateJSON(gate *statev1.SessionGate) *chatSessionGate {
	if gate == nil {
		return nil
	}
	result := &chatSessionGate{
		GateID:   strings.TrimSpace(gate.GetId()),
		GateType: strings.TrimSpace(gate.GetType()),
		Status:   communicationSessionGateStatusJSON(gate.GetStatus()),
		Reason:   strings.TrimSpace(gate.GetReason()),
	}
	if metadata := gate.GetMetadata(); metadata != nil {
		result.Metadata = metadata.AsMap()
	}
	if progress := gate.GetProgress(); progress != nil {
		result.Progress = progress.AsMap()
	}
	return result
}

func communicationSessionSpotlightJSON(spotlight *statev1.SessionSpotlight) *chatSessionSpotlight {
	if spotlight == nil {
		return nil
	}
	return &chatSessionSpotlight{
		Type:        communicationSessionSpotlightTypeJSON(spotlight.GetType()),
		CharacterID: strings.TrimSpace(spotlight.GetCharacterId()),
	}
}

func communicationSessionGateStatusJSON(status statev1.SessionGateStatus) string {
	switch status {
	case statev1.SessionGateStatus_SESSION_GATE_OPEN:
		return "open"
	case statev1.SessionGateStatus_SESSION_GATE_RESOLVED:
		return "resolved"
	case statev1.SessionGateStatus_SESSION_GATE_ABANDONED:
		return "abandoned"
	default:
		return "unspecified"
	}
}

func communicationSessionSpotlightTypeJSON(spotlightType statev1.SessionSpotlightType) string {
	switch spotlightType {
	case statev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM:
		return "gm"
	case statev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_CHARACTER:
		return "character"
	default:
		return "unspecified"
	}
}

func communicationPersonaKindJSON(kind statev1.CommunicationPersonaKind) string {
	switch kind {
	case statev1.CommunicationPersonaKind_COMMUNICATION_PERSONA_KIND_CHARACTER:
		return "character"
	default:
		return "participant"
	}
}
