package app

import (
	"context"
	"net/http"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	gogrpc "google.golang.org/grpc"
)

type interactionRoute struct {
	pattern string
	handler http.Handler
}

func (r interactionRoute) register(rootMux *http.ServeMux) {
	rootMux.Handle(r.pattern, r.handler)
}

// interactionRoutes is the single index of browser-facing interaction
// mutations so contributors can see the full surface in one place.
func interactionRoutes(server *Server) []interactionRoute {
	return []interactionRoute{
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/set-active-scene", func() *gamev1.SetActiveSceneRequest {
			return &gamev1.SetActiveSceneRequest{}
		}, func(req *gamev1.SetActiveSceneRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.SetActiveScene),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/start-scene-player-phase", func() *gamev1.StartScenePlayerPhaseRequest {
			return &gamev1.StartScenePlayerPhaseRequest{}
		}, func(req *gamev1.StartScenePlayerPhaseRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.StartScenePlayerPhase),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/submit-scene-player-post", func() *gamev1.SubmitScenePlayerPostRequest {
			return &gamev1.SubmitScenePlayerPostRequest{}
		}, func(req *gamev1.SubmitScenePlayerPostRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.SubmitScenePlayerPost),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/yield-scene-player-phase", func() *gamev1.YieldScenePlayerPhaseRequest {
			return &gamev1.YieldScenePlayerPhaseRequest{}
		}, func(req *gamev1.YieldScenePlayerPhaseRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.YieldScenePlayerPhase),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/unyield-scene-player-phase", func() *gamev1.UnyieldScenePlayerPhaseRequest {
			return &gamev1.UnyieldScenePlayerPhaseRequest{}
		}, func(req *gamev1.UnyieldScenePlayerPhaseRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.UnyieldScenePlayerPhase),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/end-scene-player-phase", func() *gamev1.EndScenePlayerPhaseRequest {
			return &gamev1.EndScenePlayerPhaseRequest{}
		}, func(req *gamev1.EndScenePlayerPhaseRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.EndScenePlayerPhase),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/commit-scene-gm-output", func() *gamev1.CommitSceneGMOutputRequest {
			return &gamev1.CommitSceneGMOutputRequest{}
		}, func(req *gamev1.CommitSceneGMOutputRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.CommitSceneGMOutput),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/accept-scene-player-phase", func() *gamev1.AcceptScenePlayerPhaseRequest {
			return &gamev1.AcceptScenePlayerPhaseRequest{}
		}, func(req *gamev1.AcceptScenePlayerPhaseRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.AcceptScenePlayerPhase),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/request-scene-player-revisions", func() *gamev1.RequestScenePlayerRevisionsRequest {
			return &gamev1.RequestScenePlayerRevisionsRequest{}
		}, func(req *gamev1.RequestScenePlayerRevisionsRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.RequestScenePlayerRevisions),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/resolve-scene-player-phase-review", func() *gamev1.ResolveScenePlayerPhaseReviewRequest {
			return &gamev1.ResolveScenePlayerPhaseReviewRequest{}
		}, func(req *gamev1.ResolveScenePlayerPhaseReviewRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.ResolveScenePlayerPhaseReview),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/pause-session-for-ooc", func() *gamev1.PauseSessionForOOCRequest {
			return &gamev1.PauseSessionForOOCRequest{}
		}, func(req *gamev1.PauseSessionForOOCRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.PauseSessionForOOC),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/post-session-ooc", func() *gamev1.PostSessionOOCRequest {
			return &gamev1.PostSessionOOCRequest{}
		}, func(req *gamev1.PostSessionOOCRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.PostSessionOOC),
		scopeOnlyInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/mark-ooc-ready-to-resume", func(campaignID string) *gamev1.MarkOOCReadyToResumeRequest {
			return &gamev1.MarkOOCReadyToResumeRequest{CampaignId: campaignID}
		}, server.interaction.MarkOOCReadyToResume),
		scopeOnlyInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/clear-ooc-ready-to-resume", func(campaignID string) *gamev1.ClearOOCReadyToResumeRequest {
			return &gamev1.ClearOOCReadyToResumeRequest{CampaignId: campaignID}
		}, server.interaction.ClearOOCReadyToResume),
		scopeOnlyInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/resume-from-ooc", func(campaignID string) *gamev1.ResumeFromOOCRequest {
			return &gamev1.ResumeFromOOCRequest{CampaignId: campaignID}
		}, server.interaction.ResumeFromOOC),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/resolve-interrupted-scene-phase", func() *gamev1.ResolveInterruptedScenePhaseRequest {
			return &gamev1.ResolveInterruptedScenePhaseRequest{}
		}, func(req *gamev1.ResolveInterruptedScenePhaseRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.ResolveInterruptedScenePhase),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/set-session-gm-authority", func() *gamev1.SetSessionGMAuthorityRequest {
			return &gamev1.SetSessionGMAuthorityRequest{}
		}, func(req *gamev1.SetSessionGMAuthorityRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.SetSessionGMAuthority),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/retry-ai-gm-turn", func() *gamev1.RetryAIGMTurnRequest {
			return &gamev1.RetryAIGMTurnRequest{}
		}, func(req *gamev1.RetryAIGMTurnRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.RetryAIGMTurn),
	}
}

func jsonInteractionRoute[TReq any, TResp interactionStateResponse](
	server *Server,
	pattern string,
	newRequest func() *TReq,
	setCampaignID func(*TReq, string),
	call func(context.Context, *TReq, ...gogrpc.CallOption) (TResp, error),
) interactionRoute {
	return interactionRoute{
		pattern: pattern,
		handler: rpcInteractionMutationHandler(server, newRequest, setCampaignID, call),
	}
}

func scopeOnlyInteractionRoute[TReq any, TResp interactionStateResponse](
	server *Server,
	pattern string,
	buildRequest func(string) *TReq,
	call func(context.Context, *TReq, ...gogrpc.CallOption) (TResp, error),
) interactionRoute {
	return interactionRoute{
		pattern: pattern,
		handler: rpcInteractionMutationHandlerWithoutBody(server, buildRequest, call),
	}
}
