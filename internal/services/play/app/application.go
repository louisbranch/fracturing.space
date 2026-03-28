package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/imagecdn"
	playprotocol "github.com/louisbranch/fracturing.space/internal/services/play/protocol"
	playdaggerheart "github.com/louisbranch/fracturing.space/internal/services/play/protocol/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
)

// playApplication owns browser-facing state assembly so transport handlers and
// realtime orchestration can reuse one application seam.
type playApplication struct {
	deps         Dependencies
	logger       *slog.Logger
	assetBaseURL string
}

type characterSheetResult struct {
	charID string
	char   *gamev1.Character
	resp   *gamev1.GetCharacterSheetResponse
}

func (s *Server) application() playApplication {
	return playApplication{
		deps:         s.deps,
		logger:       s.logger,
		assetBaseURL: s.assetBaseURL,
	}
}

func (a playApplication) log() *slog.Logger {
	return loggerOrDefault(a.logger)
}

func (a playApplication) bootstrap(ctx context.Context, req playRequest) (playprotocol.Bootstrap, error) {
	state, err := a.interactionState(ctx, req)
	if err != nil {
		return playprotocol.Bootstrap{}, err
	}
	system, err := a.systemMetadata(ctx, req)
	if err != nil {
		return playprotocol.Bootstrap{}, err
	}
	chat, err := a.recentChatSnapshot(ctx, req.CampaignID, state)
	if err != nil {
		return playprotocol.Bootstrap{}, err
	}
	participants, catalog := a.enrichedData(ctx, req, state.GetLocale())
	a.log().InfoContext(ctx, "play: bootstrap assembled",
		"campaign_id", strings.TrimSpace(req.CampaignID),
		"user_id", strings.TrimSpace(req.UserID),
		"participants", len(participants),
		"character_catalog_entries", len(catalog),
		"active_session_id", strings.TrimSpace(state.GetActiveSession().GetSessionId()),
	)
	return playprotocol.Bootstrap{
		CampaignID:                 strings.TrimSpace(req.CampaignID),
		AIDebugEnabled:             true,
		Viewer:                     playprotocol.ViewerFromGameViewer(state.GetViewer()),
		System:                     system,
		InteractionState:           playprotocol.InteractionStateFromGameState(state),
		Participants:               participants,
		CharacterInspectionCatalog: catalog,
		Chat:                       chat,
		Realtime: playprotocol.RealtimeConfig{
			URL:             "/realtime",
			ProtocolVersion: playprotocol.RealtimeProtocolVersion,
			TypingTTLMs:     int(defaultTypingTTL.Milliseconds()),
		},
		TransitionSFX: a.transitionSFX(),
	}, nil
}

// transitionSFX resolves SFX URLs for scene and interaction transitions from the
// embedded asset catalog. Returns nil when neither asset resolves.
func (a playApplication) transitionSFX() *playprotocol.TransitionSFX {
	const setID = "interface_sound_effect_set_v1"
	sceneURL := audioAssetURL(a.assetBaseURL, setID, "scene_transition")
	interactionURL := audioAssetURL(a.assetBaseURL, setID, "scene_interaction_transition")
	if sceneURL == "" && interactionURL == "" {
		return nil
	}
	return &playprotocol.TransitionSFX{
		SceneChangeURL:       sceneURL,
		InteractionChangeURL: interactionURL,
	}
}

// audioAssetURL resolves a Cloudinary audio URL from the embedded catalog.
// Cloudinary serves audio under /video/upload/ rather than /image/upload/.
func audioAssetURL(imageBaseURL, setID, assetID string) string {
	versionedPublicID := catalog.ResolveCDNAssetID(setID, assetID)
	if versionedPublicID == assetID {
		// No catalog entry found — ResolveCDNAssetID fell back to the raw asset ID.
		return ""
	}
	audioBaseURL := strings.Replace(imageBaseURL, "/image/upload", "/video/upload", 1)
	u, err := imagecdn.New(audioBaseURL).URL(imagecdn.Request{
		AssetID:   versionedPublicID,
		Extension: ".mp3",
	})
	if err != nil {
		return ""
	}
	return u
}

