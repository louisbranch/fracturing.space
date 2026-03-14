package game

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRequestGMHandoffUsesResolvedParticipantIdentity(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	sessionStore := newFakeSessionStore()
	gateStore := newFakeSessionGateStore()
	sceneStore := newFakeCommunicationSceneStore()
	sceneCharacterStore := newFakeCommunicationSceneCharacterStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()
	now := time.Date(2026, 3, 9, 15, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = activeCampaignRecord("c1")
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	sessionStore.activeSession["c1"] = "s1"
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"part-1": {
			ID:             "part-1",
			CampaignID:     "c1",
			UserID:         "user-1",
			Name:           "Mira",
			Role:           participant.RolePlayer,
			CampaignAccess: participant.CampaignAccessMember,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.gate_opened"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "part-1",
			SessionID:   "s1",
			EntityType:  "session_gate",
			EntityID:    "gate-1",
			PayloadJSON: []byte(`{"gate_id":"gate-1","gate_type":"gm_handoff","reason":"party ready"}`),
		}),
	}}

	svc := &CommunicationService{
		stores: Stores{
			Campaign:       campaignStore,
			Character:      characterStore,
			Session:        sessionStore,
			SessionGate:    gateStore,
			Scene:          sceneStore,
			SceneCharacter: sceneCharacterStore,
			Participant:    participantStore,
			Event:          eventStore,
			Write:          domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		idGenerator: fixedSequenceIDGenerator("gate-1"),
	}

	resp, err := svc.RequestGMHandoff(contextWithUserID("user-1"), &campaignv1.RequestGMHandoffRequest{
		CampaignId: "c1",
		Reason:     "party ready",
	})
	if err != nil {
		t.Fatalf("RequestGMHandoff returned error: %v", err)
	}
	if resp.GetContext().GetActiveSessionGate().GetId() != "gate-1" {
		t.Fatalf("active gate id = %q, want %q", resp.GetContext().GetActiveSessionGate().GetId(), "gate-1")
	}
	if resp.GetContext().GetActiveSessionGate().GetType() != communicationGMHandoffGateType {
		t.Fatalf("active gate type = %q, want %q", resp.GetContext().GetActiveSessionGate().GetType(), communicationGMHandoffGateType)
	}
	if domain.lastCommand.ActorType != command.ActorTypeParticipant {
		t.Fatalf("command actor type = %q, want %q", domain.lastCommand.ActorType, command.ActorTypeParticipant)
	}
	if domain.lastCommand.ActorID != "part-1" {
		t.Fatalf("command actor id = %q, want %q", domain.lastCommand.ActorID, "part-1")
	}
}

func TestOpenCommunicationGateUsesManagerIdentity(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	sessionStore := newFakeSessionStore()
	gateStore := newFakeSessionGateStore()
	sceneStore := newFakeCommunicationSceneStore()
	sceneCharacterStore := newFakeCommunicationSceneCharacterStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2026, 3, 9, 15, 15, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = activeCampaignRecord("c1")
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	sessionStore.activeSession["c1"] = "s1"

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.gate_opened"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "manager-1",
			SessionID:   "s1",
			EntityType:  "session_gate",
			EntityID:    "gate-1",
			PayloadJSON: []byte(`{"gate_id":"gate-1","gate_type":"choice","reason":"pick a route"}`),
		}),
	}}

	svc := &CommunicationService{
		stores: Stores{
			Campaign:       campaignStore,
			Character:      characterStore,
			Session:        sessionStore,
			SessionGate:    gateStore,
			Scene:          sceneStore,
			SceneCharacter: sceneCharacterStore,
			Participant:    participantStore,
			Event:          eventStore,
			Write:          domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		idGenerator: fixedSequenceIDGenerator("gate-1"),
	}

	resp, err := svc.OpenCommunicationGate(contextWithParticipantID("manager-1"), &campaignv1.OpenCommunicationGateRequest{
		CampaignId: "c1",
		GateType:   "choice",
		Reason:     "pick a route",
	})
	if err != nil {
		t.Fatalf("OpenCommunicationGate returned error: %v", err)
	}
	if resp.GetContext().GetActiveSessionGate().GetId() != "gate-1" {
		t.Fatalf("active gate id = %q, want %q", resp.GetContext().GetActiveSessionGate().GetId(), "gate-1")
	}
	if resp.GetContext().GetActiveSessionGate().GetType() != "choice" {
		t.Fatalf("active gate type = %q, want %q", resp.GetContext().GetActiveSessionGate().GetType(), "choice")
	}
	if domain.lastCommand.ActorType != command.ActorTypeParticipant {
		t.Fatalf("command actor type = %q, want %q", domain.lastCommand.ActorType, command.ActorTypeParticipant)
	}
	if domain.lastCommand.ActorID != "manager-1" {
		t.Fatalf("command actor id = %q, want %q", domain.lastCommand.ActorID, "manager-1")
	}
}

