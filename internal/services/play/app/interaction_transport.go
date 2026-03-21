package app

import (
	"context"
	"net/http"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	gogrpc "google.golang.org/grpc"
)

type interactionStateResponse interface {
	GetState() *gamev1.InteractionState
}

// rpcInteractionMutationHandler adapts standard interaction RPCs into the shared
// browser mutation flow so route wiring does not repeat campaign assignment and
// state extraction boilerplate.
func rpcInteractionMutationHandler[TReq any, TResp interactionStateResponse](
	server *Server,
	newRequest func() *TReq,
	setCampaignID func(*TReq, string),
	call func(context.Context, *TReq, ...gogrpc.CallOption) (TResp, error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := newRequest()
		server.handleInteractionMutation(w, r, req, func(ctx context.Context, campaignID string) (*gamev1.InteractionState, error) {
			setCampaignID(req, campaignID)
			resp, err := call(ctx, req)
			if err != nil {
				return nil, err
			}
			return resp.GetState(), nil
		})
	}
}

// rpcInteractionMutationHandlerWithoutBody adapts interaction RPCs whose only
// input is the authenticated campaign scope.
func rpcInteractionMutationHandlerWithoutBody[TReq any, TResp interactionStateResponse](
	server *Server,
	buildRequest func(campaignID string) *TReq,
	call func(context.Context, *TReq, ...gogrpc.CallOption) (TResp, error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		server.handleInteractionMutation(w, r, nil, func(ctx context.Context, campaignID string) (*gamev1.InteractionState, error) {
			resp, err := call(ctx, buildRequest(campaignID))
			if err != nil {
				return nil, err
			}
			return resp.GetState(), nil
		})
	}
}

// handleInteractionMutation owns the common request mapping and response refresh
// behavior for the browser-facing interaction mutation surface.
func (s *Server) handleInteractionMutation(
	w http.ResponseWriter,
	r *http.Request,
	target any,
	call func(context.Context, string) (*gamev1.InteractionState, error),
) {
	req, ok := s.requirePlayRequest(w, r)
	if !ok {
		return
	}
	if target != nil {
		if err := decodeStrictJSON(r, target); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid json body")
			return
		}
	}
	state, err := call(req.authContext(r.Context()), req.CampaignID)
	if err != nil {
		writeRPCError(w, err)
		return
	}
	response, err := s.application().interactionResponse(r.Context(), state)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to refresh play interaction state")
		return
	}
	writeJSON(w, http.StatusOK, response)
	s.realtime.broadcastCurrent(req.CampaignID)
}
