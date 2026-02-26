package web

import (
	"context"
	"errors"
	"net/http"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	campaignfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

func buildCampaignFeatureSessionContextDependencies(h *handler, d *campaignfeature.AppCampaignDependencies) {
	d.CampaignSessionPresent = func(w http.ResponseWriter, r *http.Request) bool {
		if h == nil || h.sessions == nil {
			return false
		}
		sess := sessionFromRequest(r, h.sessions)
		if sess == nil {
			http.Redirect(w, r, routepath.AuthLogin, http.StatusFound)
			return false
		}
		return true
	}

	d.CampaignListUserID = func(r *http.Request) (string, bool) {
		if h == nil || h.sessions == nil || r == nil {
			return "", false
		}
		sess := sessionFromRequest(r, h.sessions)
		if sess == nil {
			return "", false
		}
		userID, err := h.sessionUserIDForSession(r.Context(), sess)
		if err != nil {
			return "", false
		}
		return strings.TrimSpace(userID), strings.TrimSpace(userID) != ""
	}

	d.CampaignReadContext = func(w http.ResponseWriter, r *http.Request, unavailableTitle string) (context.Context, string, bool) {
		return campaignfeature.ReadCampaignContext(
			w,
			r,
			unavailableTitle,
			func(response http.ResponseWriter, request *http.Request) bool {
				if request == nil {
					return false
				}
				if sessionFromRequest(request, h.sessions) == nil {
					http.Redirect(response, request, routepath.AuthLogin, http.StatusFound)
					return false
				}
				return sessionFromRequest(request, h.sessions) != nil
			},
			func(ctx context.Context, request *http.Request) (string, error) {
				if request == nil {
					return "", nil
				}
				sess := sessionFromRequest(request, h.sessions)
				return h.sessionUserIDForSession(ctx, sess)
			},
			h.renderErrorPage,
		)
	}

	d.RequireCampaignActor = func(w http.ResponseWriter, r *http.Request, campaignID string) (*statev1.Participant, bool) {
		return campaignfeature.RequireCampaignActor(
			w,
			r,
			campaignID,
			func(response http.ResponseWriter, request *http.Request) bool {
				if request == nil {
					return false
				}
				if sessionFromRequest(request, h.sessions) == nil {
					http.Redirect(response, request, routepath.AuthLogin, http.StatusFound)
					return false
				}
				return true
			},
			func(ctx context.Context, requestedCampaignID string) (*statev1.Participant, error) {
				if h == nil || h.participantClient == nil {
					return nil, errors.New("participant client is not configured")
				}
				if r == nil {
					return nil, nil
				}
				sess := sessionFromRequest(r, h.sessions)
				userID, err := h.sessionUserIDForSession(ctx, sess)
				if err != nil {
					return nil, err
				}
				userID = strings.TrimSpace(userID)
				if userID == "" {
					return nil, nil
				}
				return campaignfeature.ResolveCampaignParticipantByUserID(
					grpcauthctx.WithUserID(ctx, userID),
					requestedCampaignID,
					userID,
					func(ctx context.Context, req *statev1.ListParticipantsRequest) (*statev1.ListParticipantsResponse, error) {
						return h.participantClient.ListParticipants(ctx, req)
					},
				)
			},
			h.renderErrorPage,
		)
	}

	d.CampaignParticipantByUserID = func(ctx context.Context, campaignID string, userID string) (*statev1.Participant, error) {
		var listParticipants func(context.Context, *statev1.ListParticipantsRequest) (*statev1.ListParticipantsResponse, error)
		if h != nil && h.participantClient != nil {
			listParticipants = func(ctx context.Context, req *statev1.ListParticipantsRequest) (*statev1.ListParticipantsResponse, error) {
				return h.participantClient.ListParticipants(ctx, req)
			}
		}
		return campaignfeature.ResolveCampaignParticipantByUserID(ctx, campaignID, userID, listParticipants)
	}

	d.SessionDisplayName = func(r *http.Request) string {
		sess := sessionFromRequest(r, h.sessions)
		if sess == nil {
			return ""
		}
		return strings.TrimSpace(sess.displayName)
	}

	d.PageContext = func(w http.ResponseWriter, r *http.Request) webtemplates.PageContext {
		return h.pageContext(w, r)
	}
	d.PageContextForCampaign = func(w http.ResponseWriter, r *http.Request, campaignID string) webtemplates.PageContext {
		return h.pageContextForCampaign(w, r, campaignID)
	}
}
