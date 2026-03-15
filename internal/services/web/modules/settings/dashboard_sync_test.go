package settings

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestMountProfilePostTriggersDashboardSyncOnSuccess(t *testing.T) {
	t.Parallel()

	sync := &settingsDashboardSyncStub{}
	m := newSettingsModuleFromGateways(newPopulatedFakeGateway(), nil, settingsTestBase(), withDashboardSync(sync))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	form := url.Values{"name": {"Rhea Vale"}}
	req := httptest.NewRequest(http.MethodPost, routepath.AppSettingsProfile, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if len(sync.userIDs) != 1 || sync.userIDs[0] != "user-1" {
		t.Fatalf("ProfileSaved userIDs = %v, want [user-1]", sync.userIDs)
	}
}

func TestMountProfilePostDoesNotTriggerDashboardSyncOnValidationError(t *testing.T) {
	t.Parallel()

	sync := &settingsDashboardSyncStub{}
	m := newSettingsModuleFromGateways(newPopulatedFakeGateway(), nil, settingsTestBase(), withDashboardSync(sync))
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	form := url.Values{"name": {strings.Repeat("x", settingsapp.UserProfileNameMaxLength+1)}}
	req := httptest.NewRequest(http.MethodPost, routepath.AppSettingsProfile, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if len(sync.userIDs) != 0 {
		t.Fatalf("ProfileSaved userIDs = %v, want none", sync.userIDs)
	}
}

type settingsDashboardSyncStub struct {
	userIDs []string
}

func (s *settingsDashboardSyncStub) ProfileSaved(_ context.Context, userID string) {
	s.userIDs = append(s.userIDs, userID)
}

func (*settingsDashboardSyncStub) CampaignCreated(context.Context, string, string) {}

func (*settingsDashboardSyncStub) SessionStarted(context.Context, string, string) {}

func (*settingsDashboardSyncStub) SessionEnded(context.Context, string, string) {}

func (*settingsDashboardSyncStub) InviteChanged(context.Context, []string, string) {}
