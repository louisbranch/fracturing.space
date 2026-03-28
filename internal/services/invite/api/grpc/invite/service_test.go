package invite

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/invite/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/joingrant"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"github.com/louisbranch/fracturing.space/internal/test/mock/invitefakes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

var testClock = func() time.Time {
	return time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
}

func newIDGenerator() func() (string, error) {
	var seq atomic.Int64
	return func() (string, error) {
		n := seq.Add(1)
		return fmt.Sprintf("test-id-%d", n), nil
	}
}

func newTestService(store *invitefakes.InviteStore, outbox storage.OutboxStore) *Service {
	return NewService(Deps{
		Store:       store,
		Outbox:      outbox,
		IDGenerator: newIDGenerator(),
		Clock:       testClock,
	})
}

func ctxWithUserID(userID string) context.Context {
	md := metadata.Pairs(grpcmeta.UserIDHeader, userID)
	return metadata.NewIncomingContext(context.Background(), md)
}

// pendingInvite returns a seeded pending invite record for tests that need one
// pre-populated in the store.
func pendingInvite() storage.InviteRecord {
	return storage.InviteRecord{
		ID:              "inv-1",
		CampaignID:      "camp-1",
		ParticipantID:   "part-1",
		RecipientUserID: "",
		Status:          storage.StatusPending,
		CreatedAt:       testClock().Add(-time.Hour),
		UpdatedAt:       testClock().Add(-time.Hour),
	}
}

// seedInvite places an invite record into the fake store.
func seedInvite(store *invitefakes.InviteStore, rec storage.InviteRecord) {
	store.Invites[rec.ID] = rec
}

// ---------------------------------------------------------------------------
// Minimal gRPC client fakes for game service
// ---------------------------------------------------------------------------

// fakeParticipantClient implements gamev1.ParticipantServiceClient for tests
// that need the game dependency non-nil.
type fakeParticipantClient struct {
	gamev1.ParticipantServiceClient // embed to satisfy interface

	getResp  *gamev1.GetParticipantResponse
	getErr   error
	bindResp *gamev1.BindParticipantResponse
	bindErr  error
	listResp *gamev1.ListParticipantsResponse
	listErr  error
}

func (f *fakeParticipantClient) GetParticipant(_ context.Context, _ *gamev1.GetParticipantRequest, _ ...grpc.CallOption) (*gamev1.GetParticipantResponse, error) {
	return f.getResp, f.getErr
}

func (f *fakeParticipantClient) BindParticipant(_ context.Context, _ *gamev1.BindParticipantRequest, _ ...grpc.CallOption) (*gamev1.BindParticipantResponse, error) {
	return f.bindResp, f.bindErr
}

func (f *fakeParticipantClient) ListParticipants(_ context.Context, _ *gamev1.ListParticipantsRequest, _ ...grpc.CallOption) (*gamev1.ListParticipantsResponse, error) {
	return f.listResp, f.listErr
}

// fakeVerifier implements joingrant.Verifier for testing.
type fakeVerifier struct {
	err error
}

func (v *fakeVerifier) Validate(_ string, _ joingrant.Expectation) (joingrant.Claims, error) {
	return joingrant.Claims{}, v.err
}

// ===========================================================================
// CreateInvite tests
// ===========================================================================

func TestCreateInvite_Validation(t *testing.T) {
	t.Parallel()
	svc := newTestService(invitefakes.NewInviteStore(), nil)

	tests := []struct {
		name string
		req  *invitev1.CreateInviteRequest
		code codes.Code
	}{
		{"nil_request", nil, codes.InvalidArgument},
		{"empty_campaign_id", &invitev1.CreateInviteRequest{ParticipantId: "p"}, codes.InvalidArgument},
		{"empty_participant_id", &invitev1.CreateInviteRequest{CampaignId: "c"}, codes.InvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.CreateInvite(context.Background(), tt.req)
			grpcassert.StatusCode(t, err, tt.code)
		})
	}
}

