package invites

import (
	"net/http"
	"strings"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
)

const inviteSearchMaxJSONBodyBytes = 16 << 10

// inviteSearchInput stores the JSON payload accepted by invite search.
type inviteSearchInput struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

// inviteSearchUserResponse stores one JSON user suggestion row.
type inviteSearchUserResponse struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	IsContact bool   `json:"is_contact"`
}

// inviteSearchResponse wraps invite search results for transport responses.
type inviteSearchResponse struct {
	Users []inviteSearchUserResponse `json:"users"`
}

// HandleInviteSearch returns invite-recipient suggestions for one campaign.
func (h Handler) HandleInviteSearch(w http.ResponseWriter, r *http.Request, campaignID string) {
	input, err := parseInviteSearchInput(r)
	if err != nil {
		h.writeJSONError(w, r, err)
		return
	}
	ctx, userID := h.RequestContextAndUserID(r)
	results, err := h.invites.reads.SearchInviteUsers(ctx, campaignID, campaignapp.SearchInviteUsersInput{
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

// parseInviteSearchInput validates and trims the invite search JSON payload.
func parseInviteSearchInput(r *http.Request) (inviteSearchInput, error) {
	var payload inviteSearchInput
	if err := httpx.DecodeJSONStrictInvalidInput(r, &payload, inviteSearchMaxJSONBodyBytes); err != nil {
		return inviteSearchInput{}, err
	}
	return inviteSearchInput{
		Query: strings.TrimSpace(payload.Query),
		Limit: payload.Limit,
	}, nil
}

// newInviteSearchResponse maps app-layer search results into JSON response rows.
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

// writeJSONError localizes invite search errors for JSON clients.
func (h Handler) writeJSONError(w http.ResponseWriter, r *http.Request, err error) {
	loc, lang := h.PageLocalizer(w, r)
	_ = httpx.WriteJSONError(w, apperrors.HTTPStatus(err), webi18n.LocalizeError(loc, err, lang))
}
