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
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/activate-scene", func() *gamev1.ActivateSceneRequest {
			return &gamev1.ActivateSceneRequest{}
		}, func(req *gamev1.ActivateSceneRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.ActivateScene),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/open-scene-player-phase", func() *gamev1.OpenScenePlayerPhaseRequest {
			return &gamev1.OpenScenePlayerPhaseRequest{}
		}, func(req *gamev1.OpenScenePlayerPhaseRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.OpenScenePlayerPhase),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/submit-scene-player-action", func() *gamev1.SubmitScenePlayerActionRequest {
			return &gamev1.SubmitScenePlayerActionRequest{}
		}, func(req *gamev1.SubmitScenePlayerActionRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.SubmitScenePlayerAction),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/yield-scene-player-phase", func() *gamev1.YieldScenePlayerPhaseRequest {
			return &gamev1.YieldScenePlayerPhaseRequest{}
		}, func(req *gamev1.YieldScenePlayerPhaseRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.YieldScenePlayerPhase),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/withdraw-scene-player-yield", func() *gamev1.WithdrawScenePlayerYieldRequest {
			return &gamev1.WithdrawScenePlayerYieldRequest{}
		}, func(req *gamev1.WithdrawScenePlayerYieldRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.WithdrawScenePlayerYield),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/interrupt-scene-player-phase", func() *gamev1.InterruptScenePlayerPhaseRequest {
			return &gamev1.InterruptScenePlayerPhaseRequest{}
		}, func(req *gamev1.InterruptScenePlayerPhaseRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.InterruptScenePlayerPhase),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/record-scene-gm-interaction", func() *gamev1.RecordSceneGMInteractionRequest {
			return &gamev1.RecordSceneGMInteractionRequest{}
		}, func(req *gamev1.RecordSceneGMInteractionRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.RecordSceneGMInteraction),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/resolve-scene-player-review", func() *gamev1.ResolveScenePlayerReviewRequest {
			return &gamev1.ResolveScenePlayerReviewRequest{}
		}, func(req *gamev1.ResolveScenePlayerReviewRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.ResolveScenePlayerReview),
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/open-session-ooc", func() *gamev1.OpenSessionOOCRequest {
			return &gamev1.OpenSessionOOCRequest{}
		}, func(req *gamev1.OpenSessionOOCRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.OpenSessionOOC),
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
		jsonInteractionRoute(server, "POST /api/campaigns/{campaignID}/interaction/resolve-session-ooc", func() *gamev1.ResolveSessionOOCRequest {
			return &gamev1.ResolveSessionOOCRequest{}
		}, func(req *gamev1.ResolveSessionOOCRequest, campaignID string) {
			req.CampaignId = campaignID
		}, server.interaction.ResolveSessionOOC),
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
