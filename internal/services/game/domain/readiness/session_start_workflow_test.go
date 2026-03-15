package readiness

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

func TestNewSessionStartWorkflow_BindsRegistry(t *testing.T) {
	systems := module.NewRegistry()

	workflow, ok := NewSessionStartWorkflow(systems).(sessionStartWorkflow)
	if !ok {
		t.Fatal("expected concrete sessionStartWorkflow")
	}
	if workflow.systems != systems {
		t.Fatal("expected workflow to keep the provided systems registry")
	}
}

func TestSessionStartWorkflowStart(t *testing.T) {
	now := time.Date(2026, 3, 10, 16, 30, 0, 0, time.UTC)
	startCommand := command.Command{
		CampaignID:  "camp-1",
		Type:        session.CommandTypeStart,
		ActorType:   command.ActorTypeParticipant,
		ActorID:     "gm-1",
		PayloadJSON: []byte(`{"session_id":"sess-1","session_name":"First Session"}`),
	}

	t.Run("readiness blocker wins before session decision", func(t *testing.T) {
		decision := sessionStartWorkflow{}.Start(
			aggregate.State{
				Campaign: campaign.State{Created: true, Status: campaign.StatusActive},
				Participants: map[ids.ParticipantID]participant.State{
					"gm-1": {
						ParticipantID: "gm-1",
						Joined:        true,
						Role:          participant.RoleGM,
					},
					"player-1": {
						ParticipantID: "player-1",
						Joined:        true,
						Role:          participant.RolePlayer,
					},
				},
				Characters: map[ids.CharacterID]character.State{
					"char-1": {CharacterID: "char-1", Created: true},
				},
			},
			startCommand,
			func() time.Time { return now },
		)

		if len(decision.Events) != 0 {
			t.Fatalf("events = %d, want 0", len(decision.Events))
		}
		if len(decision.Rejections) != 1 {
			t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
		}
		if decision.Rejections[0].Code != RejectionCodeSessionReadinessCharacterControllerRequired {
			t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, RejectionCodeSessionReadinessCharacterControllerRequired)
		}
	})

	t.Run("session decider rejection is preserved", func(t *testing.T) {
		cmd := startCommand
		cmd.PayloadJSON = []byte(`{"session_name":"Missing ID"}`)

		decision := sessionStartWorkflow{}.Start(
			readyWorkflowState(campaign.StatusActive),
			cmd,
			func() time.Time { return now },
		)

		if len(decision.Events) != 0 {
			t.Fatalf("events = %d, want 0", len(decision.Events))
		}
		if len(decision.Rejections) != 1 {
			t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
		}
		if decision.Rejections[0].Code != "SESSION_ID_REQUIRED" {
			t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, "SESSION_ID_REQUIRED")
		}
	})

	t.Run("active campaign only emits session started", func(t *testing.T) {
		decision := sessionStartWorkflow{}.Start(
			readyWorkflowState(campaign.StatusActive),
			startCommand,
			func() time.Time { return now },
		)

		if len(decision.Rejections) != 0 {
			t.Fatalf("rejections = %d, want 0", len(decision.Rejections))
		}
		if len(decision.Events) != 1 {
			t.Fatalf("events = %d, want 1", len(decision.Events))
		}
		if decision.Events[0].Type != session.EventTypeStarted {
			t.Fatalf("event type = %s, want %s", decision.Events[0].Type, session.EventTypeStarted)
		}
	})

	t.Run("draft campaign prepends activation event", func(t *testing.T) {
		decision := sessionStartWorkflow{}.Start(
			readyWorkflowState(campaign.StatusDraft),
			startCommand,
			func() time.Time { return now },
		)

		if len(decision.Rejections) != 0 {
			t.Fatalf("rejections = %d, want 0", len(decision.Rejections))
		}
		if len(decision.Events) != 2 {
			t.Fatalf("events = %d, want 2", len(decision.Events))
		}
		if decision.Events[0].Type != campaign.EventTypeUpdated {
			t.Fatalf("event 0 type = %s, want %s", decision.Events[0].Type, campaign.EventTypeUpdated)
		}
		if decision.Events[1].Type != session.EventTypeStarted {
			t.Fatalf("event 1 type = %s, want %s", decision.Events[1].Type, session.EventTypeStarted)
		}
		if !decision.Events[0].Timestamp.Equal(now) {
			t.Fatalf("campaign activation time = %s, want %s", decision.Events[0].Timestamp, now)
		}

		var updatePayload campaign.UpdatePayload
		if err := json.Unmarshal(decision.Events[0].PayloadJSON, &updatePayload); err != nil {
			t.Fatalf("decode campaign.updated payload: %v", err)
		}
		if updatePayload.Fields["status"] != string(campaign.StatusActive) {
			t.Fatalf("campaign status field = %q, want %q", updatePayload.Fields["status"], campaign.StatusActive)
		}
	})

	t.Run("default clock uses current time", func(t *testing.T) {
		decision := sessionStartWorkflow{}.Start(
			readyWorkflowState(campaign.StatusActive),
			startCommand,
			time.Now,
		)

		if len(decision.Rejections) != 0 {
			t.Fatalf("rejections = %d, want 0", len(decision.Rejections))
		}
		if len(decision.Events) != 1 {
			t.Fatalf("events = %d, want 1", len(decision.Events))
		}
		if decision.Events[0].Timestamp.IsZero() {
			t.Fatal("expected default clock to stamp the emitted event")
		}
	})
}

