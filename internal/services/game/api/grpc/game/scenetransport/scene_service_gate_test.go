package scenetransport

import (
	"context"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc/codes"
)

func TestOpenSceneGate_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.OpenSceneGate(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestResolveSceneGate_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.ResolveSceneGate(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAbandonSceneGate_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.AbandonSceneGate(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestOpenSceneGate_MissingCampaignId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.OpenSceneGate(context.Background(), &statev1.OpenSceneGateRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestResolveSceneGate_MissingCampaignId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.ResolveSceneGate(context.Background(), &statev1.ResolveSceneGateRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAbandonSceneGate_MissingCampaignId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.AbandonSceneGate(context.Background(), &statev1.AbandonSceneGateRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestOpenSceneGate_MissingSceneId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.OpenSceneGate(context.Background(), &statev1.OpenSceneGateRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestOpenSceneGate_MissingGateType(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.OpenSceneGate(context.Background(), &statev1.OpenSceneGateRequest{
		CampaignId: "c1", SceneId: "sc-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestResolveSceneGate_MissingSceneId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.ResolveSceneGate(context.Background(), &statev1.ResolveSceneGateRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestResolveSceneGate_MissingGateId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.ResolveSceneGate(context.Background(), &statev1.ResolveSceneGateRequest{
		CampaignId: "c1", SceneId: "sc-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAbandonSceneGate_MissingSceneId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.AbandonSceneGate(context.Background(), &statev1.AbandonSceneGateRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAbandonSceneGate_MissingGateId(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.AbandonSceneGate(context.Background(), &statev1.AbandonSceneGateRequest{
		CampaignId: "c1", SceneId: "sc-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}
