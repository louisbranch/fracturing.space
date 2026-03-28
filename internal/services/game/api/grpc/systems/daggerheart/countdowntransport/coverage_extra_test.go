package countdowntransport

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCampaignCountdownHelpersApplyDefaultsAndStartingRolls(t *testing.T) {
	stored := projectionstore.DaggerheartCountdown{
		CampaignID:        "camp-1",
		CountdownID:       "cd-1",
		Name:              "Long Project",
		Tone:              rules.CountdownToneProgress,
		StartingValue:     6,
		RemainingValue:    4,
		LoopBehavior:      rules.CountdownLoopBehaviorReset,
		LinkedCountdownID: "linked-1",
		StartingRollMin:   1,
		StartingRollMax:   6,
		StartingRollValue: 4,
	}

	value := countdownFromStorage(stored)
	if value.AdvancementPolicy != rules.CountdownAdvancementPolicyManual {
		t.Fatalf("advancement policy = %q, want %q", value.AdvancementPolicy, rules.CountdownAdvancementPolicyManual)
	}
	if value.Status != rules.CountdownStatusActive {
		t.Fatalf("status = %q, want %q", value.Status, rules.CountdownStatusActive)
	}
	if value.StartingRoll == nil || value.StartingRoll.Value != 4 {
		t.Fatalf("starting roll = %#v, want value 4", value.StartingRoll)
	}

	proto := CampaignCountdownToProto(stored)
	if proto.GetAdvancementPolicy() != pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL {
		t.Fatalf("advancement policy = %v, want manual", proto.GetAdvancementPolicy())
	}
	if proto.GetStatus() != pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE {
		t.Fatalf("status = %v, want active", proto.GetStatus())
	}
	if proto.GetStartingRoll() == nil || proto.GetStartingRoll().GetValue() != 4 {
		t.Fatalf("starting roll = %#v, want value 4", proto.GetStartingRoll())
	}
}

func TestCountdownStatusAndPolicyConvertersCoverRequiredBranches(t *testing.T) {
	tests := []struct {
		name    string
		value   pb.DaggerheartCountdownStatus
		want    string
		wantErr codes.Code
	}{
		{name: "active", value: pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE, want: rules.CountdownStatusActive},
		{name: "trigger pending", value: pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_TRIGGER_PENDING, want: rules.CountdownStatusTriggerPending},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := countdownStatusFromProto(tt.value)
			if err != nil {
				t.Fatalf("countdownStatusFromProto returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("status = %q, want %q", got, tt.want)
			}
		})
	}

	if _, err := countdownStatusFromProto(pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_UNSPECIFIED); err == nil {
		t.Fatal("expected error for unspecified countdown status")
	}
	if _, err := countdownPolicyFromProto(pb.DaggerheartCountdownAdvancementPolicy(99)); err == nil {
		t.Fatal("expected error for invalid countdown advancement policy")
	}
	if got := countdownPolicyToProto(rules.CountdownAdvancementPolicyActionDynamic); got != pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_DYNAMIC {
		t.Fatalf("countdownPolicyToProto(action_dynamic) = %v, want action_dynamic", got)
	}
	if got := countdownStatusToProto(rules.CountdownStatusTriggerPending); got != pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_TRIGGER_PENDING {
		t.Fatalf("countdownStatusToProto(trigger_pending) = %v, want trigger_pending", got)
	}
}