func (a playApplication) aiDebugTurns(ctx context.Context, req playRequest, page aiDebugPage) (playprotocol.AIDebugTurnsPage, error) {
	state, err := a.interactionState(ctx, req)
	if err != nil {
		return playprotocol.AIDebugTurnsPage{}, err
	}
	sessionID := strings.TrimSpace(state.GetActiveSession().GetSessionId())
	if sessionID == "" {
		return playprotocol.AIDebugTurnsPage{Turns: []playprotocol.AIDebugTurnSummary{}}, nil
	}
	resp, err := a.deps.AIDebug.ListCampaignDebugTurns(req.authContext(ctx), &aiv1.ListCampaignDebugTurnsRequest{
		CampaignId: req.CampaignID,
		SessionId:  sessionID,
		PageSize:   int32(page.PageSize),
		PageToken:  page.PageToken,
	})
	if err != nil {
		return playprotocol.AIDebugTurnsPage{}, err
	}
	return playprotocol.AIDebugTurnsPageFromProto(resp), nil
}

func (a playApplication) aiDebugTurn(ctx context.Context, req playRequest, turnID string) (playprotocol.AIDebugTurn, error) {
	resp, err := a.deps.AIDebug.GetCampaignDebugTurn(req.authContext(ctx), &aiv1.GetCampaignDebugTurnRequest{
		CampaignId: req.CampaignID,
		TurnId:     strings.TrimSpace(turnID),
	})
	if err != nil {
		return playprotocol.AIDebugTurn{}, err
	}
	return playprotocol.AIDebugTurnFromProto(resp.GetTurn()), nil
}

func (a playApplication) interactionResponse(ctx context.Context, req playRequest, state *gamev1.InteractionState) (playprotocol.RoomSnapshot, error) {
	return a.roomSnapshotFromState(ctx, req, state, 0)
}

func (a playApplication) roomSnapshotFromState(ctx context.Context, req playRequest, state *gamev1.InteractionState, latestGameSeq uint64) (playprotocol.RoomSnapshot, error) {
	campaignID := strings.TrimSpace(req.CampaignID)
	if campaignID == "" {
		campaignID = strings.TrimSpace(state.GetCampaignId())
	}
	chat, err := a.chatCursor(ctx, campaignID, state)
	if err != nil {
		return playprotocol.RoomSnapshot{}, err
	}
	participants, catalog := a.enrichedData(req.authContext(ctx), playRequest{
		campaignRequest: campaignRequest{CampaignID: campaignID},
		UserID:          req.UserID,
	}, state.GetLocale())
	a.log().InfoContext(ctx, "play: room snapshot assembled",
		"campaign_id", campaignID,
		"user_id", strings.TrimSpace(req.UserID),
		"participants", len(participants),
		"character_catalog_entries", len(catalog),
		"latest_game_seq", latestGameSeq,
		"active_session_id", strings.TrimSpace(state.GetActiveSession().GetSessionId()),
	)
	return playprotocol.RoomSnapshot{
		InteractionState:           playprotocol.InteractionStateFromGameState(state),
		Participants:               participants,
		CharacterInspectionCatalog: catalog,
		Chat:                       chat,
		LatestGameSeq:              latestGameSeq,
	}, nil
}

func (a playApplication) history(ctx context.Context, req playRequest, page chatHistoryPage) (playprotocol.HistoryResponse, error) {
	state, err := a.interactionState(ctx, req)
	if err != nil {
		return playprotocol.HistoryResponse{}, err
	}
	sessionID := strings.TrimSpace(state.GetActiveSession().GetSessionId())
	if sessionID == "" {
		return playprotocol.HistoryResponse{SessionID: "", Messages: []playprotocol.ChatMessage{}}, nil
	}
	messages, err := a.deps.Transcripts.HistoryBefore(ctx, transcript.HistoryBeforeQuery{
		Scope: transcript.Scope{
			CampaignID: req.CampaignID,
			SessionID:  sessionID,
		},
		BeforeSequenceID: page.BeforeSequenceID,
		Limit:            page.Limit,
	})
	if err != nil {
		return playprotocol.HistoryResponse{}, errChatHistoryUnavailable
	}
	return playprotocol.HistoryResponse{
		SessionID: sessionID,
		Messages:  playprotocol.TranscriptMessages(messages),
	}, nil
}

