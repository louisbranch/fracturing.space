package campaigns

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestCreateCampaignTriggersDashboardSyncOnSuccess(t *testing.T) {
	t.Parallel()

	sync := &campaignDashboardSyncStub{}
	m := New(configWithGatewayAndSync(fakeGateway{createdCampaignID: "camp-777"}, managerMutationBase(), nil, sync))
	mount, _ := m.Mount()

	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignsCreate, strings.NewReader("name=New+Campaign&system=daggerheart&gm_mode=human"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if len(sync.created) != 1 || sync.created[0] != "user-123/camp-777" {
		t.Fatalf("created sync calls = %v, want [user-123/camp-777]", sync.created)
	}
}

func TestSessionMutationsTriggerDashboardSyncOnSuccess(t *testing.T) {
	t.Parallel()

	sync := &campaignDashboardSyncStub{}
	m := New(configWithGatewayAndSync(managerMutationGateway(), managerMutationBase(), nil, sync))
	mount, _ := m.Mount()

	startReq := httptest.NewRequest(http.MethodPost, routepath.AppCampaignSessionStart("c1"), strings.NewReader("name=Session+Two"))
	startReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	startRR := httptest.NewRecorder()
	mount.Handler.ServeHTTP(startRR, startReq)

	endReq := httptest.NewRequest(http.MethodPost, routepath.AppCampaignSessionEnd("c1"), strings.NewReader("session_id=sess-1"))
	endReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	endRR := httptest.NewRecorder()
	mount.Handler.ServeHTTP(endRR, endReq)

	if len(sync.started) != 1 || sync.started[0] != "user-123/c1" {
		t.Fatalf("started sync calls = %v, want [user-123/c1]", sync.started)
	}
	if len(sync.ended) != 1 || sync.ended[0] != "user-123/c1" {
		t.Fatalf("ended sync calls = %v, want [user-123/c1]", sync.ended)
	}
}

func TestSessionMutationDoesNotTriggerDashboardSyncOnValidationError(t *testing.T) {
	t.Parallel()

	sync := &campaignDashboardSyncStub{}
	m := New(configWithGatewayAndSync(managerMutationGateway(), managerMutationBase(), nil, sync))
	mount, _ := m.Mount()

	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignSessionEnd("c1"), strings.NewReader("session_id=   "))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)

	if len(sync.started) != 0 || len(sync.ended) != 0 || len(sync.created) != 0 {
		t.Fatalf("unexpected sync calls: created=%v started=%v ended=%v", sync.created, sync.started, sync.ended)
	}
}

type campaignDashboardSyncStub struct {
	created []string
	started []string
	ended   []string
	invites []string
}

func (s *campaignDashboardSyncStub) CampaignCreated(_ context.Context, userID, campaignID string) {
	s.created = append(s.created, userID+"/"+campaignID)
}

func (s *campaignDashboardSyncStub) SessionStarted(_ context.Context, userID, campaignID string) {
	s.started = append(s.started, userID+"/"+campaignID)
}

func (s *campaignDashboardSyncStub) SessionEnded(_ context.Context, userID, campaignID string) {
	s.ended = append(s.ended, userID+"/"+campaignID)
}

func (s *campaignDashboardSyncStub) InviteChanged(_ context.Context, userIDs []string, campaignID string) {
	s.invites = append(s.invites, strings.Join(userIDs, ",")+"/"+campaignID)
}