func TestReadDependencyAndValidationHelpersCoverMissingBranches(t *testing.T) {
	if err := NewHandler(Dependencies{}).requireReadDependencies(); status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
	if err := NewHandler(Dependencies{Campaign: testCampaignStore{}}).requireReadDependencies(); status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}

	handler := newTestHandler(Dependencies{
		Campaign: testCampaignStore{err: errors.New("boom")},
	})
	if err := handler.validateCampaignRead(testContext(), "camp-1", "unsupported"); status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}

	handler = newTestHandler(Dependencies{
		Session: testSessionStore{err: errors.New("boom")},
	})
	if err := handler.validateCampaignSessionRead(testContext(), "camp-1", "sess-1", "unsupported"); status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}

	handler = newTestHandler(Dependencies{
		Session: testSessionStore{record: storage.SessionRecord{
			ID:         "sess-1",
			CampaignID: "camp-1",
			Status:     session.StatusEnded,
		}},
	})
	if err := handler.validateCampaignSessionRead(testContext(), "camp-1", "sess-1", "unsupported"); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}

	handler = newTestHandler(Dependencies{
		Campaign: testCampaignStore{record: storage.CampaignRecord{
			ID:     "camp-1",
			System: systembridge.SystemIDUnspecified,
			Status: campaign.StatusActive,
		}},
	})
	if err := handler.validateCampaignMutate(testContext(), "camp-1", "unsupported"); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func TestCreateCampaignCountdownCoversRandomizedStartAndValidation(t *testing.T) {
	handler := newTestHandler(Dependencies{
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})
	_, err := handler.CreateCampaignCountdown(testContext(), &pb.DaggerheartCreateCampaignCountdownRequest{
		CampaignId:        "camp-1",
		Name:              "Clock",
		Tone:              pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS,
		AdvancementPolicy: pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL,
		StartingValue: &pb.DaggerheartCreateCampaignCountdownRequest_RandomizedStart{
			RandomizedStart: &pb.DaggerheartCountdownRandomizedStart{Min: 0, Max: 2},
		},
		LoopBehavior: pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}

	store := &testDaggerheartStore{countdowns: map[string]projectionstore.DaggerheartCountdown{}}
	var commandInput DomainCommandInput
	handler = newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			commandInput = in
			var payload daggerheartpayload.CampaignCountdownCreatePayload
			if err := json.Unmarshal(in.PayloadJSON, &payload); err != nil {
				return err
			}
			store.countdowns["camp-1:generated-id"] = projectionstore.DaggerheartCountdown{
				CampaignID:        "camp-1",
				CountdownID:       "generated-id",
				Name:              payload.Name,
				Tone:              payload.Tone,
				AdvancementPolicy: payload.AdvancementPolicy,
				StartingValue:     payload.StartingValue,
				RemainingValue:    payload.RemainingValue,
				LoopBehavior:      payload.LoopBehavior,
				Status:            payload.Status,
				StartingRollMin:   payload.StartingRoll.Min,
				StartingRollMax:   payload.StartingRoll.Max,
				StartingRollValue: payload.StartingRoll.Value,
			}
			return nil
		},
	})
	seed := uint64(7)
	resp, err := handler.CreateCampaignCountdown(testContext(), &pb.DaggerheartCreateCampaignCountdownRequest{
		CampaignId:        "camp-1",
		Name:              "Clock",
		Tone:              pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS,
		AdvancementPolicy: pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL,
		StartingValue: &pb.DaggerheartCreateCampaignCountdownRequest_RandomizedStart{
			RandomizedStart: &pb.DaggerheartCountdownRandomizedStart{
				Min: 3,
				Max: 3,
				Rng: &commonv1.RngRequest{Seed: &seed},
			},
		},
		LoopBehavior: pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET,
	})
	if err != nil {
		t.Fatalf("CreateCampaignCountdown returned error: %v", err)
	}
	if commandInput.CommandType != commandids.DaggerheartCampaignCountdownCreate {
		t.Fatalf("command type = %q, want %q", commandInput.CommandType, commandids.DaggerheartCampaignCountdownCreate)
	}
	if resp.Countdown.CountdownID != "generated-id" || resp.Countdown.StartingRollValue != 3 {
		t.Fatalf("response = %#v, want generated countdown with starting roll value 3", resp.Countdown)
	}
}

