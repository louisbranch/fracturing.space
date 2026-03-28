package authz

import (
	"context"
	"testing"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type testCampaignStore struct {
	record storage.CampaignRecord
	err    error
}

func (s testCampaignStore) Put(context.Context, storage.CampaignRecord) error { return nil }

func (s testCampaignStore) Get(context.Context, string) (storage.CampaignRecord, error) {
	if s.err != nil {
		return storage.CampaignRecord{}, s.err
	}
	return s.record, nil
}

func (s testCampaignStore) List(context.Context, int, string) (storage.CampaignPage, error) {
	return storage.CampaignPage{}, nil
}

func TestEvaluatorEvaluate_NilRequest(t *testing.T) {
	evaluator := NewEvaluator(EvaluatorStores{})

	_, err := evaluator.Evaluate(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("error code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestEvaluatorEvaluate_DeniedActorReturnsCanResponse(t *testing.T) {
	evaluator := NewEvaluator(EvaluatorStores{
		Campaign: testCampaignStore{record: storage.CampaignRecord{ID: "camp-1"}},
		Participant: testParticipantStore{get: func(context.Context, string, string) (storage.ParticipantRecord, error) {
			return storage.ParticipantRecord{}, storage.ErrNotFound
		}},
		Audit: &testAuditStore{},
	})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "missing-seat"))

	resp, err := evaluator.Evaluate(ctx, &campaignv1.CanRequest{
		CampaignId: "camp-1",
		Action:     campaignv1.AuthorizationAction_AUTHORIZATION_ACTION_READ,
		Resource:   campaignv1.AuthorizationResource_AUTHORIZATION_RESOURCE_CAMPAIGN,
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if resp.GetAllowed() {
		t.Fatal("allowed = true, want false")
	}
	if resp.GetActorParticipantId() != "" {
		t.Fatalf("actor participant id = %q, want empty", resp.GetActorParticipantId())
	}
}