func (a playApplication) incrementalChatMessages(ctx context.Context, scope transcript.Scope, afterSeq int64) ([]playprotocol.ChatMessage, error) {
	messages, err := a.deps.Transcripts.HistoryAfter(ctx, transcript.HistoryAfterQuery{
		Scope:           scope,
		AfterSequenceID: afterSeq,
	})
	if err != nil {
		return nil, fmt.Errorf("load transcript messages after sequence: %w", err)
	}
	return playprotocol.TranscriptMessages(messages), nil
}

func (a playApplication) interactionState(ctx context.Context, req playRequest) (*gamev1.InteractionState, error) {
	resp, err := a.deps.Interaction.GetInteractionState(req.authContext(ctx), &gamev1.GetInteractionStateRequest{CampaignId: req.CampaignID})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.GetState() == nil {
		return nil, errors.New("interaction state response was empty")
	}
	return resp.GetState(), nil
}

func (a playApplication) systemMetadata(ctx context.Context, req playRequest) (playprotocol.System, error) {
	resp, err := a.deps.Campaign.GetCampaign(req.authContext(ctx), &gamev1.GetCampaignRequest{CampaignId: req.CampaignID})
	if err != nil {
		return playprotocol.System{}, err
	}
	campaign := resp.GetCampaign()
	if campaign == nil || campaign.GetSystem() == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		return playprotocol.System{}, nil
	}
	system := playprotocol.System{ID: gameSystemIDString(campaign.GetSystem())}
	infoResp, err := a.deps.System.GetGameSystem(req.authContext(ctx), &gamev1.GetGameSystemRequest{Id: campaign.GetSystem()})
	if err != nil {
		return playprotocol.System{}, err
	}
	if info := infoResp.GetSystem(); info != nil {
		system.Name = strings.TrimSpace(info.GetName())
		system.Version = strings.TrimSpace(info.GetVersion())
	}
	if system.Name == "" {
		system.Name = system.ID
	}
	return system, nil
}

func (a playApplication) recentChatSnapshot(ctx context.Context, campaignID string, state *gamev1.InteractionState) (playprotocol.ChatSnapshot, error) {
	historyURL := pathForCampaignAPI(campaignID, "chat/history")
	sessionID := strings.TrimSpace(state.GetActiveSession().GetSessionId())
	if sessionID == "" {
		return playprotocol.ChatSnapshot{SessionID: "", LatestSequenceID: 0, Messages: []playprotocol.ChatMessage{}, HistoryURL: historyURL}, nil
	}
	scope := transcript.Scope{CampaignID: campaignID, SessionID: sessionID}
	latest, err := a.deps.Transcripts.LatestSequence(ctx, scope)
	if err != nil {
		return playprotocol.ChatSnapshot{}, fmt.Errorf("load latest transcript sequence: %w", err)
	}
	messages, err := a.deps.Transcripts.HistoryBefore(ctx, transcript.HistoryBeforeQuery{
		Scope:            scope,
		BeforeSequenceID: latest + 1,
		Limit:            transcript.DefaultHistoryLimit,
	})
	if err != nil {
		return playprotocol.ChatSnapshot{}, fmt.Errorf("load recent transcript history: %w", err)
	}
	return playprotocol.ChatSnapshot{
		SessionID:        sessionID,
		LatestSequenceID: latest,
		Messages:         playprotocol.TranscriptMessages(messages),
		HistoryURL:       historyURL,
	}, nil
}

