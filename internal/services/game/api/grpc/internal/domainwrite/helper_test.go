package domainwrite

import (
	"errors"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewIntentFilter_SkipsAuditOnlyEvents(t *testing.T) {
	registry := event.NewRegistry()
	if err := registry.Register(event.Definition{
		Type:   event.Type("action.roll_resolved"),
		Owner:  event.OwnerCore,
		Intent: event.IntentProjectionAndReplay,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}
	if err := registry.Register(event.Definition{
		Type:   event.Type("story.note_added"),
		Owner:  event.OwnerCore,
		Intent: event.IntentAuditOnly,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}

	filter := NewIntentFilter(registry)

	tests := []struct {
		name      string
		eventType event.Type
		want      bool
	}{
		{"projection event applies", event.Type("action.roll_resolved"), true},
		{"audit-only event skipped", event.Type("story.note_added"), false},
		{"unknown event skipped (fail closed)", event.Type("custom.unknown"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filter(event.Event{Type: tt.eventType})
			if got != tt.want {
				t.Fatalf("filter(%s) = %t, want %t", tt.eventType, got, tt.want)
			}
		})
	}
}

func TestNewIntentFilter_SkipsReplayOnlyEvents(t *testing.T) {
	registry := event.NewRegistry()
	if err := registry.Register(event.Definition{
		Type:   event.Type("action.roll_resolved"),
		Owner:  event.OwnerCore,
		Intent: event.IntentReplayOnly,
	}); err != nil {
		t.Fatalf("register event: %v", err)
	}

	filter := NewIntentFilter(registry)
	if filter(event.Event{Type: event.Type("action.roll_resolved")}) {
		t.Fatal("expected replay-only event to be filtered out")
	}
}

func TestNewIntentFilter_NilRegistryFailsClosed(t *testing.T) {
	filter := NewIntentFilter(nil)

	if filter(event.Event{Type: event.Type("action.roll_resolved")}) {
		t.Fatal("expected nil registry filter to fail closed")
	}
}

func TestNormalizeErrorHandlers_DefaultMessages(t *testing.T) {
	executeErr, applyErr, rejectErr := NormalizeErrorHandlers(ErrorHandlerOptions{})

	executeStatus, ok := status.FromError(executeErr(errors.New("boom")))
	if !ok {
		t.Fatal("expected grpc status from execute error")
	}
	if executeStatus.Code() != codes.Internal {
		t.Fatalf("execute code = %s, want %s", executeStatus.Code(), codes.Internal)
	}
	if !strings.Contains(executeStatus.Message(), "execute domain command: boom") {
		t.Fatalf("execute message = %q", executeStatus.Message())
	}

	applyStatus, ok := status.FromError(applyErr(errors.New("boom")))
	if !ok {
		t.Fatal("expected grpc status from apply error")
	}
	if applyStatus.Code() != codes.Internal {
		t.Fatalf("apply code = %s, want %s", applyStatus.Code(), codes.Internal)
	}
	if !strings.Contains(applyStatus.Message(), "apply event: boom") {
		t.Fatalf("apply message = %q", applyStatus.Message())
	}

	rejectStatus, ok := status.FromError(rejectErr("nope"))
	if !ok {
		t.Fatal("expected grpc status from reject error")
	}
	if rejectStatus.Code() != codes.FailedPrecondition {
		t.Fatalf("reject code = %s, want %s", rejectStatus.Code(), codes.FailedPrecondition)
	}
}

func TestNormalizeErrorHandlers_MessageOverrides(t *testing.T) {
	executeErr, applyErr, _ := NormalizeErrorHandlers(ErrorHandlerOptions{
		ExecuteErrMessage: "exec custom",
		ApplyErrMessage:   "apply custom",
	})

	executeStatus, ok := status.FromError(executeErr(errors.New("boom")))
	if !ok {
		t.Fatal("expected grpc status from execute error")
	}
	if !strings.Contains(executeStatus.Message(), "exec custom: boom") {
		t.Fatalf("execute message = %q", executeStatus.Message())
	}

	applyStatus, ok := status.FromError(applyErr(errors.New("boom")))
	if !ok {
		t.Fatal("expected grpc status from apply error")
	}
	if !strings.Contains(applyStatus.Message(), "apply custom: boom") {
		t.Fatalf("apply message = %q", applyStatus.Message())
	}
}

func TestNormalizeErrorHandlers_CallbackOverrides(t *testing.T) {
	wantExecute := status.Error(codes.PermissionDenied, "custom execute")
	wantApply := status.Error(codes.Aborted, "custom apply")
	wantReject := status.Error(codes.AlreadyExists, "custom reject")

	executeErr, applyErr, rejectErr := NormalizeErrorHandlers(ErrorHandlerOptions{
		ExecuteErr: func(error) error { return wantExecute },
		ApplyErr:   func(error) error { return wantApply },
		RejectErr:  func(string) error { return wantReject },
	})

	if got := executeErr(errors.New("boom")); got != wantExecute {
		t.Fatalf("execute override mismatch: got %v, want %v", got, wantExecute)
	}
	if got := applyErr(errors.New("boom")); got != wantApply {
		t.Fatalf("apply override mismatch: got %v, want %v", got, wantApply)
	}
	if got := rejectErr("boom"); got != wantReject {
		t.Fatalf("reject override mismatch: got %v, want %v", got, wantReject)
	}
}
