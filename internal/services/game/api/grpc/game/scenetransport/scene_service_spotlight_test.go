package scenetransport

import (
	"context"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc/codes"
)

func TestSetSceneSpotlight_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.SetSceneSpotlight(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestClearSceneSpotlight_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.ClearSceneSpotlight(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetSceneSpotlight_MissingCampaignId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.SetSceneSpotlight(context.Background(), &statev1.SetSceneSpotlightRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestClearSceneSpotlight_MissingCampaignId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.ClearSceneSpotlight(context.Background(), &statev1.ClearSceneSpotlightRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetSceneSpotlight_MissingSceneId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.SetSceneSpotlight(context.Background(), &statev1.SetSceneSpotlightRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetSceneSpotlight_InvalidType(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.SetSceneSpotlight(context.Background(), &statev1.SetSceneSpotlightRequest{
		CampaignId: "c1", SceneId: "sc-1", Type: "invalid",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestClearSceneSpotlight_MissingSceneId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.ClearSceneSpotlight(context.Background(), &statev1.ClearSceneSpotlightRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}