func (a playApplication) chatCursor(ctx context.Context, campaignID string, state *gamev1.InteractionState) (playprotocol.ChatSnapshot, error) {
	historyURL := pathForCampaignAPI(campaignID, "chat/history")
	sessionID := strings.TrimSpace(state.GetActiveSession().GetSessionId())
	if sessionID == "" {
		return playprotocol.ChatSnapshot{SessionID: "", LatestSequenceID: 0, Messages: []playprotocol.ChatMessage{}, HistoryURL: historyURL}, nil
	}
	latest, err := a.deps.Transcripts.LatestSequence(ctx, transcript.Scope{CampaignID: campaignID, SessionID: sessionID})
	if err != nil {
		return playprotocol.ChatSnapshot{}, fmt.Errorf("load latest transcript sequence: %w", err)
	}
	return playprotocol.ChatSnapshot{
		SessionID:        sessionID,
		LatestSequenceID: latest,
		Messages:         []playprotocol.ChatMessage{},
		HistoryURL:       historyURL,
	}, nil
}

func gameSystemIDString(value commonv1.GameSystem) string {
	name := strings.TrimSpace(value.String())
	if name == "" {
		return ""
	}
	name = strings.TrimPrefix(name, "GAME_SYSTEM_")
	return strings.ToLower(name)
}

// enrichedData loads participant and character data for the bootstrap response
// which has an authenticated playRequest available.
func (a playApplication) enrichedData(ctx context.Context, req playRequest, locale commonv1.Locale) ([]playprotocol.Participant, map[string]playprotocol.CharacterInspection) {
	return a.enrichedDataForCampaign(req.authContext(ctx), req.CampaignID, locale)
}

// enrichedDataForCampaign loads participant and character data using a raw
// context and campaign ID — used by room snapshot paths where playRequest is
// not always available.
func (a playApplication) enrichedDataForCampaign(ctx context.Context, campaignID string, locale commonv1.Locale) ([]playprotocol.Participant, map[string]playprotocol.CharacterInspection) {
	participants := a.listAllParticipants(ctx, campaignID)
	characters := a.listAllCharacters(ctx, campaignID)

	// Associate characters with participants via persistent owner_participant_id.
	charsByParticipant := map[string][]string{}
	for _, c := range characters {
		pid := strings.TrimSpace(c.GetOwnerParticipantId().GetValue())
		if pid != "" {
			charsByParticipant[pid] = append(charsByParticipant[pid], strings.TrimSpace(c.GetId()))
		}
	}
	for i := range participants {
		participants[i].CharacterIDs = charsByParticipant[participants[i].ID]
	}

	catalog := a.buildCharacterInspectionCatalog(ctx, campaignID, locale, characters)
	return participants, catalog
}

const enrichmentPageSize = 10

