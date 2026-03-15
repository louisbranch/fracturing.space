package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	playprotocol "github.com/louisbranch/fracturing.space/internal/services/play/protocol"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
)

// playApplication owns browser-facing state assembly so transport handlers and
// realtime orchestration can reuse one application seam.
type playApplication struct {
	interaction interactionClient
	campaign    campaignClient
	system      systemClient
	transcripts transcript.Store
}

func (s *Server) application() playApplication {
	return playApplication{
		interaction: s.interaction,
		campaign:    s.campaign,
		system:      s.system,
		transcripts: s.transcripts,
	}
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
	return playprotocol.Bootstrap{
		CampaignID:       strings.TrimSpace(req.CampaignID),
		Viewer:           state.GetViewer(),
		System:           system,
		InteractionState: state,
		Chat:             chat,
		Realtime: playprotocol.RealtimeConfig{
			URL:             "/realtime",
			ProtocolVersion: playprotocol.RealtimeProtocolVersion,
		},
	}, nil
}

func (a playApplication) roomSnapshot(ctx context.Context, req playRequest, latestGameSeq uint64) (playprotocol.RoomSnapshot, error) {
	state, err := a.interactionState(ctx, req)
	if err != nil {
		return playprotocol.RoomSnapshot{}, err
	}
	return a.roomSnapshotFromState(ctx, req.CampaignID, state, latestGameSeq)
}

func (a playApplication) interactionResponse(ctx context.Context, state *gamev1.InteractionState) (playprotocol.RoomSnapshot, error) {
	return a.roomSnapshotFromState(ctx, strings.TrimSpace(state.GetCampaignId()), state, 0)
}

func (a playApplication) roomSnapshotFromState(ctx context.Context, campaignID string, state *gamev1.InteractionState, latestGameSeq uint64) (playprotocol.RoomSnapshot, error) {
	chat, err := a.chatCursor(ctx, campaignID, state)
	if err != nil {
		return playprotocol.RoomSnapshot{}, err
	}
	return playprotocol.RoomSnapshot{
		InteractionState: state,
		Chat:             chat,
		LatestGameSeq:    latestGameSeq,
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
	messages, err := a.transcripts.HistoryBefore(ctx, transcript.HistoryBeforeQuery{
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
	messages, err := a.transcripts.HistoryAfter(ctx, transcript.HistoryAfterQuery{
		Scope:           scope,
		AfterSequenceID: afterSeq,
	})
	if err != nil {
		return nil, fmt.Errorf("load transcript messages after sequence: %w", err)
	}
	return playprotocol.TranscriptMessages(messages), nil
}

func (a playApplication) interactionState(ctx context.Context, req playRequest) (*gamev1.InteractionState, error) {
	resp, err := a.interaction.GetInteractionState(req.authContext(ctx), &gamev1.GetInteractionStateRequest{CampaignId: req.CampaignID})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.GetState() == nil {
		return nil, errors.New("interaction state response was empty")
	}
	return resp.GetState(), nil
}

func (a playApplication) systemMetadata(ctx context.Context, req playRequest) (playprotocol.System, error) {
	resp, err := a.campaign.GetCampaign(req.authContext(ctx), &gamev1.GetCampaignRequest{CampaignId: req.CampaignID})
	if err != nil {
		return playprotocol.System{}, err
	}
	campaign := resp.GetCampaign()
	if campaign == nil || campaign.GetSystem() == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		return playprotocol.System{}, nil
	}
	system := playprotocol.System{ID: gameSystemIDString(campaign.GetSystem())}
	infoResp, err := a.system.GetGameSystem(req.authContext(ctx), &gamev1.GetGameSystemRequest{Id: campaign.GetSystem()})
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
	latest, err := a.transcripts.LatestSequence(ctx, scope)
	if err != nil {
		return playprotocol.ChatSnapshot{}, fmt.Errorf("load latest transcript sequence: %w", err)
	}
	messages, err := a.transcripts.HistoryBefore(ctx, transcript.HistoryBeforeQuery{
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
	latest, err := a.transcripts.LatestSequence(ctx, transcript.Scope{CampaignID: campaignID, SessionID: sessionID})
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