func TestOpenCommunicationGateNormalizesReadyCheckWorkflowMetadata(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	sessionStore := newFakeSessionStore()
	gateStore := newFakeSessionGateStore()
	sceneStore := newFakeCommunicationSceneStore()
	sceneCharacterStore := newFakeCommunicationSceneCharacterStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2026, 3, 9, 15, 20, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = activeCampaignRecord("c1")
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	sessionStore.activeSession["c1"] = "s1"

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.gate_opened"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "manager-1",
			SessionID:   "s1",
			EntityType:  "session_gate",
			EntityID:    "gate-1",
			PayloadJSON: []byte(`{"gate_id":"gate-1","gate_type":"ready_check","reason":"confirm readiness"}`),
		}),
	}}

	svc := &CommunicationService{
		stores: Stores{
			Campaign:       campaignStore,
			Character:      characterStore,
			Session:        sessionStore,
			SessionGate:    gateStore,
			Scene:          sceneStore,
			SceneCharacter: sceneCharacterStore,
			Participant:    participantStore,
			Event:          eventStore,
			Write:          domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		idGenerator: fixedSequenceIDGenerator("gate-1"),
	}

	_, err := svc.OpenCommunicationGate(contextWithParticipantID("manager-1"), &campaignv1.OpenCommunicationGateRequest{
		CampaignId: "c1",
		GateType:   session.GateTypeReadyCheck,
		Reason:     "confirm readiness",
	})
	if err != nil {
		t.Fatalf("OpenCommunicationGate returned error: %v", err)
	}

	var payload session.GateOpenedPayload
	if err := json.Unmarshal(domain.lastCommand.PayloadJSON, &payload); err != nil {
		t.Fatalf("decode command payload: %v", err)
	}
	if got := payload.Metadata["response_authority"]; got != session.GateResponseAuthorityParticipant {
		t.Fatalf("response_authority = %v, want %q", got, session.GateResponseAuthorityParticipant)
	}
	options, ok := payload.Metadata["options"].([]any)
	if !ok {
		t.Fatalf("options type = %T, want []any", payload.Metadata["options"])
	}
	if len(options) != 2 || options[0] != "ready" || options[1] != "wait" {
		t.Fatalf("options = %#v", options)
	}
}

