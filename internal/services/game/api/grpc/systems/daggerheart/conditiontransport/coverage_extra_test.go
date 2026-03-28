package conditiontransport

import (
	"context"
	"errors"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestConditionDependencyGuardsCoverMissingBranches(t *testing.T) {
	tests := []struct {
		name string
		deps Dependencies
		fn   func(*Handler) error
	}{
		{name: "character missing campaign", deps: Dependencies{}, fn: (*Handler).requireCharacterDependencies},
		{name: "character missing gate", deps: Dependencies{Campaign: testCampaignStore{}}, fn: (*Handler).requireCharacterDependencies},
		{name: "character missing daggerheart", deps: Dependencies{Campaign: testCampaignStore{}, SessionGate: testSessionGateStore{}}, fn: (*Handler).requireCharacterDependencies},
		{name: "character missing event", deps: Dependencies{Campaign: testCampaignStore{}, SessionGate: testSessionGateStore{}, Daggerheart: testDaggerheartStore{}}, fn: (*Handler).requireCharacterDependencies},
		{name: "character missing executor", deps: Dependencies{Campaign: testCampaignStore{}, SessionGate: testSessionGateStore{}, Daggerheart: testDaggerheartStore{}, Event: testEventStore{}}, fn: (*Handler).requireCharacterDependencies},
		{name: "adversary missing loader", deps: Dependencies{Campaign: testCampaignStore{}, SessionGate: testSessionGateStore{}, Daggerheart: testDaggerheartStore{}, Event: testEventStore{}, ExecuteDomainCommand: func(context.Context, DomainCommandInput) error { return nil }}, fn: (*Handler).requireAdversaryDependencies},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(tt.deps)
			if err := tt.fn(handler); status.Code(err) != codes.Internal {
				t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
			}
		})
	}
}

func TestConditionHelpersCoverClassesTriggersAndLifeStateBranches(t *testing.T) {
	if got, err := standardConditionCode(pb.DaggerheartCondition_DAGGERHEART_CONDITION_RESTRAINED); err != nil || got != rules.ConditionRestrained {
		t.Fatalf("standardConditionCode(restrained) = %q, %v", got, err)
	}
	if got, err := standardConditionCode(pb.DaggerheartCondition_DAGGERHEART_CONDITION_CLOAKED); err != nil || got != rules.ConditionCloaked {
		t.Fatalf("standardConditionCode(cloaked) = %q, %v", got, err)
	}

	if got := conditionClassToProto(string(rules.ConditionClassTag)); got != pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_TAG {
		t.Fatalf("conditionClassToProto(tag) = %v, want tag", got)
	}
	if got := conditionClassToProto(string(rules.ConditionClassSpecial)); got != pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_SPECIAL {
		t.Fatalf("conditionClassToProto(special) = %v, want special", got)
	}
	if got := conditionClassFromProto(pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_TAG); got != rules.ConditionClassTag {
		t.Fatalf("conditionClassFromProto(tag) = %q, want tag", got)
	}
	if got := conditionClassFromProto(pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_SPECIAL); got != rules.ConditionClassSpecial {
		t.Fatalf("conditionClassFromProto(special) = %q, want special", got)
	}

	clearTriggerTests := []struct {
		name  string
		value pb.DaggerheartConditionClearTrigger
		want  rules.ConditionClearTrigger
	}{
		{name: "short rest", value: pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SHORT_REST, want: rules.ConditionClearTriggerShortRest},
		{name: "long rest", value: pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_LONG_REST, want: rules.ConditionClearTriggerLongRest},
		{name: "session end", value: pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SESSION_END, want: rules.ConditionClearTriggerSessionEnd},
		{name: "damage taken", value: pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_DAMAGE_TAKEN, want: rules.ConditionClearTriggerDamageTaken},
	}
	for _, tt := range clearTriggerTests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := clearTriggerFromProto(tt.value)
			if err != nil {
				t.Fatalf("clearTriggerFromProto returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("trigger = %q, want %q", got, tt.want)
			}
		})
	}
	if _, err := clearTriggerFromProto(pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_UNSPECIFIED); err == nil {
		t.Fatal("expected error for unspecified clear trigger")
	}
	if got := domainClearTriggerToProto(rules.ConditionClearTriggerDamageTaken); got != pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_DAMAGE_TAKEN {
		t.Fatalf("domainClearTriggerToProto(damage_taken) = %v, want damage_taken", got)
	}
	if got := projectionClearTriggersToProto([]string{" short_rest ", "unknown"}); len(got) != 2 || got[0] != pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SHORT_REST || got[1] != pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_UNSPECIFIED {
		t.Fatalf("projectionClearTriggersToProto = %v", got)
	}
	if got := domainClearTriggersToProto([]rules.ConditionClearTrigger{rules.ConditionClearTriggerLongRest, rules.ConditionClearTriggerSessionEnd}); len(got) != 2 || got[0] != pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_LONG_REST || got[1] != pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SESSION_END {
		t.Fatalf("domainClearTriggersToProto = %v", got)
	}

	if got, err := lifeStateFromProto(pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE); err != nil || got == "" {
		t.Fatalf("lifeStateFromProto(alive) = %q, %v", got, err)
	}
	if got, err := lifeStateFromProto(pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD); err != nil || got == "" {
		t.Fatalf("lifeStateFromProto(dead) = %q, %v", got, err)
	}
	if _, err := lifeStateFromProto(pb.DaggerheartLifeState(99)); err == nil {
		t.Fatal("expected error for invalid life_state")
	}
}

