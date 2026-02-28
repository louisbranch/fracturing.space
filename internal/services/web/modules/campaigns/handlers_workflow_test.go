package campaigns

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestHandleCharacterCreationStepRouteAppliesStepAndRedirects(t *testing.T) {
	t.Parallel()

	gateway := &workflowCaptureGateway{
		fakeGateway: fakeGateway{
			items: []CampaignSummary{{ID: "c1", Name: "Campaign"}},
			characterCreationProgress: CampaignCharacterCreationProgress{
				Steps:    []CampaignCharacterCreationStep{{Step: 1, Key: "class_subclass", Complete: false}},
				NextStep: 1,
			},
		},
	}
	m := NewStableWithGateway(gateway, modulehandler.NewBase(func(*http.Request) string { return "user-123" }, nil, nil), "", defaultTestWorkflows())
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreationStep("c1", "char-1"), strings.NewReader(url.Values{
		"class_id":    {"warrior"},
		"subclass_id": {"guardian"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignCharacter("c1", "char-1") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignCharacter("c1", "char-1"))
	}
	if !gateway.applyCalled {
		t.Fatalf("expected apply step call")
	}
	if got := gateway.applyCampaignID; got != "c1" {
		t.Fatalf("apply campaign id = %q, want %q", got, "c1")
	}
	if got := gateway.applyCharacterID; got != "char-1" {
		t.Fatalf("apply character id = %q, want %q", got, "char-1")
	}
	if got := gateway.applyInput.ClassSubclass; got == nil {
		t.Fatalf("apply input type mismatch: %T", gateway.applyInput)
	}
	if got := gateway.applyInput.ClassSubclass.ClassID; got != "warrior" {
		t.Fatalf("class id = %q, want %q", got, "warrior")
	}
	if got := gateway.applyInput.ClassSubclass.SubclassID; got != "guardian" {
		t.Fatalf("subclass id = %q, want %q", got, "guardian")
	}
}

func TestHandleCharacterCreationStepRouteUsesHXRedirect(t *testing.T) {
	t.Parallel()

	gateway := &workflowCaptureGateway{
		fakeGateway: fakeGateway{
			items: []CampaignSummary{{ID: "c1", Name: "Campaign"}},
			characterCreationProgress: CampaignCharacterCreationProgress{
				Steps:    []CampaignCharacterCreationStep{{Step: 1, Key: "class_subclass", Complete: false}},
				NextStep: 1,
			},
		},
	}
	m := NewStableWithGateway(gateway, modulehandler.NewTestBase(), "", defaultTestWorkflows())
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreationStep("c1", "char-1"), strings.NewReader(url.Values{
		"class_id":    {"warrior"},
		"subclass_id": {"guardian"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()

	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("HX-Redirect"); got != routepath.AppCampaignCharacter("c1", "char-1") {
		t.Fatalf("HX-Redirect = %q, want %q", got, routepath.AppCampaignCharacter("c1", "char-1"))
	}
	if !gateway.applyCalled {
		t.Fatalf("expected apply step call")
	}
}

func TestHandleCharacterCreationStepRouteRejectsInvalidCharacterCreationForm(t *testing.T) {
	t.Parallel()

	m := NewStableWithGateway(fakeGateway{
		items: []CampaignSummary{{ID: "c1", Name: "Campaign"}},
		characterCreationProgress: CampaignCharacterCreationProgress{
			Steps:    []CampaignCharacterCreationStep{{Step: 1, Key: "class_subclass", Complete: false}},
			NextStep: 1,
		},
	}, modulehandler.NewTestBase(), "", defaultTestWorkflows())
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreationStep("c1", "char-1"), strings.NewReader("class_id=warrior"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleCharacterCreationStepRouteReturnsBadRequestWhenWorkflowReady(t *testing.T) {
	t.Parallel()

	gateway := &workflowCaptureGateway{
		fakeGateway: fakeGateway{
			items: []CampaignSummary{{ID: "c1", Name: "Campaign"}},
			characterCreationProgress: CampaignCharacterCreationProgress{
				Ready: true,
			},
		},
	}
	m := NewStableWithGateway(gateway, modulehandler.NewTestBase(), "", defaultTestWorkflows())
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreationStep("c1", "char-1"), strings.NewReader("class_id=warrior&subclass_id=guardian"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if gateway.applyCalled {
		t.Fatalf("unexpected apply step call when workflow is ready")
	}
}

func TestHandleCharacterCreationResetRouteRedirectsAndCallsGateway(t *testing.T) {
	t.Parallel()

	gateway := &workflowCaptureGateway{
		fakeGateway: fakeGateway{
			items: []CampaignSummary{{ID: "c1", Name: "Campaign"}},
		},
	}
	m := NewStableWithGateway(gateway, modulehandler.NewTestBase(), "", defaultTestWorkflows())
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreationReset("c1", "char-1"), nil)
	rr := httptest.NewRecorder()

	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignCharacter("c1", "char-1") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignCharacter("c1", "char-1"))
	}
	if !gateway.resetCalled {
		t.Fatalf("expected reset workflow call")
	}
	if got := gateway.resetCampaignID; got != "c1" {
		t.Fatalf("reset campaign id = %q, want %q", got, "c1")
	}
	if got := gateway.resetCharacterID; got != "char-1" {
		t.Fatalf("reset character id = %q, want %q", got, "char-1")
	}
}

func TestHandleCharacterCreationResetRouteUsesHXRedirect(t *testing.T) {
	t.Parallel()

	gateway := &workflowCaptureGateway{
		fakeGateway: fakeGateway{
			items: []CampaignSummary{{ID: "c1", Name: "Campaign"}},
		},
	}
	m := NewStableWithGateway(gateway, modulehandler.NewTestBase(), "", defaultTestWorkflows())
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreationReset("c1", "char-1"), nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()

	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("HX-Redirect"); got != routepath.AppCampaignCharacter("c1", "char-1") {
		t.Fatalf("HX-Redirect = %q, want %q", got, routepath.AppCampaignCharacter("c1", "char-1"))
	}
}

type workflowCaptureGateway struct {
	fakeGateway
	applyCalled      bool
	applyCampaignID  string
	applyCharacterID string
	applyInput       *CampaignCharacterCreationStepInput
	resetCalled      bool
	resetCampaignID  string
	resetCharacterID string
}

func (g *workflowCaptureGateway) ApplyCharacterCreationStep(
	_ context.Context,
	campaignID string,
	characterID string,
	input *CampaignCharacterCreationStepInput,
) error {
	g.applyCalled = true
	g.applyCampaignID = campaignID
	g.applyCharacterID = characterID
	g.applyInput = input
	return g.fakeGateway.ApplyCharacterCreationStep(context.Background(), campaignID, characterID, input)
}

func (g *workflowCaptureGateway) ResetCharacterCreationWorkflow(
	_ context.Context,
	campaignID string,
	characterID string,
) error {
	g.resetCalled = true
	g.resetCampaignID = campaignID
	g.resetCharacterID = characterID
	return g.fakeGateway.ResetCharacterCreationWorkflow(context.Background(), campaignID, characterID)
}
