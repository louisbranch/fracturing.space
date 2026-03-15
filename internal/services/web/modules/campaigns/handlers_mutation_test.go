package campaigns

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformerrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// flashNoticeFromResponse extracts the flash notice from a response's Set-Cookie header.
func flashNoticeFromResponse(t *testing.T, rr *httptest.ResponseRecorder) flash.Notice {
	t.Helper()
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == flash.CookieName {
			decoded, err := base64.RawURLEncoding.DecodeString(cookie.Value)
			if err != nil {
				t.Fatalf("flash cookie base64 decode: %v", err)
			}
			var notice flash.Notice
			if err := json.Unmarshal(decoded, &notice); err != nil {
				t.Fatalf("flash cookie json unmarshal: %v", err)
			}
			return notice
		}
	}
	t.Fatal("no flash cookie found in response")
	return flash.Notice{}
}

func TestMountCharacterCreateUsesHXRedirect(t *testing.T) {
	t.Parallel()
	m := New(configWithGateway(managerMutationGateway(), managerMutationBase(), nil))
	mount, _ := m.Mount()
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreate("c1"), strings.NewReader("name=Hero&kind=pc"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if got := rr.Header().Get("HX-Redirect"); got != routepath.AppCampaignCharacter("c1", "char-created") {
		t.Fatalf("HX-Redirect = %q, want %q", got, routepath.AppCampaignCharacter("c1", "char-created"))
	}
	notice := flashNoticeFromResponse(t, rr)
	if notice.Kind != flash.KindSuccess || notice.Key != "web.campaigns.notice_character_created" {
		t.Fatalf("flash = %+v, want success/web.campaigns.notice_character_created", notice)
	}
}

func TestMountCharacterCreateRedirectsForNonHTMX(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(managerMutationGateway(), managerMutationBase(), nil))
	mount, _ := m.Mount()
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreate("c1"), strings.NewReader("name=Hero&kind=pc"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignCharacter("c1", "char-created") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignCharacter("c1", "char-created"))
	}
	notice := flashNoticeFromResponse(t, rr)
	if notice.Kind != flash.KindSuccess || notice.Key != "web.campaigns.notice_character_created" {
		t.Fatalf("flash = %+v, want success/web.campaigns.notice_character_created", notice)
	}
}

func TestMountCharacterCreateRedirectsToCreationFlowWhenWorkflowExists(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(managerMutationGateway(), managerMutationBase(), defaultTestWorkflows()))
	mount, _ := m.Mount()
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreate("c1"), strings.NewReader("name=Hero&pronouns=they%2Fthem&kind=pc"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignCharacterCreation("c1", "char-created") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignCharacterCreation("c1", "char-created"))
	}
	notice := flashNoticeFromResponse(t, rr)
	if notice.Kind != flash.KindSuccess || notice.Key != "web.campaigns.notice_character_created" {
		t.Fatalf("flash = %+v, want success/web.campaigns.notice_character_created", notice)
	}
}

func TestMountCharacterCreateRejectsInvalidKind(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(managerMutationGateway(), managerMutationBase(), nil))
	mount, _ := m.Mount()
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreate("c1"), strings.NewReader("name=Hero&kind=invalid"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	notice := flashNoticeFromResponse(t, rr)
	if notice.Kind != flash.KindError {
		t.Fatalf("flash kind = %q, want %q", notice.Kind, flash.KindError)
	}
	if notice.Key != "error.web.message.character_kind_value_is_invalid" {
		t.Fatalf("flash key = %q, want %q", notice.Key, "error.web.message.character_kind_value_is_invalid")
	}
}