func TestConditionStateViewsToDomainAndStateParsingCoverRemainingBranches(t *testing.T) {
	states, err := ConditionStateViewsToDomain([]*ConditionStateView{
		{
			ID:            " hidden ",
			Class:         pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_STANDARD,
			Standard:      pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN,
			ClearTriggers: []pb.DaggerheartConditionClearTrigger{pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SHORT_REST},
		},
		{
			ID:            "tag-1",
			Class:         pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_TAG,
			Code:          " marked ",
			Label:         " Marked ",
			Source:        " gm ",
			SourceID:      " scene-1 ",
			ClearTriggers: []pb.DaggerheartConditionClearTrigger{pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_DAMAGE_TAKEN},
		},
		nil,
	})
	if err != nil {
		t.Fatalf("ConditionStateViewsToDomain returned error: %v", err)
	}
	if len(states) != 2 || states[0].Code != rules.ConditionHidden || states[1].Code != "marked" {
		t.Fatalf("states = %#v", states)
	}
	if len(states[0].ClearTriggers) != 1 || states[0].ClearTriggers[0] != rules.ConditionClearTriggerShortRest {
		t.Fatalf("standard clear triggers = %#v", states[0].ClearTriggers)
	}
	if len(states[1].ClearTriggers) != 1 || states[1].ClearTriggers[0] != rules.ConditionClearTriggerDamageTaken {
		t.Fatalf("tag clear triggers = %#v", states[1].ClearTriggers)
	}

	if _, err := ConditionStateViewsToDomain([]*ConditionStateView{{
		ID:    "bad-1",
		Class: pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_STANDARD,
	}}); err == nil {
		t.Fatal("expected error for missing standard condition")
	}

	special, err := conditionStateFromProto(&pb.DaggerheartConditionState{
		Id:    "special-1",
		Class: pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_SPECIAL,
		Code:  "foggy",
	})
	if err != nil {
		t.Fatalf("conditionStateFromProto returned error: %v", err)
	}
	if special.Label != "foggy" {
		t.Fatalf("label = %q, want foggy", special.Label)
	}

	if _, err := conditionStateFromProto(&pb.DaggerheartConditionState{
		Id:    "bad-trigger",
		Class: pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_SPECIAL,
		Code:  "foggy",
		ClearTriggers: []pb.DaggerheartConditionClearTrigger{
			pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_UNSPECIFIED,
		},
	}); err == nil {
		t.Fatal("expected error for unspecified clear trigger")
	}
}

func TestConditionMutationHelpersCoverRemovalAndRollSeqBranches(t *testing.T) {
	got, err := normalizeConditionRemovalIDs([]string{" hidden ", "hidden", " marked "})
	if err != nil {
		t.Fatalf("normalizeConditionRemovalIDs returned error: %v", err)
	}
	if len(got) != 2 || got[0] != "hidden" || got[1] != "marked" {
		t.Fatalf("normalizeConditionRemovalIDs = %v, want hidden/marked", got)
	}
	if _, err := normalizeConditionRemovalIDs([]string{"hidden", " "}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}

	handler := newTestHandler(Dependencies{})
	if err := handler.validateRollSeq(testContextWithSessionID("sess-1"), "camp-1", "sess-1", nil); err != nil {
		t.Fatalf("validateRollSeq(nil) returned error: %v", err)
	}

	rollSeq := uint64(7)
	handler = newTestHandler(Dependencies{
		Event: testEventStore{err: errors.New("boom")},
	})
	if err := handler.validateRollSeq(testContextWithSessionID("sess-1"), "camp-1", "sess-1", &rollSeq); status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}

	handler = newTestHandler(Dependencies{
		Event: testEventStore{event: event.Event{SessionID: "other-session"}},
	})
	if err := handler.validateRollSeq(testContextWithSessionID("sess-1"), "camp-1", "sess-1", &rollSeq); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}