func TestSessionStartWorkflowSystemReadinessGuardRails(t *testing.T) {
	t.Run("nil registry", func(t *testing.T) {
		checker := sessionStartWorkflow{}.systemReadiness(aggregate.State{})
		if checker != nil {
			t.Fatal("expected nil checker when systems registry is missing")
		}
	})

	t.Run("blank system id", func(t *testing.T) {
		checker := sessionStartWorkflow{systems: module.NewRegistry()}.systemReadiness(aggregate.State{})
		if checker != nil {
			t.Fatal("expected nil checker when campaign system id is blank")
		}
	})

	t.Run("module without readiness checker", func(t *testing.T) {
		systems := module.NewRegistry()
		if err := systems.Register(stubModuleWithoutReadiness{}); err != nil {
			t.Fatalf("register module: %v", err)
		}
		checker := sessionStartWorkflow{systems: systems}.systemReadiness(aggregate.State{
			Campaign: workflowCampaignState("stub"),
		})
		if checker != nil {
			t.Fatal("expected nil checker when module does not implement readiness")
		}
	})

	t.Run("missing module", func(t *testing.T) {
		checker := sessionStartWorkflow{systems: module.NewRegistry()}.systemReadiness(aggregate.State{
			Campaign: workflowCampaignState("missing"),
		})
		if checker != nil {
			t.Fatal("expected nil checker when system registry has no matching module")
		}
	})

	t.Run("missing character", func(t *testing.T) {
		systems := module.NewRegistry()
		if err := systems.Register(stubReadinessModule{ready: true, reason: "ready"}); err != nil {
			t.Fatalf("register module: %v", err)
		}
		checker := sessionStartWorkflow{systems: systems}.systemReadiness(aggregate.State{
			Campaign: workflowCampaignState("stub"),
			Systems: map[module.Key]any{
				{ID: "stub", Version: "1.0.0"}: struct{}{},
			},
		})
		if checker == nil {
			t.Fatal("expected readiness checker")
		}
		ready, reason := checker("missing")
		if ready || reason != "character is missing" {
			t.Fatalf("checker result = (%t, %q), want (false, %q)", ready, reason, "character is missing")
		}
	})

	t.Run("delegates to module checker", func(t *testing.T) {
		systems := module.NewRegistry()
		if err := systems.Register(stubReadinessModule{ready: false, reason: "class is required"}); err != nil {
			t.Fatalf("register module: %v", err)
		}
		checker := sessionStartWorkflow{systems: systems}.systemReadiness(aggregate.State{
			Campaign: workflowCampaignState("stub"),
			Characters: map[ids.CharacterID]character.State{
				"char-1": {CharacterID: "char-1", Created: true},
			},
			Systems: map[module.Key]any{
				{ID: "stub", Version: "1.0.0"}: struct{}{},
			},
		})
		if checker == nil {
			t.Fatal("expected readiness checker")
		}
		ready, reason := checker("char-1")
		if ready || reason != "class is required" {
			t.Fatalf("checker result = (%t, %q), want (false, %q)", ready, reason, "class is required")
		}
	})
}

func workflowCampaignState(systemID string) campaign.State {
	return campaign.State{GameSystem: campaign.GameSystem(systemID)}
}

func readyWorkflowState(status campaign.Status) aggregate.State {
	return aggregate.State{
		Campaign: campaign.State{Created: true, Status: status},
		Participants: map[ids.ParticipantID]participant.State{
			"gm-1": {
				ParticipantID: "gm-1",
				Joined:        true,
				Role:          participant.RoleGM,
			},
			"player-1": {
				ParticipantID: "player-1",
				Joined:        true,
				Role:          participant.RolePlayer,
			},
		},
		Characters: map[ids.CharacterID]character.State{
			"char-1": {
				CharacterID:   "char-1",
				Created:       true,
				ParticipantID: "player-1",
			},
		},
	}
}

type stubModuleWithoutReadiness struct{}

func (stubModuleWithoutReadiness) ID() string                               { return "stub" }
func (stubModuleWithoutReadiness) Version() string                          { return "1.0.0" }
func (stubModuleWithoutReadiness) RegisterCommands(*command.Registry) error { return nil }
func (stubModuleWithoutReadiness) RegisterEvents(*event.Registry) error     { return nil }
func (stubModuleWithoutReadiness) EmittableEventTypes() []event.Type        { return nil }
func (stubModuleWithoutReadiness) Decider() module.Decider                  { return nil }
func (stubModuleWithoutReadiness) Folder() module.Folder                    { return nil }
func (stubModuleWithoutReadiness) StateFactory() module.StateFactory        { return nil }

type stubReadinessModule struct {
	ready  bool
	reason string
}

func (m stubReadinessModule) ID() string                               { return "stub" }
func (m stubReadinessModule) Version() string                          { return "1.0.0" }
func (m stubReadinessModule) RegisterCommands(*command.Registry) error { return nil }
func (m stubReadinessModule) RegisterEvents(*event.Registry) error     { return nil }
func (m stubReadinessModule) EmittableEventTypes() []event.Type        { return nil }
func (m stubReadinessModule) Decider() module.Decider                  { return nil }
func (m stubReadinessModule) Folder() module.Folder                    { return nil }
func (m stubReadinessModule) StateFactory() module.StateFactory        { return nil }
func (m stubReadinessModule) CharacterReady(any, character.State) (bool, string) {
	return m.ready, m.reason
}