func TestStableMutationRoutesReturnParseErrorFlashKeys(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(managerMutationGateway(), managerMutationBase(), nil))
	mount, _ := m.Mount()

	tests := []struct {
		name    string
		path    string
		wantKey string
	}{
		{
			name:    "campaign update parse error",
			path:    routepath.AppCampaignEdit("c1"),
			wantKey: "error.web.message.failed_to_parse_campaign_update_form",
		},
		{
			name:    "session start parse error",
			path:    routepath.AppCampaignSessionStart("c1"),
			wantKey: "error.web.message.failed_to_parse_session_start_form",
		},
		{
			name:    "session end parse error",
			path:    routepath.AppCampaignSessionEnd("c1"),
			wantKey: "error.web.message.failed_to_parse_session_end_form",
		},
		{
			name:    "invite create parse error",
			path:    routepath.AppCampaignInviteCreate("c1"),
			wantKey: "error.web.message.failed_to_parse_invite_create_form",
		},
		{
			name:    "invite revoke parse error",
			path:    routepath.AppCampaignInviteRevoke("c1"),
			wantKey: "error.web.message.failed_to_parse_invite_revoke_form",
		},
		{
			name:    "participant create parse error",
			path:    routepath.AppCampaignParticipantCreate("c1"),
			wantKey: "error.web.message.failed_to_parse_participant_create_form",
		},
		{
			name:    "participant update parse error",
			path:    routepath.AppCampaignParticipantEdit("c1", "p-manager"),
			wantKey: "error.web.message.failed_to_parse_participant_update_form",
		},
		{
			name:    "character update parse error",
			path:    routepath.AppCampaignCharacterEdit("c1", "char-1"),
			wantKey: "error.web.message.failed_to_parse_character_update_form",
		},
		{
			name:    "character controller parse error",
			path:    routepath.AppCampaignCharacterControl("c1", "char-1"),
			wantKey: "error.web.message.failed_to_parse_character_controller_form",
		},
		{
			name:    "character claim parse error",
			path:    routepath.AppCampaignCharacterControlClaim("c1", "char-1"),
			wantKey: "error.web.message.failed_to_parse_character_controller_form",
		},
		{
			name:    "character release parse error",
			path:    routepath.AppCampaignCharacterControlRelease("c1", "char-1"),
			wantKey: "error.web.message.failed_to_parse_character_controller_form",
		},
		{
			name:    "character delete parse error",
			path:    routepath.AppCampaignCharacterDelete("c1", "char-1"),
			wantKey: "error.web.message.failed_to_parse_character_delete_form",
		},
		{
			name:    "campaign ai binding parse error",
			path:    routepath.AppCampaignAIBinding("c1"),
			wantKey: "error.web.message.failed_to_parse_campaign_ai_binding_form",
		},
		{
			name:    "character create parse error",
			path:    routepath.AppCampaignCharacterCreate("c1"),
			wantKey: "error.web.message.failed_to_parse_character_create_form",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader("bad=%zz"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rr := httptest.NewRecorder()
			mount.Handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusFound {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
			}
			notice := flashNoticeFromResponse(t, rr)
			if notice.Key != tc.wantKey {
				t.Fatalf("flash key = %q, want %q", notice.Key, tc.wantKey)
			}
		})
	}
}

func TestStableMutationRoutesReturnRequiredFieldFlashKeys(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(managerMutationGateway(), managerMutationBase(), nil))
	mount, _ := m.Mount()

	tests := []struct {
		name    string
		path    string
		body    string
		wantKey string
	}{
		{
			name:    "session end missing session id",
			path:    routepath.AppCampaignSessionEnd("c1"),
			body:    "session_id=   ",
			wantKey: "error.web.message.session_id_is_required",
		},
		{
			name:    "invite create missing participant id",
			path:    routepath.AppCampaignInviteCreate("c1"),
			body:    "participant_id=   &username=alice",
			wantKey: "error.web.message.participant_id_is_required",
		},
		{
			name:    "invite revoke missing invite id",
			path:    routepath.AppCampaignInviteRevoke("c1"),
			body:    "invite_id=   ",
			wantKey: "error.web.message.invite_id_is_required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rr := httptest.NewRecorder()
			mount.Handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusFound {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
			}
			notice := flashNoticeFromResponse(t, rr)
			if notice.Key != tc.wantKey {
				t.Fatalf("flash key = %q, want %q", notice.Key, tc.wantKey)
			}
		})
	}
}

func TestInviteCreateRichErrorRendersSpecificToastAfterRedirect(t *testing.T) {
	t.Parallel()

	st := status.New(codes.AlreadyExists, "internal invite detail")
	richStatus, err := st.WithDetails(
		&errdetails.ErrorInfo{
			Reason: string(platformerrors.CodeInviteRecipientAlreadyInvited),
			Domain: platformerrors.Domain,
			Metadata: map[string]string{
				"CampaignID": "c1",
			},
		},
		&errdetails.LocalizedMessage{
			Locale:  "pt-BR",
			Message: "Mensagem em portugues",
		},
	)
	if err != nil {
		t.Fatalf("WithDetails() error = %v", err)
	}

	gateway := managerMutationGateway()
	gateway.createInviteErr = apperrors.MapGRPCTransportError(richStatus.Err(), apperrors.GRPCStatusMapping{
		FallbackKind:    apperrors.KindUnknown,
		FallbackKey:     "error.web.message.failed_to_create_invite",
		FallbackMessage: "failed to create invite",
	})
	m := New(configWithGateway(gateway, managerMutationBase(), nil))
	mount, _ := m.Mount()

	postReq := httptest.NewRequest(http.MethodPost, routepath.AppCampaignInviteCreate("c1"), strings.NewReader("participant_id=p-1&username=alice"))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postRR := httptest.NewRecorder()
	mount.Handler.ServeHTTP(postRR, postReq)
	if postRR.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", postRR.Code, http.StatusFound)
	}
	notice := flashNoticeFromResponse(t, postRR)
	if notice.Key != "error.web.message.failed_to_create_invite" {
		t.Fatalf("flash key = %q, want fallback key", notice.Key)
	}

	getReq := httptest.NewRequest(http.MethodGet, routepath.AppCampaignInvites("c1"), nil)
	for _, cookie := range postRR.Result().Cookies() {
		getReq.AddCookie(cookie)
	}
	getRR := httptest.NewRecorder()
	mount.Handler.ServeHTTP(getRR, getReq)
	body := getRR.Body.String()
	if !strings.Contains(body, "failed to create invite") {
		t.Fatalf("body missing localized invite error copy: %q", body)
	}
}