// listAllParticipants paginates through all participants in a campaign.
// Returns partial results on pagination error — interaction state is the
// primary data, enrichment is supplementary.
func (a playApplication) listAllParticipants(ctx context.Context, campaignID string) []playprotocol.Participant {
	var all []playprotocol.Participant
	pageToken := ""
	for {
		resp, err := a.deps.Participants.ListParticipants(ctx, &gamev1.ListParticipantsRequest{
			CampaignId: campaignID,
			PageSize:   enrichmentPageSize,
			PageToken:  pageToken,
		})
		if err != nil {
			a.log().WarnContext(ctx, "play: participant pagination truncated", "campaign_id", campaignID, "collected", len(all), "error", err)
			return all
		}
		for _, p := range resp.GetParticipants() {
			all = append(all, playprotocol.ParticipantFromGameParticipant(a.assetBaseURL, p))
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}
	return all
}

// listAllCharacters paginates through all characters in a campaign.
// Returns partial results on pagination error — interaction state is the
// primary data, enrichment is supplementary.
func (a playApplication) listAllCharacters(ctx context.Context, campaignID string) []*gamev1.Character {
	var all []*gamev1.Character
	pageToken := ""
	for {
		resp, err := a.deps.Characters.ListCharacters(ctx, &gamev1.ListCharactersRequest{
			CampaignId: campaignID,
			PageSize:   enrichmentPageSize,
			PageToken:  pageToken,
		})
		if err != nil {
			a.log().WarnContext(ctx, "play: character pagination truncated", "campaign_id", campaignID, "collected", len(all), "error", err)
			return all
		}
		all = append(all, resp.GetCharacters()...)
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}
	return all
}

// buildCharacterInspectionCatalog fetches character sheets in parallel and maps
// them via the Daggerheart protocol functions.
func (a playApplication) buildCharacterInspectionCatalog(ctx context.Context, campaignID string, locale commonv1.Locale, characters []*gamev1.Character) map[string]playprotocol.CharacterInspection {
	if len(characters) == 0 {
		return nil
	}

	var (
		mu      sync.Mutex
		results []characterSheetResult
	)
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(10)
	for _, char := range characters {
		charID := strings.TrimSpace(char.GetId())
		if charID == "" {
			continue
		}
		c := char
		g.Go(func() error {
			resp, err := a.deps.Characters.GetCharacterSheet(gCtx, &gamev1.GetCharacterSheetRequest{
				CampaignId:  campaignID,
				CharacterId: charID,
			})
			if err != nil {
				a.log().WarnContext(gCtx, "play: character sheet enrichment skipped", "character_id", charID, "error", err)
				return nil // best-effort enrichment — skip failures
			}
			mu.Lock()
			results = append(results, characterSheetResult{charID: charID, char: c, resp: resp})
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()

	domainCards := a.loadDaggerheartDomainCards(ctx, locale, results)
	catalog := make(map[string]playprotocol.CharacterInspection, len(results))
	for _, r := range results {
		dhProfile := r.resp.GetProfile().GetDaggerheart()
		dhState := r.resp.GetState().GetDaggerheart()
		if dhProfile == nil && dhState == nil {
			continue
		}
		catalog[r.charID] = playprotocol.CharacterInspection{
			System: playdaggerheart.SystemID,
			Card:   playdaggerheart.CardFromSheet(a.assetBaseURL, r.char, dhProfile, dhState),
			Sheet:  playdaggerheart.SheetFromResponse(a.assetBaseURL, r.char, dhProfile, dhState, domainCards),
		}
	}
	if len(catalog) == 0 {
		return nil
	}
	return catalog
}

// loadDaggerheartDomainCards resolves unique domain-card ids into browser-ready
// content so the sheet can render full card text without client-side fetches.
func (a playApplication) loadDaggerheartDomainCards(
	ctx context.Context,
	locale commonv1.Locale,
	results []characterSheetResult,
) map[string]playdaggerheart.DomainCard {
	if a.deps.DaggerheartContent == nil {
		return nil
	}

	uniqueIDs := make(map[string]struct{})
	for _, result := range results {
		profile := result.resp.GetProfile().GetDaggerheart()
		if profile == nil {
			continue
		}
		for _, id := range profile.GetDomainCardIds() {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				uniqueIDs[trimmed] = struct{}{}
			}
		}
	}
	if len(uniqueIDs) == 0 {
		return nil
	}

	var (
		mu    sync.Mutex
		cards = make(map[string]playdaggerheart.DomainCard, len(uniqueIDs))
	)
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(10)
	for id := range uniqueIDs {
		cardID := id
		g.Go(func() error {
			resp, err := a.deps.DaggerheartContent.GetDomainCard(gCtx, &daggerheartv1.GetDaggerheartDomainCardRequest{
				Id:     cardID,
				Locale: locale,
			})
			if err != nil {
				a.log().WarnContext(gCtx, "play: daggerheart domain card enrichment skipped", "card_id", cardID, "error", err)
				return nil
			}
			card := playdaggerheart.DomainCardFromContent(resp.GetDomainCard())
			if card.Name == "" {
				a.log().WarnContext(gCtx, "play: daggerheart domain card enrichment returned empty card", "card_id", cardID)
				return nil
			}
			if card.ID == "" {
				card.ID = cardID
			}

			mu.Lock()
			cards[cardID] = card
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()
	if len(cards) == 0 {
		return nil
	}
	return cards
}