func TestCampaignCountdownHandlersCoverReadDeleteAdvanceAndTrigger(t *testing.T) {
	store := &testDaggerheartStore{
		countdowns: map[string]projectionstore.DaggerheartCountdown{
			"camp-1:camp-1": {
				CampaignID:        "camp-1",
				CountdownID:       "camp-1",
				Name:              "Long Project",
				Tone:              rules.CountdownToneProgress,
				AdvancementPolicy: rules.CountdownAdvancementPolicyManual,
				StartingValue:     4,
				RemainingValue:    1,
				LoopBehavior:      rules.CountdownLoopBehaviorNone,
				Status:            rules.CountdownStatusActive,
			},
			"camp-1:scene-1": {
				CampaignID:  "camp-1",
				SessionID:   "sess-1",
				SceneID:     "scene-1",
				CountdownID: "scene-1",
			},
		},
	}
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			switch in.CommandType {
			case commandids.DaggerheartCampaignCountdownAdvance:
				var payload daggerheartpayload.CampaignCountdownAdvancedPayload
				if err := json.Unmarshal(in.PayloadJSON, &payload); err != nil {
					return err
				}
				store.countdowns["camp-1:camp-1"] = projectionstore.DaggerheartCountdown{
					CampaignID:        "camp-1",
					CountdownID:       "camp-1",
					Name:              "Long Project",
					Tone:              rules.CountdownToneProgress,
					AdvancementPolicy: rules.CountdownAdvancementPolicyManual,
					StartingValue:     4,
					RemainingValue:    payload.AfterRemaining,
					LoopBehavior:      rules.CountdownLoopBehaviorNone,
					Status:            payload.StatusAfter,
				}
			case commandids.DaggerheartCampaignCountdownTriggerResolve:
				var payload daggerheartpayload.CampaignCountdownTriggerResolvedPayload
				if err := json.Unmarshal(in.PayloadJSON, &payload); err != nil {
					return err
				}
				store.countdowns["camp-1:camp-1"] = projectionstore.DaggerheartCountdown{
					CampaignID:        "camp-1",
					CountdownID:       "camp-1",
					Name:              "Long Project",
					Tone:              rules.CountdownToneProgress,
					AdvancementPolicy: rules.CountdownAdvancementPolicyManual,
					StartingValue:     payload.StartingValueAfter,
					RemainingValue:    payload.RemainingValueAfter,
					LoopBehavior:      rules.CountdownLoopBehaviorNone,
					Status:            payload.StatusAfter,
				}
			case commandids.DaggerheartCampaignCountdownDelete:
				delete(store.countdowns, "camp-1:camp-1")
			default:
				return errors.New("unexpected command type")
			}
			return nil
		},
	})

	getResp, err := handler.GetCampaignCountdown(testContext(), &pb.DaggerheartGetCampaignCountdownRequest{
		CampaignId:  "camp-1",
		CountdownId: "camp-1",
	})
	if err != nil {
		t.Fatalf("GetCampaignCountdown returned error: %v", err)
	}
	if getResp.GetCountdown().GetCountdownId() != "camp-1" {
		t.Fatalf("countdown id = %q, want camp-1", getResp.GetCountdown().GetCountdownId())
	}

	if _, err := handler.GetCampaignCountdown(testContext(), &pb.DaggerheartGetCampaignCountdownRequest{
		CampaignId:  "camp-1",
		CountdownId: "scene-1",
	}); status.Code(err) != codes.NotFound {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.NotFound)
	}

	advanceResp, err := handler.AdvanceCampaignCountdown(testContext(), &pb.DaggerheartAdvanceCampaignCountdownRequest{
		CampaignId:  "camp-1",
		CountdownId: "camp-1",
		Amount:      1,
		Reason:      " mark progress ",
	})
	if err != nil {
		t.Fatalf("AdvanceCampaignCountdown returned error: %v", err)
	}
	if advanceResp.Summary.BeforeRemaining != 1 || advanceResp.Summary.AfterRemaining != 0 || !advanceResp.Summary.Triggered {
		t.Fatalf("advance summary = %#v, want before 1 after 0 with trigger", advanceResp.Summary)
	}

	triggerResp, err := handler.ResolveCampaignCountdownTrigger(testContext(), &pb.DaggerheartResolveCampaignCountdownTriggerRequest{
		CampaignId:  "camp-1",
		CountdownId: "camp-1",
		Reason:      " clear trigger ",
	})
	if err != nil {
		t.Fatalf("ResolveCampaignCountdownTrigger returned error: %v", err)
	}
	if triggerResp.Countdown.Status != rules.CountdownStatusActive {
		t.Fatalf("countdown status = %q, want %q", triggerResp.Countdown.Status, rules.CountdownStatusActive)
	}

	deleteResp, err := handler.DeleteCampaignCountdown(testContext(), &pb.DaggerheartDeleteCampaignCountdownRequest{
		CampaignId:  "camp-1",
		CountdownId: "camp-1",
		Reason:      " no longer needed ",
	})
	if err != nil {
		t.Fatalf("DeleteCampaignCountdown returned error: %v", err)
	}
	if deleteResp.CountdownID != "camp-1" {
		t.Fatalf("countdown id = %q, want camp-1", deleteResp.CountdownID)
	}

	if _, err := handler.DeleteCampaignCountdown(testContext(), &pb.DaggerheartDeleteCampaignCountdownRequest{
		CampaignId:  "camp-1",
		CountdownId: "scene-1",
	}); status.Code(err) != codes.NotFound {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.NotFound)
	}
}