func TestStableMutationRoutesRedirectWithHTMXParity(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(managerMutationGateway(), managerMutationBase(), nil))
	mount, _ := m.Mount()

	tests := []struct {
		name         string
		path         string
		body         string
		wantLocation string
		wantFlashKey string
	}{
		{
			name:         "campaign update",
			path:         routepath.AppCampaignEdit("c1"),
			body:         "name=Campaign+One&theme_prompt=Updated+theme&locale=en-US",
			wantLocation: routepath.AppCampaign("c1"),
			wantFlashKey: "web.campaigns.notice_campaign_updated",
		},
		{
			name:         "session start",
			path:         routepath.AppCampaignSessionStart("c1"),
			body:         "name=Session+Two",
			wantLocation: routepath.AppCampaignGame("c1"),
			wantFlashKey: "web.campaigns.notice_session_started",
		},
		{
			name:         "session end",
			path:         routepath.AppCampaignSessionEnd("c1"),
			body:         "session_id=sess-1",
			wantLocation: routepath.AppCampaignSessions("c1"),
			wantFlashKey: "web.campaigns.notice_session_ended",
		},
		{
			name:         "invite create",
			path:         routepath.AppCampaignInviteCreate("c1"),
			body:         "participant_id=p-1&username=alice",
			wantLocation: routepath.AppCampaignInvites("c1"),
			wantFlashKey: "web.campaigns.notice_invite_created",
		},
		{
			name:         "invite revoke",
			path:         routepath.AppCampaignInviteRevoke("c1"),
			body:         "invite_id=inv-1",
			wantLocation: routepath.AppCampaignInvites("c1"),
			wantFlashKey: "web.campaigns.notice_invite_revoked",
		},
		{
			name:         "participant create",
			path:         routepath.AppCampaignParticipantCreate("c1"),
			body:         "name=Pending+Seat&role=player&campaign_access=member",
			wantLocation: routepath.AppCampaignInvites("c1"),
			wantFlashKey: "web.campaigns.notice_participant_created",
		},
		{
			name:         "participant update",
			path:         routepath.AppCampaignParticipantEdit("c1", "p-manager"),
			body:         "name=Manager+One&role=player&pronouns=they%2Fthem",
			wantLocation: routepath.AppCampaignParticipants("c1"),
			wantFlashKey: "web.campaigns.notice_participant_updated",
		},
		{
			name:         "character update",
			path:         routepath.AppCampaignCharacterEdit("c1", "char-1"),
			body:         "name=Hero+Updated&pronouns=they%2Fthem",
			wantLocation: routepath.AppCampaignCharacter("c1", "char-1"),
			wantFlashKey: "web.campaigns.notice_character_updated",
		},
		{
			name:         "character controller update",
			path:         routepath.AppCampaignCharacterControl("c1", "char-1"),
			body:         "participant_id=p-manager",
			wantLocation: routepath.AppCampaignCharacter("c1", "char-1"),
			wantFlashKey: "web.campaigns.notice_character_controller_updated",
		},
		{
			name:         "character claim",
			path:         routepath.AppCampaignCharacterControlClaim("c1", "char-1"),
			body:         "",
			wantLocation: routepath.AppCampaignCharacter("c1", "char-1"),
			wantFlashKey: "web.campaigns.notice_character_control_claimed",
		},
		{
			name:         "character release",
			path:         routepath.AppCampaignCharacterControlRelease("c1", "char-1"),
			body:         "",
			wantLocation: routepath.AppCampaignCharacter("c1", "char-1"),
			wantFlashKey: "web.campaigns.notice_character_control_released",
		},
		{
			name:         "character delete",
			path:         routepath.AppCampaignCharacterDelete("c1", "char-1"),
			body:         "",
			wantLocation: routepath.AppCampaignCharacters("c1"),
			wantFlashKey: "web.campaigns.notice_character_deleted",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name+" browser", func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rr := httptest.NewRecorder()
			mount.Handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusFound {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
			}
			if got := rr.Header().Get("Location"); got != tc.wantLocation {
				t.Fatalf("Location = %q, want %q", got, tc.wantLocation)
			}
			notice := flashNoticeFromResponse(t, rr)
			if notice.Kind != flash.KindSuccess {
				t.Fatalf("flash kind = %q, want %q", notice.Kind, flash.KindSuccess)
			}
			if notice.Key != tc.wantFlashKey {
				t.Fatalf("flash key = %q, want %q", notice.Key, tc.wantFlashKey)
			}
		})

		t.Run(tc.name+" htmx", func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			rr := httptest.NewRecorder()
			mount.Handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
			}
			if got := rr.Header().Get("HX-Redirect"); got != tc.wantLocation {
				t.Fatalf("HX-Redirect = %q, want %q", got, tc.wantLocation)
			}
			notice := flashNoticeFromResponse(t, rr)
			if notice.Kind != flash.KindSuccess {
				t.Fatalf("flash kind = %q, want %q", notice.Kind, flash.KindSuccess)
			}
			if notice.Key != tc.wantFlashKey {
				t.Fatalf("flash key = %q, want %q", notice.Key, tc.wantFlashKey)
			}
		})
	}
}