func TestCreateInvite_HappyPath_NilGame(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	outbox := invitefakes.NewOutboxStore()
	svc := newTestService(store, outbox)

	resp, err := svc.CreateInvite(context.Background(), &invitev1.CreateInviteRequest{
		CampaignId:    "camp-1",
		ParticipantId: "part-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inv := resp.GetInvite()
	if inv == nil {
		t.Fatal("expected invite in response")
	}
	if inv.CampaignId != "camp-1" {
		t.Fatalf("campaign_id = %q, want camp-1", inv.CampaignId)
	}
	if inv.ParticipantId != "part-1" {
		t.Fatalf("participant_id = %q, want part-1", inv.ParticipantId)
	}
	if inv.Status != invitev1.InviteStatus_PENDING {
		t.Fatalf("status = %v, want PENDING", inv.Status)
	}
	// Verify the record was persisted.
	if len(store.Invites) != 1 {
		t.Fatalf("store has %d invites, want 1", len(store.Invites))
	}
	// Verify outbox event was enqueued.
	if len(outbox.Events) != 1 {
		t.Fatalf("outbox has %d events, want 1", len(outbox.Events))
	}
	if outbox.Events[0].EventType != outboxEventCreated {
		t.Fatalf("event type = %q, want %q", outbox.Events[0].EventType, outboxEventCreated)
	}
}

func TestCreateInvite_NoOutboxWhenNil(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	svc := newTestService(store, nil)

	_, err := svc.CreateInvite(context.Background(), &invitev1.CreateInviteRequest{
		CampaignId:    "camp-1",
		ParticipantId: "part-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No outbox configured — should still succeed without panic.
}

func TestCreateInvite_IDGeneratorFailure(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	svc := NewService(Deps{
		Store:       store,
		IDGenerator: func() (string, error) { return "", errors.New("id boom") },
		Clock:       testClock,
	})

	_, err := svc.CreateInvite(context.Background(), &invitev1.CreateInviteRequest{
		CampaignId:    "camp-1",
		ParticipantId: "part-1",
	})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestCreateInvite_StorePutError(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	store.PutErr = errors.New("db write fail")
	svc := newTestService(store, nil)

	_, err := svc.CreateInvite(context.Background(), &invitev1.CreateInviteRequest{
		CampaignId:    "camp-1",
		ParticipantId: "part-1",
	})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestCreateInvite_WithGame_UnboundParticipant(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	game := &fakeParticipantClient{
		getResp: &gamev1.GetParticipantResponse{
			Participant: &gamev1.Participant{Id: "part-1", UserId: ""},
		},
	}
	svc := NewService(Deps{
		Store:       store,
		Game:        game,
		IDGenerator: newIDGenerator(),
		Clock:       testClock,
	})

	resp, err := svc.CreateInvite(context.Background(), &invitev1.CreateInviteRequest{
		CampaignId:    "camp-1",
		ParticipantId: "part-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetInvite() == nil {
		t.Fatal("expected invite in response")
	}
}

func TestCreateInvite_WithGame_AlreadyBound(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	game := &fakeParticipantClient{
		getResp: &gamev1.GetParticipantResponse{
			Participant: &gamev1.Participant{Id: "part-1", UserId: "user-occupied"},
		},
	}
	svc := NewService(Deps{
		Store:       store,
		Game:        game,
		IDGenerator: newIDGenerator(),
		Clock:       testClock,
	})

	_, err := svc.CreateInvite(context.Background(), &invitev1.CreateInviteRequest{
		CampaignId:    "camp-1",
		ParticipantId: "part-1",
	})
	grpcassert.StatusCode(t, err, codes.AlreadyExists)
}

func TestCreateInvite_WithGame_ParticipantNotFound(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	game := &fakeParticipantClient{
		getErr: status.Error(codes.NotFound, "no such participant"),
	}
	svc := NewService(Deps{
		Store:       store,
		Game:        game,
		IDGenerator: newIDGenerator(),
		Clock:       testClock,
	})

	_, err := svc.CreateInvite(context.Background(), &invitev1.CreateInviteRequest{
		CampaignId:    "camp-1",
		ParticipantId: "part-1",
	})
	grpcassert.StatusCode(t, err, codes.NotFound)
}

func TestCreateInvite_WithGame_InternalError(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	game := &fakeParticipantClient{
		getErr: status.Error(codes.Unavailable, "game down"),
	}
	svc := NewService(Deps{
		Store:       store,
		Game:        game,
		IDGenerator: newIDGenerator(),
		Clock:       testClock,
	})

	_, err := svc.CreateInvite(context.Background(), &invitev1.CreateInviteRequest{
		CampaignId:    "camp-1",
		ParticipantId: "part-1",
	})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestCreateInvite_RecipientAlreadySeated(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	game := &fakeParticipantClient{
		getResp: &gamev1.GetParticipantResponse{
			Participant: &gamev1.Participant{Id: "part-1", UserId: ""},
		},
		// ListParticipants returns a participant already bound to the recipient.
		listResp: &gamev1.ListParticipantsResponse{
			Participants: []*gamev1.Participant{
				{Id: "part-other", UserId: "user-recipient"},
			},
		},
	}
	svc := NewService(Deps{
		Store:       store,
		Game:        game,
		IDGenerator: newIDGenerator(),
		Clock:       testClock,
	})

	_, err := svc.CreateInvite(context.Background(), &invitev1.CreateInviteRequest{
		CampaignId:      "camp-1",
		ParticipantId:   "part-1",
		RecipientUserId: "user-recipient",
	})
	grpcassert.StatusCode(t, err, codes.FailedPrecondition)
}

func TestCreateInvite_RecipientNotSeated(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	game := &fakeParticipantClient{
		getResp: &gamev1.GetParticipantResponse{
			Participant: &gamev1.Participant{Id: "part-1", UserId: ""},
		},
		// No matching user in participants list.
		listResp: &gamev1.ListParticipantsResponse{
			Participants: []*gamev1.Participant{
				{Id: "part-other", UserId: "different-user"},
			},
		},
	}
	svc := NewService(Deps{
		Store:       store,
		Game:        game,
		IDGenerator: newIDGenerator(),
		Clock:       testClock,
	})

	resp, err := svc.CreateInvite(context.Background(), &invitev1.CreateInviteRequest{
		CampaignId:      "camp-1",
		ParticipantId:   "part-1",
		RecipientUserId: "user-recipient",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetInvite().GetRecipientUserId() != "user-recipient" {
		t.Fatalf("recipient = %q, want user-recipient", resp.GetInvite().GetRecipientUserId())
	}
}

// ===========================================================================
// ClaimInvite tests
// ===========================================================================

func TestClaimInvite_Validation(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	svc := newTestService(store, nil)

	tests := []struct {
		name string
		ctx  context.Context
		req  *invitev1.ClaimInviteRequest
		code codes.Code
	}{
		{"nil_request", context.Background(), nil, codes.InvalidArgument},
		{"empty_campaign_id", context.Background(), &invitev1.ClaimInviteRequest{InviteId: "i", JoinGrant: "g"}, codes.InvalidArgument},
		{"empty_invite_id", context.Background(), &invitev1.ClaimInviteRequest{CampaignId: "c", JoinGrant: "g"}, codes.InvalidArgument},
		{"empty_join_grant", context.Background(), &invitev1.ClaimInviteRequest{CampaignId: "c", InviteId: "i"}, codes.InvalidArgument},
		{"no_user_identity", context.Background(), &invitev1.ClaimInviteRequest{CampaignId: "c", InviteId: "i", JoinGrant: "g"}, codes.Unauthenticated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.ClaimInvite(tt.ctx, tt.req)
			grpcassert.StatusCode(t, err, tt.code)
		})
	}
}

func TestClaimInvite_InviteNotFound(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	svc := newTestService(store, nil)
	ctx := ctxWithUserID("user-1")

	_, err := svc.ClaimInvite(ctx, &invitev1.ClaimInviteRequest{
		CampaignId: "camp-1",
		InviteId:   "inv-missing",
		JoinGrant:  "grant-token",
	})
	grpcassert.StatusCode(t, err, codes.NotFound)
}

func TestClaimInvite_StoreGetError(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	store.GetErr = errors.New("db read fail")
	svc := newTestService(store, nil)
	ctx := ctxWithUserID("user-1")

	_, err := svc.ClaimInvite(ctx, &invitev1.ClaimInviteRequest{
		CampaignId: "camp-1",
		InviteId:   "inv-1",
		JoinGrant:  "grant-token",
	})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestClaimInvite_CampaignMismatch(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	inv := pendingInvite()
	seedInvite(store, inv)
	svc := newTestService(store, nil)
	ctx := ctxWithUserID("user-1")

	_, err := svc.ClaimInvite(ctx, &invitev1.ClaimInviteRequest{
		CampaignId: "different-camp",
		InviteId:   inv.ID,
		JoinGrant:  "grant-token",
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestClaimInvite_NotPending(t *testing.T) {
	t.Parallel()

	for _, st := range []storage.Status{storage.StatusClaimed, storage.StatusRevoked, storage.StatusDeclined} {
		t.Run(string(st), func(t *testing.T) {
			t.Parallel()
			store := invitefakes.NewInviteStore()
			inv := pendingInvite()
			inv.Status = st
			seedInvite(store, inv)
			svc := newTestService(store, nil)
			ctx := ctxWithUserID("user-1")

			_, err := svc.ClaimInvite(ctx, &invitev1.ClaimInviteRequest{
				CampaignId: inv.CampaignID,
				InviteId:   inv.ID,
				JoinGrant:  "grant-token",
			})
			grpcassert.StatusCode(t, err, codes.FailedPrecondition)
		})
	}
}

func TestClaimInvite_RecipientMismatch(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	inv := pendingInvite()
	inv.RecipientUserID = "user-specific"
	seedInvite(store, inv)
	svc := newTestService(store, nil)
	ctx := ctxWithUserID("user-wrong")

	_, err := svc.ClaimInvite(ctx, &invitev1.ClaimInviteRequest{
		CampaignId: inv.CampaignID,
		InviteId:   inv.ID,
		JoinGrant:  "grant-token",
	})
	grpcassert.StatusCode(t, err, codes.PermissionDenied)
}

func TestClaimInvite_VerifierRejects(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	inv := pendingInvite()
	seedInvite(store, inv)
	svc := NewService(Deps{
		Store:       store,
		Game:        &fakeParticipantClient{bindResp: &gamev1.BindParticipantResponse{}},
		IDGenerator: newIDGenerator(),
		Clock:       testClock,
		Verifier:    &fakeVerifier{err: errors.New("bad grant")},
	})
	ctx := ctxWithUserID("user-1")

	_, err := svc.ClaimInvite(ctx, &invitev1.ClaimInviteRequest{
		CampaignId: inv.CampaignID,
		InviteId:   inv.ID,
		JoinGrant:  "bad-token",
	})
	grpcassert.StatusCode(t, err, codes.PermissionDenied)
}

func TestClaimInvite_BindParticipantError(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	inv := pendingInvite()
	seedInvite(store, inv)
	game := &fakeParticipantClient{
		bindErr: status.Error(codes.AlreadyExists, "already bound"),
	}
	svc := NewService(Deps{
		Store:       store,
		Game:        game,
		IDGenerator: newIDGenerator(),
		Clock:       testClock,
	})
	ctx := ctxWithUserID("user-1")

	_, err := svc.ClaimInvite(ctx, &invitev1.ClaimInviteRequest{
		CampaignId: inv.CampaignID,
		InviteId:   inv.ID,
		JoinGrant:  "grant-token",
	})
	// Invariant: gRPC status from game service is propagated directly.
	grpcassert.StatusCode(t, err, codes.AlreadyExists)
}

func TestClaimInvite_UpdateStatusError(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	inv := pendingInvite()
	seedInvite(store, inv)
	store.UpdateErr = errors.New("db update fail")
	game := &fakeParticipantClient{
		bindResp: &gamev1.BindParticipantResponse{
			Participant: &gamev1.Participant{Id: inv.ParticipantID, UserId: "user-1"},
		},
	}
	svc := NewService(Deps{
		Store:       store,
		Game:        game,
		IDGenerator: newIDGenerator(),
		Clock:       testClock,
	})
	ctx := ctxWithUserID("user-1")

	_, err := svc.ClaimInvite(ctx, &invitev1.ClaimInviteRequest{
		CampaignId: inv.CampaignID,
		InviteId:   inv.ID,
		JoinGrant:  "grant-token",
	})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestClaimInvite_HappyPath(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	outbox := invitefakes.NewOutboxStore()
	inv := pendingInvite()
	seedInvite(store, inv)
	game := &fakeParticipantClient{
		bindResp: &gamev1.BindParticipantResponse{
			Participant: &gamev1.Participant{
				Id:     inv.ParticipantID,
				Name:   "Hero",
				UserId: "user-1",
			},
		},
	}
	svc := NewService(Deps{
		Store:       store,
		Outbox:      outbox,
		Game:        game,
		IDGenerator: newIDGenerator(),
		Clock:       testClock,
	})
	ctx := ctxWithUserID("user-1")

	resp, err := svc.ClaimInvite(ctx, &invitev1.ClaimInviteRequest{
		CampaignId: inv.CampaignID,
		InviteId:   inv.ID,
		JoinGrant:  "grant-token",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetInvite().GetStatus() != invitev1.InviteStatus_CLAIMED {
		t.Fatalf("status = %v, want CLAIMED", resp.GetInvite().GetStatus())
	}
	if resp.GetParticipant() == nil {
		t.Fatal("expected participant summary in response")
	}
	if resp.GetParticipant().GetName() != "Hero" {
		t.Fatalf("participant name = %q, want Hero", resp.GetParticipant().GetName())
	}
	// Verify store was updated.
	stored := store.Invites[inv.ID]
	if stored.Status != storage.StatusClaimed {
		t.Fatalf("stored status = %v, want claimed", stored.Status)
	}
	// Verify outbox event.
	if len(outbox.Events) != 1 {
		t.Fatalf("outbox has %d events, want 1", len(outbox.Events))
	}
	if outbox.Events[0].EventType != outboxEventClaimed {
		t.Fatalf("event type = %q, want %q", outbox.Events[0].EventType, outboxEventClaimed)
	}
}

func TestClaimInvite_RecipientMatchesUser(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	inv := pendingInvite()
	inv.RecipientUserID = "user-1"
	seedInvite(store, inv)
	game := &fakeParticipantClient{
		bindResp: &gamev1.BindParticipantResponse{
			Participant: &gamev1.Participant{Id: inv.ParticipantID, UserId: "user-1"},
		},
	}
	svc := NewService(Deps{
		Store:       store,
		Game:        game,
		IDGenerator: newIDGenerator(),
		Clock:       testClock,
	})
	ctx := ctxWithUserID("user-1")

	resp, err := svc.ClaimInvite(ctx, &invitev1.ClaimInviteRequest{
		CampaignId: inv.CampaignID,
		InviteId:   inv.ID,
		JoinGrant:  "grant-token",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetInvite().GetStatus() != invitev1.InviteStatus_CLAIMED {
		t.Fatalf("status = %v, want CLAIMED", resp.GetInvite().GetStatus())
	}
}

// ===========================================================================
// DeclineInvite tests
// ===========================================================================

func TestDeclineInvite_Validation(t *testing.T) {
	t.Parallel()
	svc := newTestService(invitefakes.NewInviteStore(), nil)

	tests := []struct {
		name string
		req  *invitev1.DeclineInviteRequest
		code codes.Code
	}{
		{"nil_request", nil, codes.InvalidArgument},
		{"empty_invite_id", &invitev1.DeclineInviteRequest{}, codes.InvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.DeclineInvite(context.Background(), tt.req)
			grpcassert.StatusCode(t, err, tt.code)
		})
	}
}

func TestDeclineInvite_InviteNotFound(t *testing.T) {
	t.Parallel()
	svc := newTestService(invitefakes.NewInviteStore(), nil)

	_, err := svc.DeclineInvite(context.Background(), &invitev1.DeclineInviteRequest{InviteId: "missing"})
	grpcassert.StatusCode(t, err, codes.NotFound)
}

func TestDeclineInvite_StoreGetError(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	store.GetErr = errors.New("db fail")
	svc := newTestService(store, nil)

	_, err := svc.DeclineInvite(context.Background(), &invitev1.DeclineInviteRequest{InviteId: "inv-1"})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestDeclineInvite_NotPending(t *testing.T) {
	t.Parallel()

	for _, st := range []storage.Status{storage.StatusClaimed, storage.StatusRevoked, storage.StatusDeclined} {
		t.Run(string(st), func(t *testing.T) {
			t.Parallel()
			store := invitefakes.NewInviteStore()
			inv := pendingInvite()
			inv.Status = st
			seedInvite(store, inv)
			svc := newTestService(store, nil)

			_, err := svc.DeclineInvite(context.Background(), &invitev1.DeclineInviteRequest{InviteId: inv.ID})
			grpcassert.StatusCode(t, err, codes.FailedPrecondition)
		})
	}
}

func TestDeclineInvite_UpdateStatusError(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	inv := pendingInvite()
	seedInvite(store, inv)
	store.UpdateErr = errors.New("db update fail")
	svc := newTestService(store, nil)

	_, err := svc.DeclineInvite(context.Background(), &invitev1.DeclineInviteRequest{InviteId: inv.ID})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestDeclineInvite_HappyPath(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	outbox := invitefakes.NewOutboxStore()
	inv := pendingInvite()
	seedInvite(store, inv)
	svc := newTestService(store, outbox)

	resp, err := svc.DeclineInvite(context.Background(), &invitev1.DeclineInviteRequest{InviteId: inv.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetInvite().GetStatus() != invitev1.InviteStatus_DECLINED {
		t.Fatalf("status = %v, want DECLINED", resp.GetInvite().GetStatus())
	}
	// Verify store was updated.
	stored := store.Invites[inv.ID]
	if stored.Status != storage.StatusDeclined {
		t.Fatalf("stored status = %v, want declined", stored.Status)
	}
	// Verify outbox event.
	if len(outbox.Events) != 1 {
		t.Fatalf("outbox has %d events, want 1", len(outbox.Events))
	}
	if outbox.Events[0].EventType != outboxEventDeclined {
		t.Fatalf("event type = %q, want %q", outbox.Events[0].EventType, outboxEventDeclined)
	}
}

// ===========================================================================
// RevokeInvite tests
// ===========================================================================

func TestRevokeInvite_Validation(t *testing.T) {
	t.Parallel()
	svc := newTestService(invitefakes.NewInviteStore(), nil)

	tests := []struct {
		name string
		req  *invitev1.RevokeInviteRequest
		code codes.Code
	}{
		{"nil_request", nil, codes.InvalidArgument},
		{"empty_invite_id", &invitev1.RevokeInviteRequest{}, codes.InvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.RevokeInvite(context.Background(), tt.req)
			grpcassert.StatusCode(t, err, tt.code)
		})
	}
}

func TestRevokeInvite_InviteNotFound(t *testing.T) {
	t.Parallel()
	svc := newTestService(invitefakes.NewInviteStore(), nil)

	_, err := svc.RevokeInvite(context.Background(), &invitev1.RevokeInviteRequest{InviteId: "missing"})
	grpcassert.StatusCode(t, err, codes.NotFound)
}

func TestRevokeInvite_StoreGetError(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	store.GetErr = errors.New("db fail")
	svc := newTestService(store, nil)

	_, err := svc.RevokeInvite(context.Background(), &invitev1.RevokeInviteRequest{InviteId: "inv-1"})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestRevokeInvite_NotPending(t *testing.T) {
	t.Parallel()

	for _, st := range []storage.Status{storage.StatusClaimed, storage.StatusRevoked, storage.StatusDeclined} {
		t.Run(string(st), func(t *testing.T) {
			t.Parallel()
			store := invitefakes.NewInviteStore()
			inv := pendingInvite()
			inv.Status = st
			seedInvite(store, inv)
			svc := newTestService(store, nil)

			_, err := svc.RevokeInvite(context.Background(), &invitev1.RevokeInviteRequest{InviteId: inv.ID})
			grpcassert.StatusCode(t, err, codes.FailedPrecondition)
		})
	}
}

func TestRevokeInvite_UpdateStatusError(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	inv := pendingInvite()
	seedInvite(store, inv)
	store.UpdateErr = errors.New("db update fail")
	svc := newTestService(store, nil)

	_, err := svc.RevokeInvite(context.Background(), &invitev1.RevokeInviteRequest{InviteId: inv.ID})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestRevokeInvite_HappyPath(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	inv := pendingInvite()
	seedInvite(store, inv)
	svc := newTestService(store, nil)

	resp, err := svc.RevokeInvite(context.Background(), &invitev1.RevokeInviteRequest{InviteId: inv.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetInvite().GetStatus() != invitev1.InviteStatus_REVOKED {
		t.Fatalf("status = %v, want REVOKED", resp.GetInvite().GetStatus())
	}
	// Verify store was updated.
	stored := store.Invites[inv.ID]
	if stored.Status != storage.StatusRevoked {
		t.Fatalf("stored status = %v, want revoked", stored.Status)
	}
}

func TestRevokeInvite_NoOutbox(t *testing.T) {
	// Invariant: RevokeInvite does not enqueue outbox events (unlike
	// decline/claim). This test documents that behavior.
	t.Parallel()
	store := invitefakes.NewInviteStore()
	outbox := invitefakes.NewOutboxStore()
	inv := pendingInvite()
	seedInvite(store, inv)
	svc := newTestService(store, outbox)

	_, err := svc.RevokeInvite(context.Background(), &invitev1.RevokeInviteRequest{InviteId: inv.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(outbox.Events) != 0 {
		t.Fatalf("outbox has %d events, want 0", len(outbox.Events))
	}
}

// ===========================================================================
// GetInvite tests
// ===========================================================================

func TestGetInvite_Validation(t *testing.T) {
	t.Parallel()
	svc := newTestService(invitefakes.NewInviteStore(), nil)

	tests := []struct {
		name string
		req  *invitev1.GetInviteRequest
		code codes.Code
	}{
		{"nil_request", nil, codes.InvalidArgument},
		{"empty_invite_id", &invitev1.GetInviteRequest{}, codes.InvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.GetInvite(context.Background(), tt.req)
			grpcassert.StatusCode(t, err, tt.code)
		})
	}
}

func TestGetInvite_NotFound(t *testing.T) {
	t.Parallel()
	svc := newTestService(invitefakes.NewInviteStore(), nil)

	_, err := svc.GetInvite(context.Background(), &invitev1.GetInviteRequest{InviteId: "missing"})
	grpcassert.StatusCode(t, err, codes.NotFound)
}

func TestGetInvite_StoreError(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	store.GetErr = errors.New("db fail")
	svc := newTestService(store, nil)

	_, err := svc.GetInvite(context.Background(), &invitev1.GetInviteRequest{InviteId: "inv-1"})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestGetInvite_HappyPath(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	inv := pendingInvite()
	seedInvite(store, inv)
	svc := newTestService(store, nil)

	resp, err := svc.GetInvite(context.Background(), &invitev1.GetInviteRequest{InviteId: inv.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetInvite().GetId() != inv.ID {
		t.Fatalf("invite id = %q, want %q", resp.GetInvite().GetId(), inv.ID)
	}
	if resp.GetInvite().GetCampaignId() != inv.CampaignID {
		t.Fatalf("campaign_id = %q, want %q", resp.GetInvite().GetCampaignId(), inv.CampaignID)
	}
}

// ===========================================================================
// GetPublicInvite tests
// ===========================================================================

func TestGetPublicInvite_Validation(t *testing.T) {
	t.Parallel()
	svc := newTestService(invitefakes.NewInviteStore(), nil)

	tests := []struct {
		name string
		req  *invitev1.GetPublicInviteRequest
		code codes.Code
	}{
		{"nil_request", nil, codes.InvalidArgument},
		{"empty_invite_id", &invitev1.GetPublicInviteRequest{}, codes.InvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.GetPublicInvite(context.Background(), tt.req)
			grpcassert.StatusCode(t, err, tt.code)
		})
	}
}

func TestGetPublicInvite_NotFound(t *testing.T) {
	t.Parallel()
	svc := newTestService(invitefakes.NewInviteStore(), nil)

	_, err := svc.GetPublicInvite(context.Background(), &invitev1.GetPublicInviteRequest{InviteId: "missing"})
	grpcassert.StatusCode(t, err, codes.NotFound)
}

func TestGetPublicInvite_HappyPath_NilDeps(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	inv := pendingInvite()
	seedInvite(store, inv)
	// All enrichment deps are nil — should return invite without campaign/participant/creator.
	svc := newTestService(store, nil)

	resp, err := svc.GetPublicInvite(context.Background(), &invitev1.GetPublicInviteRequest{InviteId: inv.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetInvite().GetId() != inv.ID {
		t.Fatalf("invite id = %q, want %q", resp.GetInvite().GetId(), inv.ID)
	}
	// No enrichment deps — summaries should be nil.
	if resp.Campaign != nil {
		t.Fatal("expected nil campaign summary when gameCampaign is nil")
	}
	if resp.Participant != nil {
		t.Fatal("expected nil participant summary when game is nil")
	}
	if resp.CreatedByUser != nil {
		t.Fatal("expected nil creator user when auth is nil")
	}
}

// ===========================================================================
// ListInvites tests
// ===========================================================================

func TestListInvites_Validation(t *testing.T) {
	t.Parallel()
	svc := newTestService(invitefakes.NewInviteStore(), nil)

	_, err := svc.ListInvites(context.Background(), nil)
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestListInvites_StoreError(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	store.ListErr = errors.New("db list fail")
	svc := newTestService(store, nil)

	_, err := svc.ListInvites(context.Background(), &invitev1.ListInvitesRequest{CampaignId: "camp-1"})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestListInvites_HappyPath(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	inv := pendingInvite()
	seedInvite(store, inv)
	svc := newTestService(store, nil)

	resp, err := svc.ListInvites(context.Background(), &invitev1.ListInvitesRequest{CampaignId: "camp-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetInvites()) != 1 {
		t.Fatalf("got %d invites, want 1", len(resp.GetInvites()))
	}
}

// ===========================================================================
// ListPendingInvites tests
// ===========================================================================

func TestListPendingInvites_Validation(t *testing.T) {
	t.Parallel()
	svc := newTestService(invitefakes.NewInviteStore(), nil)

	_, err := svc.ListPendingInvites(context.Background(), nil)
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestListPendingInvites_StoreError(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	store.ListErr = errors.New("db list fail")
	svc := newTestService(store, nil)

	_, err := svc.ListPendingInvites(context.Background(), &invitev1.ListPendingInvitesRequest{CampaignId: "camp-1"})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestListPendingInvites_HappyPath(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	inv := pendingInvite()
	seedInvite(store, inv)
	// Add a non-pending invite that should not appear.
	claimed := pendingInvite()
	claimed.ID = "inv-claimed"
	claimed.Status = storage.StatusClaimed
	seedInvite(store, claimed)
	svc := newTestService(store, nil)

	resp, err := svc.ListPendingInvites(context.Background(), &invitev1.ListPendingInvitesRequest{CampaignId: "camp-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetInvites()) != 1 {
		t.Fatalf("got %d invites, want 1 (pending only)", len(resp.GetInvites()))
	}
}

// ===========================================================================
// ListPendingInvitesForUser tests
// ===========================================================================

func TestListPendingInvitesForUser_Validation(t *testing.T) {
	t.Parallel()
	svc := newTestService(invitefakes.NewInviteStore(), nil)

	tests := []struct {
		name string
		ctx  context.Context
		req  *invitev1.ListPendingInvitesForUserRequest
		code codes.Code
	}{
		{"nil_request", context.Background(), nil, codes.InvalidArgument},
		{"no_user_identity", context.Background(), &invitev1.ListPendingInvitesForUserRequest{}, codes.Unauthenticated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.ListPendingInvitesForUser(tt.ctx, tt.req)
			grpcassert.StatusCode(t, err, tt.code)
		})
	}
}

func TestListPendingInvitesForUser_StoreError(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	store.ListErr = errors.New("db list fail")
	svc := newTestService(store, nil)
	ctx := ctxWithUserID("user-1")

	_, err := svc.ListPendingInvitesForUser(ctx, &invitev1.ListPendingInvitesForUserRequest{})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestListPendingInvitesForUser_HappyPath(t *testing.T) {
	t.Parallel()
	store := invitefakes.NewInviteStore()
	inv := pendingInvite()
	inv.RecipientUserID = "user-1"
	seedInvite(store, inv)
	svc := newTestService(store, nil)
	ctx := ctxWithUserID("user-1")

	resp, err := svc.ListPendingInvitesForUser(ctx, &invitev1.ListPendingInvitesForUserRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetInvites()) != 1 {
		t.Fatalf("got %d entries, want 1", len(resp.GetInvites()))
	}
	entry := resp.GetInvites()[0]
	if entry.GetInvite().GetId() != inv.ID {
		t.Fatalf("invite id = %q, want %q", entry.GetInvite().GetId(), inv.ID)
	}
	// No game/auth deps — enrichment fields should be nil.
	if entry.Campaign != nil {
		t.Fatal("expected nil campaign summary when gameCampaign is nil")
	}
	if entry.Participant != nil {
		t.Fatal("expected nil participant summary when game is nil")
	}
}

// ===========================================================================
// LeaseIntegrationOutboxEvents tests
// ===========================================================================

func TestLeaseOutboxEvents_Validation(t *testing.T) {
	t.Parallel()
	outbox := invitefakes.NewOutboxStore()
	svc := newTestService(invitefakes.NewInviteStore(), outbox)

	tests := []struct {
		name string
		req  *invitev1.LeaseIntegrationOutboxEventsRequest
		code codes.Code
	}{
		{"nil_request", nil, codes.InvalidArgument},
		{"empty_consumer", &invitev1.LeaseIntegrationOutboxEventsRequest{}, codes.InvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.LeaseIntegrationOutboxEvents(context.Background(), tt.req)
			grpcassert.StatusCode(t, err, tt.code)
		})
	}
}

func TestLeaseOutboxEvents_StoreError(t *testing.T) {
	t.Parallel()
	outbox := invitefakes.NewOutboxStore()
	outbox.LeaseErr = errors.New("lease fail")
	svc := newTestService(invitefakes.NewInviteStore(), outbox)

	_, err := svc.LeaseIntegrationOutboxEvents(context.Background(), &invitev1.LeaseIntegrationOutboxEventsRequest{
		Consumer: "worker-1",
	})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestLeaseOutboxEvents_HappyPath(t *testing.T) {
	t.Parallel()
	outbox := invitefakes.NewOutboxStore()
	outbox.Events = []storage.OutboxEvent{
		{ID: "evt-1", EventType: "invite.invite.created.v1", PayloadJSON: []byte(`{}`), DedupeKey: "k1", CreatedAt: testClock()},
	}
	svc := newTestService(invitefakes.NewInviteStore(), outbox)

	resp, err := svc.LeaseIntegrationOutboxEvents(context.Background(), &invitev1.LeaseIntegrationOutboxEventsRequest{
		Consumer: "worker-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetEvents()) != 1 {
		t.Fatalf("got %d events, want 1", len(resp.GetEvents()))
	}
	evt := resp.GetEvents()[0]
	if evt.GetId() != "evt-1" {
		t.Fatalf("event id = %q, want evt-1", evt.GetId())
	}
}

func TestLeaseOutboxEvents_DefaultsLimitAndTTL(t *testing.T) {
	t.Parallel()
	outbox := invitefakes.NewOutboxStore()
	svc := newTestService(invitefakes.NewInviteStore(), outbox)

	// Zero limit and ttl should use defaults and not fail.
	resp, err := svc.LeaseIntegrationOutboxEvents(context.Background(), &invitev1.LeaseIntegrationOutboxEventsRequest{
		Consumer:   "worker-1",
		Limit:      0,
		LeaseTtlMs: 0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestLeaseOutboxEvents_UsesNowFromRequest(t *testing.T) {
	t.Parallel()
	outbox := invitefakes.NewOutboxStore()
	svc := newTestService(invitefakes.NewInviteStore(), outbox)

	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	resp, err := svc.LeaseIntegrationOutboxEvents(context.Background(), &invitev1.LeaseIntegrationOutboxEventsRequest{
		Consumer: "worker-1",
		Now:      timestamppb.New(now),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

// ===========================================================================
// AckIntegrationOutboxEvent tests
// ===========================================================================

func TestAckOutboxEvent_Validation(t *testing.T) {
	t.Parallel()
	outbox := invitefakes.NewOutboxStore()
	svc := newTestService(invitefakes.NewInviteStore(), outbox)

	tests := []struct {
		name string
		req  *invitev1.AckIntegrationOutboxEventRequest
		code codes.Code
	}{
		{"nil_request", nil, codes.InvalidArgument},
		{"empty_event_id", &invitev1.AckIntegrationOutboxEventRequest{Consumer: "c", Outcome: invitev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_SUCCEEDED}, codes.InvalidArgument},
		{"empty_consumer", &invitev1.AckIntegrationOutboxEventRequest{EventId: "e", Outcome: invitev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_SUCCEEDED}, codes.InvalidArgument},
		{"unspecified_outcome", &invitev1.AckIntegrationOutboxEventRequest{EventId: "e", Consumer: "c"}, codes.InvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.AckIntegrationOutboxEvent(context.Background(), tt.req)
			grpcassert.StatusCode(t, err, tt.code)
		})
	}
}

func TestAckOutboxEvent_StoreError(t *testing.T) {
	t.Parallel()
	outbox := invitefakes.NewOutboxStore()
	outbox.AckErr = errors.New("ack fail")
	svc := newTestService(invitefakes.NewInviteStore(), outbox)

	_, err := svc.AckIntegrationOutboxEvent(context.Background(), &invitev1.AckIntegrationOutboxEventRequest{
		EventId:  "evt-1",
		Consumer: "worker-1",
		Outcome:  invitev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_SUCCEEDED,
	})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestAckOutboxEvent_HappyPath(t *testing.T) {
	t.Parallel()

	outcomes := []struct {
		name    string
		outcome invitev1.IntegrationOutboxAckOutcome
	}{
		{"succeeded", invitev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_SUCCEEDED},
		{"retry", invitev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_RETRY},
		{"dead", invitev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_DEAD},
	}
	for _, tt := range outcomes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			outbox := invitefakes.NewOutboxStore()
			svc := newTestService(invitefakes.NewInviteStore(), outbox)

			resp, err := svc.AckIntegrationOutboxEvent(context.Background(), &invitev1.AckIntegrationOutboxEventRequest{
				EventId:  "evt-1",
				Consumer: "worker-1",
				Outcome:  tt.outcome,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp == nil {
				t.Fatal("expected non-nil response")
			}
		})
	}
}
