package domainwrite

import (
	"errors"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Tests below verify that NormalizeErrorHandlers defaults produce plain errors
// (not gRPC status). Consumer packages are responsible for wrapping with gRPC
// status at the transport boundary.

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

	execResult := executeErr(errors.New("boom"))
	if !strings.Contains(execResult.Error(), "execute domain command: boom") {
		t.Fatalf("execute message = %q, want to contain %q", execResult.Error(), "execute domain command: boom")
	}
	if _, ok := status.FromError(execResult); ok && status.Code(execResult) != codes.OK {
		t.Fatal("default execute error should be a plain error, not gRPC status")
	}

	applyResult := applyErr(errors.New("boom"))
	if !strings.Contains(applyResult.Error(), "apply event: boom") {
		t.Fatalf("apply message = %q, want to contain %q", applyResult.Error(), "apply event: boom")
	}

	rejectResult := rejectErr("SOME_CODE", "nope")
	if rejectResult.Error() != "nope" {
		t.Fatalf("reject message = %q, want %q", rejectResult.Error(), "nope")
	}
}

func TestNormalizeErrorHandlers_MessageOverrides(t *testing.T) {
	executeErr, applyErr, _ := NormalizeErrorHandlers(ErrorHandlerOptions{
		ExecuteErrMessage: "exec custom",
		ApplyErrMessage:   "apply custom",
	})

	execResult := executeErr(errors.New("boom"))
	if !strings.Contains(execResult.Error(), "exec custom: boom") {
		t.Fatalf("execute message = %q, want to contain %q", execResult.Error(), "exec custom: boom")
	}

	applyResult := applyErr(errors.New("boom"))
	if !strings.Contains(applyResult.Error(), "apply custom: boom") {
		t.Fatalf("apply message = %q, want to contain %q", applyResult.Error(), "apply custom: boom")
	}
}

func TestNormalizeErrorHandlers_CallbackOverrides(t *testing.T) {
	wantExecute := status.Error(codes.PermissionDenied, "custom execute")
	wantApply := status.Error(codes.Aborted, "custom apply")
	wantReject := status.Error(codes.AlreadyExists, "custom reject")

	executeErr, applyErr, rejectErr := NormalizeErrorHandlers(ErrorHandlerOptions{
		ExecuteErr: func(error) error { return wantExecute },
		ApplyErr:   func(error) error { return wantApply },
		RejectErr:  func(string, string) error { return wantReject },
	})

	if got := executeErr(errors.New("boom")); got != wantExecute {
		t.Fatalf("execute override mismatch: got %v, want %v", got, wantExecute)
	}
	if got := applyErr(errors.New("boom")); got != wantApply {
		t.Fatalf("apply override mismatch: got %v, want %v", got, wantApply)
	}
	if got := rejectErr("SOME_CODE", "boom"); got != wantReject {
		t.Fatalf("reject override mismatch: got %v, want %v", got, wantReject)
	}
}