func TestCampaignAIBindingRouteRedirectsToCampaignOverview(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		authorizationDecision: campaignapp.AuthorizationDecision{
			Evaluated:           true,
			Allowed:             true,
			ActorCampaignAccess: "Owner",
		},
	}, managerMutationBase(), nil))
	mount, _ := m.Mount()

	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignAIBinding("c1"), strings.NewReader("ai_agent_id=agent-1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaign("c1") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaign("c1"))
	}
	notice := flashNoticeFromResponse(t, rr)
	if notice.Kind != flash.KindSuccess || notice.Key != "web.campaigns.notice_ai_binding_saved" {
		t.Fatalf("flash = %+v, want success/web.campaigns.notice_ai_binding_saved", notice)
	}
}

func TestParticipantUpdateRouteValidatesRoleAndAccess(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(managerMutationGateway(), managerMutationBase(), nil))
	mount, _ := m.Mount()

	tests := []struct {
		name    string
		body    string
		wantKey string
	}{
		{
			name:    "invalid role",
			body:    "name=Manager+One&role=invalid&pronouns=they%2Fthem",
			wantKey: "error.web.message.participant_role_value_is_invalid",
		},
		{
			name:    "invalid access",
			body:    "name=Manager+One&role=player&campaign_access=invalid",
			wantKey: "error.web.message.campaign_access_value_is_invalid",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignParticipantEdit("c1", "p-manager"), strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rr := httptest.NewRecorder()
			mount.Handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusFound {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
			}
			notice := flashNoticeFromResponse(t, rr)
			if notice.Key != tc.wantKey {
				t.Fatalf("flash key = %q, want %q", notice.Key, tc.wantKey)
			}
		})
	}
}

func TestParticipantCreateRouteValidatesFields(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(managerMutationGateway(), managerMutationBase(), nil))
	mount, _ := m.Mount()

	tests := []struct {
		name    string
		body    string
		wantKey string
	}{
		{
			name:    "missing name",
			body:    "name=   &role=player&campaign_access=member",
			wantKey: "error.web.message.participant_name_is_required",
		},
		{
			name:    "invalid role",
			body:    "name=Pending+Seat&role=invalid&campaign_access=member",
			wantKey: "error.web.message.participant_role_value_is_invalid",
		},
		{
			name:    "invalid access",
			body:    "name=Pending+Seat&role=player&campaign_access=invalid",
			wantKey: "error.web.message.campaign_access_value_is_invalid",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignParticipantCreate("c1"), strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rr := httptest.NewRecorder()
			mount.Handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusFound {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
			}
			notice := flashNoticeFromResponse(t, rr)
			if notice.Key != tc.wantKey {
				t.Fatalf("flash key = %q, want %q", notice.Key, tc.wantKey)
			}
			if got := rr.Header().Get("Location"); got != routepath.AppCampaignParticipantCreate("c1") {
				t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignParticipantCreate("c1"))
			}
		})
	}
}

