package sessions

import (
	"net/url"
	"strings"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
)

// parseStartSessionInput normalizes the session-create form into app input.
func parseStartSessionInput(form url.Values) campaignapp.StartSessionInput {
	return campaignapp.StartSessionInput{Name: strings.TrimSpace(form.Get("name"))}
}

// parseEndSessionInput normalizes the session-end form into app input.
func parseEndSessionInput(form url.Values) campaignapp.EndSessionInput {
	return campaignapp.EndSessionInput{SessionID: strings.TrimSpace(form.Get("session_id"))}
}
