package campaigns

import (
	"net/http"
	"strings"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/jsoninput"
)

const inviteSearchMaxJSONBodyBytes = 16 << 10

// inviteSearchInput captures one invite-search JSON request payload.
type inviteSearchInput struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

// inviteSearchUserResponse defines one invite-search JSON result row.
type inviteSearchUserResponse struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	IsContact bool   `json:"is_contact"`
}

// inviteSearchResponse defines the invite-search JSON response contract.
type inviteSearchResponse struct {
	Users []inviteSearchUserResponse `json:"users"`
}

// handleInviteSearch returns invite-recipient suggestions for one campaign.
func (h handlers) handleInviteSearch(w http.ResponseWriter, r *http.Request, campaignID string) {
	input, err := parseInviteSearchInput(r)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	ctx, userID := h.RequestContextAndUserID(r)
	results, err := h.service.SearchInviteUsers(ctx, campaignID, campaignapp.SearchInviteUsersInput{
		ViewerUserID: userID,
		Query:        input.Query,
		Limit:        input.Limit,
	})
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	_ = httpx.WriteJSON(w, http.StatusOK, newInviteSearchResponse(results))
}

// parseInviteSearchInput decodes and normalizes one invite-search request body.
func parseInviteSearchInput(r *http.Request) (inviteSearchInput, error) {
	if r == nil || r.Body == nil {
		return inviteSearchInput{}, invalidInviteSearchJSONBodyError()
	}
	var payload inviteSearchInput
	if err := jsoninput.DecodeStrict(r, &payload, inviteSearchMaxJSONBodyBytes); err != nil {
		return inviteSearchInput{}, invalidInviteSearchJSONBodyError()
	}
	return inviteSearchInput{
		Query: strings.TrimSpace(payload.Query),
		Limit: payload.Limit,
	}, nil
}

// invalidInviteSearchJSONBodyError returns a stable invalid-input error for malformed JSON.
func invalidInviteSearchJSONBodyError() error {
	return apperrors.E(apperrors.KindInvalidInput, "Invalid JSON body.")
}

// newInviteSearchResponse maps app-layer results into the JSON response contract.
func newInviteSearchResponse(results []campaignapp.InviteUserSearchResult) inviteSearchResponse {
	users := make([]inviteSearchUserResponse, 0, len(results))
	for _, result := range results {
		users = append(users, inviteSearchUserResponse{
			UserID:    strings.TrimSpace(result.UserID),
			Username:  strings.TrimSpace(result.Username),
			Name:      strings.TrimSpace(result.Name),
			IsContact: result.IsContact,
		})
	}
	return inviteSearchResponse{Users: users}
}

// writeJSONError writes one localized JSON error response for campaigns JSON endpoints.
func (h handlers) writeJSONError(w http.ResponseWriter, r *http.Request, err error) {
	loc, _ := h.PageLocalizer(w, r)
	_ = httpx.WriteJSONError(w, apperrors.HTTPStatus(err), webi18n.LocalizeError(loc, err))
}
