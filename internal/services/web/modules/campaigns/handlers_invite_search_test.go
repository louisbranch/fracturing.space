package campaigns

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestMountInviteSearchForwardsRawUsernameQuery(t *testing.T) {
	t.Parallel()

	lastSearch := &campaignapp.SearchInviteUsersInput{}
	gateway := &fakeGateway{
		authorizationDecision: campaignapp.AuthorizationDecision{Evaluated: true, Allowed: true},
		inviteSearchResults:   []campaignapp.InviteUserSearchResult{{UserID: "user-2", Username: "alice", Name: "Alice"}},
		lastInviteSearchInput: lastSearch,
	}
	m := New(configWithGateway(gateway, managerMutationBase(), nil))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignInviteSearch("c1"), strings.NewReader(`{"query":" @Al! ","limit":3}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := lastSearch.Query; got != "@Al!" {
		t.Fatalf("forwarded query = %q, want %q", got, "@Al!")
	}
	if got := lastSearch.Limit; got != 3 {
		t.Fatalf("limit = %d, want %d", got, 3)
	}
	if got := lastSearch.ViewerUserID; got != "user-123" {
		t.Fatalf("viewer user id = %q, want %q", got, "user-123")
	}
	body := rr.Body.String()
	if !strings.Contains(body, `"username":"alice"`) {
		t.Fatalf("body = %q, want username result", body)
	}
}
