package generator

import (
	"context"
	"fmt"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
)

func TestCreateSessions_CountLessThanOne(t *testing.T) {
	g := newTestGen(1, testDeps(nil, nil, nil, nil, &fakeSessionManager{}, nil, nil))

	err := g.createSessions(context.Background(), "camp-1", 0, PresetConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateSessions_EndsAllButLast(t *testing.T) {
	var started, ended []string
	sessSeq := 0
	sess := &fakeSessionManager{
		startSession: func(_ context.Context, in *statev1.StartSessionRequest, _ ...grpc.CallOption) (*statev1.StartSessionResponse, error) {
			sessSeq++
			id := fmt.Sprintf("s-%d", sessSeq)
			started = append(started, id)
			return &statev1.StartSessionResponse{
				Session: &statev1.Session{Id: id, Name: in.Name},
			}, nil
		},
		endSession: func(_ context.Context, in *statev1.EndSessionRequest, _ ...grpc.CallOption) (*statev1.EndSessionResponse, error) {
			ended = append(ended, in.SessionId)
			return &statev1.EndSessionResponse{}, nil
		},
	}
	evt := &fakeEventAppender{
		appendEvent: func(context.Context, *statev1.AppendEventRequest, ...grpc.CallOption) (*statev1.AppendEventResponse, error) {
			return &statev1.AppendEventResponse{}, nil
		},
	}
	cfg := PresetConfig{EventsMin: 0, EventsMax: 0}
	g := newTestGen(1, testDeps(nil, nil, nil, nil, sess, evt, nil))

	err := g.createSessions(context.Background(), "camp-1", 3, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(started) != 3 {
		t.Fatalf("expected 3 sessions started, got %d", len(started))
	}
	// First 2 should be ended, last one stays active
	if len(ended) != 2 {
		t.Fatalf("expected 2 sessions ended, got %d: %v", len(ended), ended)
	}
	if ended[0] != "s-1" || ended[1] != "s-2" {
		t.Fatalf("expected s-1,s-2 ended, got %v", ended)
	}
}

func TestCreateSessions_IncludeEndedSessions(t *testing.T) {
	var ended []string
	sessSeq := 0
	sess := &fakeSessionManager{
		startSession: func(_ context.Context, in *statev1.StartSessionRequest, _ ...grpc.CallOption) (*statev1.StartSessionResponse, error) {
			sessSeq++
			return &statev1.StartSessionResponse{
				Session: &statev1.Session{Id: fmt.Sprintf("s-%d", sessSeq)},
			}, nil
		},
		endSession: func(_ context.Context, in *statev1.EndSessionRequest, _ ...grpc.CallOption) (*statev1.EndSessionResponse, error) {
			ended = append(ended, in.SessionId)
			return &statev1.EndSessionResponse{}, nil
		},
	}
	evt := &fakeEventAppender{
		appendEvent: func(context.Context, *statev1.AppendEventRequest, ...grpc.CallOption) (*statev1.AppendEventResponse, error) {
			return &statev1.AppendEventResponse{}, nil
		},
	}
	cfg := PresetConfig{EventsMin: 0, EventsMax: 0, IncludeEndedSessions: true}
	g := newTestGen(1, testDeps(nil, nil, nil, nil, sess, evt, nil))

	err := g.createSessions(context.Background(), "camp-1", 2, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Both sessions should be ended (IncludeEndedSessions + count>1)
	if len(ended) != 2 {
		t.Fatalf("expected 2 sessions ended, got %d: %v", len(ended), ended)
	}
}

func TestCreateSessions_DoesNotAppendEvents(t *testing.T) {
	var eventCount int
	sessSeq := 0
	sess := &fakeSessionManager{
		startSession: func(context.Context, *statev1.StartSessionRequest, ...grpc.CallOption) (*statev1.StartSessionResponse, error) {
			sessSeq++
			return &statev1.StartSessionResponse{
				Session: &statev1.Session{Id: fmt.Sprintf("s-%d", sessSeq)},
			}, nil
		},
		endSession: func(context.Context, *statev1.EndSessionRequest, ...grpc.CallOption) (*statev1.EndSessionResponse, error) {
			return &statev1.EndSessionResponse{}, nil
		},
	}
	evt := &fakeEventAppender{
		appendEvent: func(context.Context, *statev1.AppendEventRequest, ...grpc.CallOption) (*statev1.AppendEventResponse, error) {
			eventCount++
			return &statev1.AppendEventResponse{}, nil
		},
	}
	cfg := PresetConfig{EventsMin: 3, EventsMax: 3}
	g := newTestGen(1, testDeps(nil, nil, nil, nil, sess, evt, nil))

	err := g.createSessions(context.Background(), "camp-1", 1, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eventCount != 0 {
		t.Fatalf("expected no events appended, got %d", eventCount)
	}
}

func TestCreateSessions_StartError(t *testing.T) {
	sess := &fakeSessionManager{
		startSession: func(context.Context, *statev1.StartSessionRequest, ...grpc.CallOption) (*statev1.StartSessionResponse, error) {
			return nil, fmt.Errorf("start failed")
		},
	}
	g := newTestGen(1, testDeps(nil, nil, nil, nil, sess, nil, nil))

	err := g.createSessions(context.Background(), "camp-1", 1, PresetConfig{})
	if err == nil {
		t.Fatal("expected error from StartSession failure")
	}
}

func TestCreateSessions_EndError(t *testing.T) {
	sess := &fakeSessionManager{
		startSession: func(context.Context, *statev1.StartSessionRequest, ...grpc.CallOption) (*statev1.StartSessionResponse, error) {
			return &statev1.StartSessionResponse{
				Session: &statev1.Session{Id: "s-1"},
			}, nil
		},
		endSession: func(context.Context, *statev1.EndSessionRequest, ...grpc.CallOption) (*statev1.EndSessionResponse, error) {
			return nil, fmt.Errorf("end failed")
		},
	}
	evt := &fakeEventAppender{
		appendEvent: func(context.Context, *statev1.AppendEventRequest, ...grpc.CallOption) (*statev1.AppendEventResponse, error) {
			return &statev1.AppendEventResponse{}, nil
		},
	}
	cfg := PresetConfig{EventsMin: 0, EventsMax: 0}
	g := newTestGen(1, testDeps(nil, nil, nil, nil, sess, evt, nil))

	// 2 sessions — first should be ended, which triggers the error
	err := g.createSessions(context.Background(), "camp-1", 2, cfg)
	if err == nil {
		t.Fatal("expected error from EndSession failure")
	}
}

func TestAddSessionEvents_NoteEvent(t *testing.T) {
	var types []string
	evt := &fakeEventAppender{
		appendEvent: func(_ context.Context, in *statev1.AppendEventRequest, _ ...grpc.CallOption) (*statev1.AppendEventResponse, error) {
			types = append(types, in.Type)
			return &statev1.AppendEventResponse{}, nil
		},
	}
	g := newTestGen(1, testDeps(nil, nil, nil, nil, nil, evt, nil))

	err := g.addSessionEvents(context.Background(), "camp-1", "s-1", 2, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(types) != 2 {
		t.Fatalf("expected 2 events, got %d", len(types))
	}
	for _, tp := range types {
		if tp != "story.note_added" {
			t.Fatalf("expected note event type, got %q", tp)
		}
	}
}

func TestCreateRollEvent_NoCharactersFallsBackToNote(t *testing.T) {
	var appendCalled bool
	evt := &fakeEventAppender{
		appendEvent: func(_ context.Context, in *statev1.AppendEventRequest, _ ...grpc.CallOption) (*statev1.AppendEventResponse, error) {
			appendCalled = true
			if in.Type != "story.note_added" {
				t.Fatalf("expected note event fallback, got %q", in.Type)
			}
			return &statev1.AppendEventResponse{}, nil
		},
	}
	g := newTestGen(1, testDeps(nil, nil, nil, nil, nil, evt, nil))

	err := g.createRollEvent(context.Background(), "camp-1", "s-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !appendCalled {
		t.Fatal("expected event to be appended")
	}
}

func TestCreateNoteEvent_AppendError(t *testing.T) {
	evt := &fakeEventAppender{
		appendEvent: func(context.Context, *statev1.AppendEventRequest, ...grpc.CallOption) (*statev1.AppendEventResponse, error) {
			return nil, fmt.Errorf("append failed")
		},
	}
	g := newTestGen(1, testDeps(nil, nil, nil, nil, nil, evt, nil))

	err := g.createNoteEvent(context.Background(), "camp-1", "s-1")
	if err == nil {
		t.Fatal("expected error from AppendEvent failure")
	}
}
