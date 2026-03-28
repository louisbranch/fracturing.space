package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type authUserClientStub struct {
	resp       *authv1.GetUserResponse
	err        error
	lastUserID string
}

func (stub *authUserClientStub) GetUser(_ context.Context, req *authv1.GetUserRequest, _ ...grpc.CallOption) (*authv1.GetUserResponse, error) {
	stub.lastUserID = req.GetUserId()
	return stub.resp, stub.err
}

func TestCommandActorTypeForEventActor(t *testing.T) {
	if got := CommandActorTypeForEventActor(event.ActorTypeParticipant); got != command.ActorTypeParticipant {
		t.Fatalf("participant actor type = %q", got)
	}
	if got := CommandActorTypeForEventActor(event.ActorTypeGM); got != command.ActorTypeGM {
		t.Fatalf("gm actor type = %q", got)
	}
	if got := CommandActorTypeForEventActor(event.ActorType("mystery")); got != command.ActorTypeSystem {
		t.Fatalf("default actor type = %q", got)
	}
}

func TestAuthUsername(t *testing.T) {
	t.Run("empty user id is ignored", func(t *testing.T) {
		username, err := AuthUsername(context.Background(), nil, "  ", nil)
		if err != nil || username != "" {
			t.Fatalf("AuthUsername(empty) = (%q, %v)", username, err)
		}
	})

	t.Run("missing client is internal error", func(t *testing.T) {
		_, err := AuthUsername(context.Background(), nil, "user-1", nil)
		if status.Code(err) != codes.Internal {
			t.Fatalf("status code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("not found maps to caller error", func(t *testing.T) {
		notFoundErr := status.Error(codes.NotFound, "participant user missing")
		_, err := AuthUsername(context.Background(), &authUserClientStub{
			err: status.Error(codes.NotFound, "auth missing"),
		}, "user-1", notFoundErr)
		if err != notFoundErr {
			t.Fatalf("error = %v, want caller-provided error", err)
		}
	})

	t.Run("missing response is internal error", func(t *testing.T) {
		_, err := AuthUsername(context.Background(), &authUserClientStub{}, "user-1", nil)
		if status.Code(err) != codes.Internal || status.Convert(err).Message() != "auth user response is missing" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("missing username is internal error", func(t *testing.T) {
		_, err := AuthUsername(context.Background(), &authUserClientStub{
			resp: &authv1.GetUserResponse{User: &authv1.User{}},
		}, "user-1", nil)
		if status.Code(err) != codes.Internal || status.Convert(err).Message() != "auth user username is missing" {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("success trims user id and returns username", func(t *testing.T) {
		client := &authUserClientStub{
			resp: &authv1.GetUserResponse{User: &authv1.User{Username: "aria"}},
		}
		username, err := AuthUsername(context.Background(), client, " user-1 ", nil)
		if err != nil {
			t.Fatalf("AuthUsername() error = %v", err)
		}
		if username != "aria" || client.lastUserID != "user-1" {
			t.Fatalf("username/request = (%q, %q)", username, client.lastUserID)
		}
	})
}

func TestResolveCommandActor(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, " participant-1 "))
	actorID, actorType := ResolveCommandActor(ctx)
	if actorID != "participant-1" || actorType != command.ActorTypeParticipant {
		t.Fatalf("ResolveCommandActor(with participant) = (%q, %q)", actorID, actorType)
	}

	actorID, actorType = ResolveCommandActor(context.Background())
	if actorID != "" || actorType != command.ActorTypeSystem {
		t.Fatalf("ResolveCommandActor(without participant) = (%q, %q)", actorID, actorType)
	}
}

func TestMapperHelpers(t *testing.T) {
	timestamp := time.Date(2026, time.March, 27, 8, 30, 0, 0, time.FixedZone("EDT", -4*60*60))
	got := TimestampOrNil(&timestamp)
	if got == nil || got.AsTime().UTC() != timestamp.UTC() {
		t.Fatalf("TimestampOrNil() = %#v", got)
	}
	if TimestampOrNil(nil) != nil {
		t.Fatal("TimestampOrNil(nil) = non-nil, want nil")
	}

	payload, err := structpb.NewStruct(map[string]any{"name": "Aria", "score": 3})
	if err != nil {
		t.Fatalf("NewStruct() error = %v", err)
	}
	values := StructToMap(payload)
	if values["name"] != "Aria" {
		t.Fatalf("StructToMap() = %#v", values)
	}
	if StructToMap(nil) != nil {
		t.Fatal("StructToMap(nil) = non-nil, want nil")
	}

	if err := ValidateStructPayload(map[string]any{"ok": 1, "two": 2}); err != nil {
		t.Fatalf("ValidateStructPayload(valid) error = %v", err)
	}
	if err := ValidateStructPayload(map[string]any{" ": 1}); err == nil || err.Error() != "payload keys must be non-empty" {
		t.Fatalf("ValidateStructPayload(invalid) error = %v", err)
	}
}

func TestDefaultPronounHelpers(t *testing.T) {
	if got := DefaultUnknownParticipantPronouns(); got == "" {
		t.Fatal("DefaultUnknownParticipantPronouns() returned empty value")
	}
	if got := DefaultAIParticipantPronouns(); got == "" {
		t.Fatal("DefaultAIParticipantPronouns() returned empty value")
	}
	if DefaultUnknownParticipantPronouns() == DefaultAIParticipantPronouns() {
		t.Fatal("default pronoun helpers returned the same value")
	}
}

func TestSystemHelpers(t *testing.T) {
	if got := SystemIDFromGameSystemProto(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART); got != systembridge.SystemIDDaggerheart {
		t.Fatalf("SystemIDFromGameSystemProto(daggerheart) = %q", got)
	}
	if got := SystemIDFromGameSystemProto(commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED); got != systembridge.SystemIDUnspecified {
		t.Fatalf("SystemIDFromGameSystemProto(unspecified) = %q", got)
	}

	if got := SystemIDFromCampaignRecord(storage.CampaignRecord{System: systembridge.SystemIDDaggerheart}); got != systembridge.SystemIDDaggerheart {
		t.Fatalf("SystemIDFromCampaignRecord(daggerheart) = %q", got)
	}
	if got := SystemIDFromCampaignRecord(storage.CampaignRecord{System: systembridge.SystemID("unknown")}); got != systembridge.SystemIDUnspecified {
		t.Fatalf("SystemIDFromCampaignRecord(unknown) = %q", got)
	}
}

func TestApplyErrorWithCodePreserve(t *testing.T) {
	handler := ApplyErrorWithCodePreserve("apply event")

	domainErr := apperrors.New(apperrors.CodeNotFound, "missing")
	if err := handler(domainErr); err != domainErr {
		t.Fatalf("wrapped error = %v, want original domain error", err)
	}

	if err := handler(errors.New("boom")); status.Code(err) != codes.Internal {
		t.Fatalf("wrapped unknown status code = %v, want Internal", status.Code(err))
	}
}
