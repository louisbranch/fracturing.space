package app

import (
	"context"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

type starterGatewayStub struct {
	lastPreviewKey string
	lastLaunchKey  string
	lastLaunchIn   LaunchStarterInput
	preview        CampaignStarterPreview
	previewErr     error
	launch         StarterLaunchResult
	launchErr      error
}

func (s *starterGatewayStub) StarterPreview(_ context.Context, starterKey string) (CampaignStarterPreview, error) {
	s.lastPreviewKey = starterKey
	if s.previewErr != nil {
		return CampaignStarterPreview{}, s.previewErr
	}
	return s.preview, nil
}

func (s *starterGatewayStub) LaunchStarter(_ context.Context, starterKey string, input LaunchStarterInput) (StarterLaunchResult, error) {
	s.lastLaunchKey = starterKey
	s.lastLaunchIn = input
	if s.launchErr != nil {
		return StarterLaunchResult{}, s.launchErr
	}
	return s.launch, nil
}

func TestStarterServicePreviewTrimsKeyAndDelegates(t *testing.T) {
	t.Parallel()

	gateway := &starterGatewayStub{preview: CampaignStarterPreview{EntryID: "starter:lantern"}}
	svc := NewStarterService(StarterServiceConfig{Gateway: gateway})

	preview, err := svc.StarterPreview(context.Background(), "  starter:lantern  ")
	if err != nil {
		t.Fatalf("StarterPreview() error = %v", err)
	}
	if gateway.lastPreviewKey != "starter:lantern" {
		t.Fatalf("preview key = %q, want %q", gateway.lastPreviewKey, "starter:lantern")
	}
	if preview.EntryID != "starter:lantern" {
		t.Fatalf("preview.EntryID = %q, want %q", preview.EntryID, "starter:lantern")
	}
}

func TestStarterServicePreviewRejectsEmptyKey(t *testing.T) {
	t.Parallel()

	svc := NewStarterService(StarterServiceConfig{Gateway: &starterGatewayStub{}})

	_, err := svc.StarterPreview(context.Background(), " \n\t ")
	if err == nil {
		t.Fatal("expected StarterPreview() error")
	}
	if got := apperrors.HTTPStatus(err); got != 400 {
		t.Fatalf("StarterPreview() HTTP status = %d, want %d", got, 400)
	}
}

func TestStarterServiceLaunchTrimsKeyAndDelegates(t *testing.T) {
	t.Parallel()

	gateway := &starterGatewayStub{launch: StarterLaunchResult{CampaignID: "camp-777"}}
	svc := NewStarterService(StarterServiceConfig{Gateway: gateway})

	result, err := svc.LaunchStarter(context.Background(), "  starter:lantern  ", LaunchStarterInput{AIAgentID: "agent-1"})
	if err != nil {
		t.Fatalf("LaunchStarter() error = %v", err)
	}
	if gateway.lastLaunchKey != "starter:lantern" {
		t.Fatalf("launch key = %q, want %q", gateway.lastLaunchKey, "starter:lantern")
	}
	if gateway.lastLaunchIn.AIAgentID != "agent-1" {
		t.Fatalf("launch input = %#v, want ai agent id agent-1", gateway.lastLaunchIn)
	}
	if result.CampaignID != "camp-777" {
		t.Fatalf("result.CampaignID = %q, want %q", result.CampaignID, "camp-777")
	}
}

func TestStarterServiceLaunchRejectsEmptyKey(t *testing.T) {
	t.Parallel()

	svc := NewStarterService(StarterServiceConfig{Gateway: &starterGatewayStub{}})

	_, err := svc.LaunchStarter(context.Background(), "", LaunchStarterInput{AIAgentID: "agent-1"})
	if err == nil {
		t.Fatal("expected LaunchStarter() error")
	}
	if got := apperrors.HTTPStatus(err); got != 400 {
		t.Fatalf("LaunchStarter() HTTP status = %d, want %d", got, 400)
	}
}