func TestParticipantCreateRouteRejectsHumanGMForAIGMCampaigns(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items:           []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		workspaceGMMode: "AI",
		participants: []campaignapp.CampaignParticipant{{
			ID:             "p-manager",
			UserID:         "user-123",
			CampaignAccess: "Manager",
		}},
	}, managerMutationBase(), nil))
	mount, _ := m.Mount()

	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignParticipantCreate("c1"), strings.NewReader("name=Pending+GM&role=gm&campaign_access=member"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	notice := flashNoticeFromResponse(t, rr)
	if notice.Key != "error.web.message.ai_gm_campaign_disallows_human_gm_participants" {
		t.Fatalf("flash key = %q, want %q", notice.Key, "error.web.message.ai_gm_campaign_disallows_human_gm_participants")
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignParticipantCreate("c1") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignParticipantCreate("c1"))
	}
}

func TestParticipantUpdateRouteRejectsAIInvariantTampering(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		participant: campaignapp.CampaignParticipant{
			ID:             "p-ai",
			Name:           "Caretaker",
			Role:           "GM",
			CampaignAccess: "Member",
			Controller:     "AI",
			Pronouns:       "it/its",
		},
		authorizationDecision: campaignapp.AuthorizationDecision{Evaluated: true, Allowed: true},
	}, managerMutationBase(), nil))
	mount, _ := m.Mount()

	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignParticipantEdit("c1", "p-ai"), strings.NewReader("name=Caretaker&role=player&campaign_access=member&pronouns=it%2Fits"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	notice := flashNoticeFromResponse(t, rr)
	if notice.Key != "error.web.message.participant_ai_role_and_access_are_fixed" {
		t.Fatalf("flash key = %q, want %q", notice.Key, "error.web.message.participant_ai_role_and_access_are_fixed")
	}
}

func TestCampaignUpdateRouteValidatesLocale(t *testing.T) {
	t.Parallel()

	m := New(configWithGateway(managerMutationGateway(), managerMutationBase(), nil))
	mount, _ := m.Mount()

	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignEdit("c1"), strings.NewReader("name=Campaign+One&theme_prompt=Theme&locale=es-ES"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	notice := flashNoticeFromResponse(t, rr)
	if notice.Key != "error.web.message.campaign_locale_value_is_invalid" {
		t.Fatalf("flash key = %q, want %q", notice.Key, "error.web.message.campaign_locale_value_is_invalid")
	}
}

func TestRequestContextWithUserIDBehavior(t *testing.T) {
	t.Parallel()

	h := newHandlersFromConfig(serviceConfigWithGateway(fakeGateway{}), modulehandler.NewBase(nil, nil, nil), "", nil)
	req := httptest.NewRequest(http.MethodGet, routepath.CampaignsPrefix, nil)
	ctx, _ := h.RequestContextAndUserID(req)
	if md, ok := metadata.FromOutgoingContext(ctx); ok && len(md.Get(grpcmeta.UserIDHeader)) > 0 {
		t.Fatalf("unexpected user metadata when resolver is nil")
	}

	h = newHandlersFromConfig(serviceConfigWithGateway(fakeGateway{}), modulehandler.NewBase(func(*http.Request) string { return "user-123" }, nil, nil), "", nil)
	ctx, _ = h.RequestContextAndUserID(req)
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatalf("expected outgoing metadata")
	}
	if got := md.Get(grpcmeta.UserIDHeader); len(got) != 1 || got[0] != "user-123" {
		t.Fatalf("user metadata = %v, want [user-123]", got)
	}
}

func TestParseAppCharacterKind(t *testing.T) {
	t.Parallel()

	if kind, ok := parseAppCharacterKind("pc"); !ok || kind != campaignapp.CharacterKindPC {
		t.Fatalf("parseAppCharacterKind pc = (%v, %v)", kind, ok)
	}
	if kind, ok := parseAppCharacterKind("npc"); !ok || kind != campaignapp.CharacterKindNPC {
		t.Fatalf("parseAppCharacterKind npc = (%v, %v)", kind, ok)
	}
	if _, ok := parseAppCharacterKind("invalid"); ok {
		t.Fatalf("expected invalid character kind to fail parse")
	}
}

func managerMutationGateway() fakeGateway {
	return fakeGateway{
		items: []campaignapp.CampaignSummary{{ID: "c1", Name: "First"}},
		participants: []campaignapp.CampaignParticipant{{
			ID:             "p-manager",
			UserID:         "user-123",
			CampaignAccess: "Manager",
		}},
	}
}

func managerMutationBase() modulehandler.Base {
	return modulehandler.NewBase(func(*http.Request) string { return "user-123" }, nil, nil)
}
