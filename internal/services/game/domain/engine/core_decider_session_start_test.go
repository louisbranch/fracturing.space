package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/readiness"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

func TestCoreDeciderSessionStart_DraftCampaignEmitsActivationAndSessionStart(t *testing.T) {
	now := time.Date(2026, 2, 26, 12, 0, 0, 0, time.UTC)
	decision := CoreDecider{}.Decide(
		aggregate.State{
			Campaign: campaign.State{Created: true, Status: campaign.StatusDraft},
			Participants: map[string]participant.State{
				"gm-1": {
					ParticipantID: "gm-1",
					Joined:        true,
					Role:          string(participant.RoleGM),
				},
				"player-1": {
					ParticipantID: "player-1",
					Joined:        true,
					Role:          string(participant.RolePlayer),
				},
			},
			Characters: map[string]character.State{
				"char-1": {
					CharacterID:   "char-1",
					Created:       true,
					ParticipantID: "player-1",
				},
			},
		},
		command.Command{
			CampaignID:  "camp-1",
			Type:        session.CommandTypeStart,
			ActorType:   command.ActorTypeParticipant,
			ActorID:     "gm-1",
			PayloadJSON: []byte(`{"session_id":"sess-1","session_name":"First Session"}`),
		},
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

	var updatePayload campaign.UpdatePayload
	if err := json.Unmarshal(decision.Events[0].PayloadJSON, &updatePayload); err != nil {
		t.Fatalf("decode campaign.updated payload: %v", err)
	}
	if updatePayload.Fields["status"] != string(campaign.StatusActive) {
		t.Fatalf("campaign status field = %q, want %q", updatePayload.Fields["status"], campaign.StatusActive)
	}
}

func TestCoreDeciderSessionStart_ReadinessFailureRejected(t *testing.T) {
	decision := CoreDecider{}.Decide(
		aggregate.State{
			Campaign: campaign.State{Created: true, Status: campaign.StatusActive},
			Participants: map[string]participant.State{
				"gm-1": {
					ParticipantID: "gm-1",
					Joined:        true,
					Role:          string(participant.RoleGM),
				},
				"player-1": {
					ParticipantID: "player-1",
					Joined:        true,
					Role:          string(participant.RolePlayer),
				},
			},
			Characters: map[string]character.State{
				"char-1": {
					CharacterID: "char-1",
					Created:     true,
				},
			},
		},
		command.Command{
			CampaignID:  "camp-1",
			Type:        session.CommandTypeStart,
			ActorType:   command.ActorTypeParticipant,
			ActorID:     "gm-1",
			PayloadJSON: []byte(`{"session_id":"sess-1"}`),
		},
		nil,
	)

	if len(decision.Events) != 0 {
		t.Fatalf("events = %d, want 0", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != readiness.RejectionCodeSessionReadinessCharacterControllerRequired {
		t.Fatalf(
			"rejection code = %s, want %s",
			decision.Rejections[0].Code,
			readiness.RejectionCodeSessionReadinessCharacterControllerRequired,
		)
	}
}

func TestCoreDeciderSessionStart_UsesSystemCharacterReadinessChecker(t *testing.T) {
	systems := module.NewRegistry()
	if err := systems.Register(stubReadinessModule{ready: false, reason: "class is required"}); err != nil {
		t.Fatalf("register stub module: %v", err)
	}

	decision := CoreDecider{Systems: systems}.Decide(
		aggregate.State{
			Campaign: campaign.State{Created: true, Status: campaign.StatusActive, GameSystem: "stub"},
			Participants: map[string]participant.State{
				"gm-1": {
					ParticipantID: "gm-1",
					Joined:        true,
					Role:          string(participant.RoleGM),
				},
				"player-1": {
					ParticipantID: "player-1",
					Joined:        true,
					Role:          string(participant.RolePlayer),
				},
			},
			Characters: map[string]character.State{
				"char-1": {
					CharacterID:   "char-1",
					Created:       true,
					ParticipantID: "player-1",
					SystemProfile: map[string]any{"stub": map[string]any{"class": ""}},
				},
			},
		},
		command.Command{
			CampaignID:  "camp-1",
			Type:        session.CommandTypeStart,
			ActorType:   command.ActorTypeParticipant,
			ActorID:     "gm-1",
			PayloadJSON: []byte(`{"session_id":"sess-1"}`),
		},
		nil,
	)

	if len(decision.Events) != 0 {
		t.Fatalf("events = %d, want 0", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != readiness.RejectionCodeSessionReadinessCharacterSystemRequired {
		t.Fatalf(
			"rejection code = %s, want %s",
			decision.Rejections[0].Code,
			readiness.RejectionCodeSessionReadinessCharacterSystemRequired,
		)
	}
}

type stubReadinessModule struct {
	ready  bool
	reason string
}

func (m stubReadinessModule) ID() string                                   { return "stub" }
func (m stubReadinessModule) Version() string                              { return "1.0.0" }
func (m stubReadinessModule) RegisterCommands(*command.Registry) error     { return nil }
func (m stubReadinessModule) RegisterEvents(*event.Registry) error         { return nil }
func (m stubReadinessModule) EmittableEventTypes() []event.Type            { return nil }
func (m stubReadinessModule) Decider() module.Decider                      { return nil }
func (m stubReadinessModule) Folder() module.Folder                        { return nil }
func (m stubReadinessModule) StateFactory() module.StateFactory            { return nil }
func (m stubReadinessModule) CharacterReady(map[string]any) (bool, string) { return m.ready, m.reason }