func TestOpenCommunicationGateRejectsVoteMetadataWithSingleOption(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	sessionStore := newFakeSessionStore()
	gateStore := newFakeSessionGateStore()
	sceneStore := newFakeCommunicationSceneStore()
	sceneCharacterStore := newFakeCommunicationSceneCharacterStore()
	participantStore := sessionManagerParticipantStore("c1")
	now := time.Date(2026, 3, 9, 15, 25, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = activeCampaignRecord("c1")
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	sessionStore.activeSession["c1"] = "s1"

	metadata, err := structpb.NewStruct(map[string]any{
		"options": []any{"north"},
	})
	if err != nil {
		t.Fatalf("build metadata struct: %v", err)
	}

	svc := &CommunicationService{
		stores: Stores{
			Campaign:       campaignStore,
			Character:      characterStore,
			Session:        sessionStore,
			SessionGate:    gateStore,
			Scene:          sceneStore,
			SceneCharacter: sceneCharacterStore,
			Participant:    participantStore,
		},
		idGenerator: fixedSequenceIDGenerator("gate-1"),
	}

	_, err = svc.OpenCommunicationGate(contextWithParticipantID("manager-1"), &campaignv1.OpenCommunicationGateRequest{
		CampaignId: "c1",
		GateType:   session.GateTypeVote,
		Reason:     "pick a route",
		Metadata:   metadata,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestRequestGMHandoffReturnsExistingGateWhenAlreadyOpen(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	sessionStore := newFakeSessionStore()
	gateStore := newFakeSessionGateStore()
	sceneStore := newFakeCommunicationSceneStore()
	sceneCharacterStore := newFakeCommunicationSceneCharacterStore()
	participantStore := newFakeParticipantStore()
	now := time.Date(2026, 3, 9, 15, 30, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = activeCampaignRecord("c1")
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	sessionStore.activeSession["c1"] = "s1"
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"part-1": {
			ID:             "part-1",
			CampaignID:     "c1",
			UserID:         "user-1",
			Name:           "Mira",
			Role:           participant.RolePlayer,
			CampaignAccess: participant.CampaignAccessMember,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}
	if err := gateStore.PutSessionGate(context.Background(), storage.SessionGate{
		CampaignID: "c1",
		SessionID:  "s1",
		GateID:     "gate-1",
		GateType:   communicationGMHandoffGateType,
		Status:     session.GateStatusOpen,
		CreatedAt:  now,
	}); err != nil {
		t.Fatalf("put session gate: %v", err)
	}

	svc := &CommunicationService{
		stores: Stores{
			Campaign:       campaignStore,
			Character:      characterStore,
			Session:        sessionStore,
			SessionGate:    gateStore,
			Scene:          sceneStore,
			SceneCharacter: sceneCharacterStore,
			Participant:    participantStore,
		},
		idGenerator: fixedSequenceIDGenerator("gate-2"),
	}

	resp, err := svc.RequestGMHandoff(contextWithUserID("user-1"), &campaignv1.RequestGMHandoffRequest{
		CampaignId: "c1",
	})
	if err != nil {
		t.Fatalf("RequestGMHandoff returned error: %v", err)
	}
	if resp.GetContext().GetActiveSessionGate().GetId() != "gate-1" {
		t.Fatalf("active gate id = %q, want %q", resp.GetContext().GetActiveSessionGate().GetId(), "gate-1")
	}
}

func TestResolveCommunicationGateUsesManagerAccessAndClearsActiveGate(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	sessionStore := newFakeSessionStore()
	gateStore := newFakeSessionGateStore()
	sceneStore := newFakeCommunicationSceneStore()
	sceneCharacterStore := newFakeCommunicationSceneCharacterStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2026, 3, 9, 15, 45, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = activeCampaignRecord("c1")
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	sessionStore.activeSession["c1"] = "s1"
	gateStore.gates["c1:s1:gate-1"] = storage.SessionGate{
		CampaignID: "c1",
		SessionID:  "s1",
		GateID:     "gate-1",
		GateType:   "choice",
		Status:     session.GateStatusOpen,
		CreatedAt:  now,
	}

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.gate_resolved"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "manager-1",
			SessionID:   "s1",
			EntityType:  "session_gate",
			EntityID:    "gate-1",
			PayloadJSON: []byte(`{"gate_id":"gate-1","decision":"left path"}`),
		}),
	}}

	svc := &CommunicationService{
		stores: Stores{
			Campaign:       campaignStore,
			Character:      characterStore,
			Session:        sessionStore,
			SessionGate:    gateStore,
			Scene:          sceneStore,
			SceneCharacter: sceneCharacterStore,
			Participant:    participantStore,
			Event:          eventStore,
			Write:          domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
	}

	resp, err := svc.ResolveCommunicationGate(contextWithParticipantID("manager-1"), &campaignv1.ResolveCommunicationGateRequest{
		CampaignId: "c1",
		Decision:   "left path",
	})
	if err != nil {
		t.Fatalf("ResolveCommunicationGate returned error: %v", err)
	}
	if resp.GetContext().GetActiveSessionGate() != nil {
		t.Fatalf("expected active session gate to be cleared, got %+v", resp.GetContext().GetActiveSessionGate())
	}
	if domain.lastCommand.ActorType != command.ActorTypeParticipant {
		t.Fatalf("command actor type = %q, want %q", domain.lastCommand.ActorType, command.ActorTypeParticipant)
	}
	if domain.lastCommand.ActorID != "manager-1" {
		t.Fatalf("command actor id = %q, want %q", domain.lastCommand.ActorID, "manager-1")
	}
}

func TestRespondToCommunicationGateUsesParticipantIdentityAndRecordsPayload(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	sessionStore := newFakeSessionStore()
	gateStore := newFakeSessionGateStore()
	sceneStore := newFakeCommunicationSceneStore()
	sceneCharacterStore := newFakeCommunicationSceneCharacterStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()
	now := time.Date(2026, 3, 9, 15, 50, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = activeCampaignRecord("c1")
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	sessionStore.activeSession["c1"] = "s1"
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"part-1": {
			ID:             "part-1",
			CampaignID:     "c1",
			UserID:         "user-1",
			Name:           "Mira",
			Role:           participant.RolePlayer,
			CampaignAccess: participant.CampaignAccessMember,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}
	gateStore.gates["c1:s1:gate-1"] = storage.SessionGate{
		CampaignID:   "c1",
		SessionID:    "s1",
		GateID:       "gate-1",
		GateType:     session.GateTypeReadyCheck,
		Status:       session.GateStatusOpen,
		MetadataJSON: []byte(`{"eligible_participant_ids":["part-1"]}`),
		ProgressJSON: []byte(`{"eligible_count":1,"responded_count":0,"pending_count":1,"all_responded":false}`),
		CreatedAt:    now,
	}

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.gate_response_recorded"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "part-1",
			SessionID:   "s1",
			EntityType:  "session_gate",
			EntityID:    "gate-1",
			PayloadJSON: []byte(`{"gate_id":"gate-1","participant_id":"part-1","decision":"ready","response":{"note":"ready to proceed"}}`),
		}),
	}}

	svc := &CommunicationService{
		stores: Stores{
			Campaign:       campaignStore,
			Character:      characterStore,
			Session:        sessionStore,
			SessionGate:    gateStore,
			Scene:          sceneStore,
			SceneCharacter: sceneCharacterStore,
			Participant:    participantStore,
			Event:          eventStore,
			Write:          domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
	}

	response, err := structpb.NewStruct(map[string]any{"note": "ready to proceed"})
	if err != nil {
		t.Fatalf("build response struct: %v", err)
	}

	resp, err := svc.RespondToCommunicationGate(contextWithUserID("user-1"), &campaignv1.RespondToCommunicationGateRequest{
		CampaignId: "c1",
		Decision:   "ready",
		Response:   response,
	})
	if err != nil {
		t.Fatalf("RespondToCommunicationGate returned error: %v", err)
	}
	if resp.GetContext().GetActiveSessionGate().GetId() != "gate-1" {
		t.Fatalf("active gate id = %q, want %q", resp.GetContext().GetActiveSessionGate().GetId(), "gate-1")
	}
	if domain.lastCommand.Type != commandTypeSessionGateRespond {
		t.Fatalf("command type = %q, want %q", domain.lastCommand.Type, commandTypeSessionGateRespond)
	}
	if domain.lastCommand.ActorType != command.ActorTypeParticipant {
		t.Fatalf("command actor type = %q, want %q", domain.lastCommand.ActorType, command.ActorTypeParticipant)
	}
	if domain.lastCommand.ActorID != "part-1" {
		t.Fatalf("command actor id = %q, want %q", domain.lastCommand.ActorID, "part-1")
	}

	var payload session.GateResponseRecordedPayload
	if err := json.Unmarshal(domain.lastCommand.PayloadJSON, &payload); err != nil {
		t.Fatalf("decode command payload: %v", err)
	}
	if payload.GateID != "gate-1" || payload.ParticipantID != "part-1" || payload.Decision != "ready" {
		t.Fatalf("unexpected command payload: %+v", payload)
	}
	if got := payload.Response["note"]; got != "ready to proceed" {
		t.Fatalf("response note = %v, want %q", got, "ready to proceed")
	}
}

func TestResolveGMHandoffUsesManagerAccessAndClearsActiveGate(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	characterStore := newFakeCharacterStore()
	sessionStore := newFakeSessionStore()
	gateStore := newFakeSessionGateStore()
	sceneStore := newFakeCommunicationSceneStore()
	sceneCharacterStore := newFakeCommunicationSceneCharacterStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2026, 3, 9, 16, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = activeCampaignRecord("c1")
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	sessionStore.activeSession["c1"] = "s1"
	gateStore.gates["c1:s1:gate-1"] = storage.SessionGate{
		CampaignID: "c1",
		SessionID:  "s1",
		GateID:     "gate-1",
		GateType:   communicationGMHandoffGateType,
		Status:     session.GateStatusOpen,
		CreatedAt:  now,
	}

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.gate_resolved"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "manager-1",
			SessionID:   "s1",
			EntityType:  "session_gate",
			EntityID:    "gate-1",
			PayloadJSON: []byte(`{"gate_id":"gate-1","decision":"proceed"}`),
		}),
	}}

	svc := &CommunicationService{
		stores: Stores{
			Campaign:       campaignStore,
			Character:      characterStore,
			Session:        sessionStore,
			SessionGate:    gateStore,
			Scene:          sceneStore,
			SceneCharacter: sceneCharacterStore,
			Participant:    participantStore,
			Event:          eventStore,
			Write:          domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		idGenerator: fixedSequenceIDGenerator("gate-1"),
	}

	resp, err := svc.ResolveGMHandoff(contextWithParticipantID("manager-1"), &campaignv1.ResolveGMHandoffRequest{
		CampaignId: "c1",
		Decision:   "proceed",
	})
	if err != nil {
		t.Fatalf("ResolveGMHandoff returned error: %v", err)
	}
	if resp.GetContext().GetActiveSessionGate() != nil {
		t.Fatalf("expected active session gate to be cleared, got %+v", resp.GetContext().GetActiveSessionGate())
	}
	if domain.lastCommand.ActorType != command.ActorTypeParticipant {
		t.Fatalf("command actor type = %q, want %q", domain.lastCommand.ActorType, command.ActorTypeParticipant)
	}
	if domain.lastCommand.ActorID != "manager-1" {
		t.Fatalf("command actor id = %q, want %q", domain.lastCommand.ActorID, "manager-1")
	}
}

func TestAbandonGMHandoffRejectsDifferentOpenGateType(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	gateStore := newFakeSessionGateStore()
	participantStore := sessionManagerParticipantStore("c1")
	now := time.Date(2026, 3, 9, 16, 30, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = activeCampaignRecord("c1")
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	sessionStore.activeSession["c1"] = "s1"
	gateStore.gates["c1:s1:gate-1"] = storage.SessionGate{
		CampaignID: "c1",
		SessionID:  "s1",
		GateID:     "gate-1",
		GateType:   "choice",
		Status:     session.GateStatusOpen,
		CreatedAt:  now,
	}

	svc := &CommunicationService{
		stores: Stores{
			Campaign:    campaignStore,
			Session:     sessionStore,
			SessionGate: gateStore,
			Participant: participantStore,
		},
	}

	_, err := svc.AbandonGMHandoff(contextWithParticipantID("manager-1"), &campaignv1.AbandonGMHandoffRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}
