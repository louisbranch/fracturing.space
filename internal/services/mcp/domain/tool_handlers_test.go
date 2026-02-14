package domain

import (
	"context"
	"fmt"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestCampaignCreateHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &fakeCampaignClient{
			createResp: &statev1.CreateCampaignResponse{
				Campaign:         testCampaign("c1", "Adventure", statev1.CampaignStatus_DRAFT),
				OwnerParticipant: testParticipant("p1", "c1", "GM", statev1.ParticipantRole_GM),
			},
		}
		handler := CampaignCreateHandler(client, nil)
		toolResult, result, err := handler(context.Background(), nil, CampaignCreateInput{
			Name:   "Adventure",
			System: "DAGGERHEART",
			GmMode: "HUMAN",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if toolResult == nil {
			t.Fatal("expected non-nil tool result")
		}
		if result.ID != "c1" {
			t.Errorf("expected id %q, got %q", "c1", result.ID)
		}
		if result.OwnerParticipantID != "p1" {
			t.Errorf("expected owner_participant_id %q, got %q", "p1", result.OwnerParticipantID)
		}
		if result.Status != "DRAFT" {
			t.Errorf("expected status DRAFT, got %q", result.Status)
		}
	})

	t.Run("gRPC error", func(t *testing.T) {
		client := &fakeCampaignClient{
			createErr: fmt.Errorf("connection refused"),
		}
		handler := CampaignCreateHandler(client, nil)
		_, _, err := handler(context.Background(), nil, CampaignCreateInput{Name: "X"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nil response", func(t *testing.T) {
		client := &fakeCampaignClient{
			createResp: &statev1.CreateCampaignResponse{},
		}
		handler := CampaignCreateHandler(client, nil)
		_, _, err := handler(context.Background(), nil, CampaignCreateInput{Name: "X"})
		if err == nil {
			t.Fatal("expected error for nil campaign in response")
		}
	})

	t.Run("missing owner participant", func(t *testing.T) {
		client := &fakeCampaignClient{
			createResp: &statev1.CreateCampaignResponse{
				Campaign: testCampaign("c1", "X", statev1.CampaignStatus_DRAFT),
			},
		}
		handler := CampaignCreateHandler(client, nil)
		_, _, err := handler(context.Background(), nil, CampaignCreateInput{Name: "X"})
		if err == nil {
			t.Fatal("expected error for missing owner participant")
		}
	})

	t.Run("notifies resource updates", func(t *testing.T) {
		client := &fakeCampaignClient{
			createResp: &statev1.CreateCampaignResponse{
				Campaign:         testCampaign("c1", "X", statev1.CampaignStatus_DRAFT),
				OwnerParticipant: testParticipant("p1", "c1", "GM", statev1.ParticipantRole_GM),
			},
		}
		var notified []string
		notify := func(_ context.Context, uri string) { notified = append(notified, uri) }
		handler := CampaignCreateHandler(client, notify)
		_, _, err := handler(context.Background(), nil, CampaignCreateInput{Name: "X"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(notified) != 2 {
			t.Errorf("expected 2 notifications, got %d", len(notified))
		}
	})
}

func TestCampaignEndHandler(t *testing.T) {
	t.Run("success with input campaign_id", func(t *testing.T) {
		client := &fakeCampaignClient{
			endResp: &statev1.EndCampaignResponse{
				Campaign: testCampaign("c1", "X", statev1.CampaignStatus_COMPLETED),
			},
		}
		handler := CampaignEndHandler(client, nil, nil)
		_, result, err := handler(context.Background(), nil, CampaignStatusChangeInput{CampaignID: "c1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != "COMPLETED" {
			t.Errorf("expected status COMPLETED, got %q", result.Status)
		}
	})

	t.Run("falls back to context campaign_id", func(t *testing.T) {
		client := &fakeCampaignClient{
			endResp: &statev1.EndCampaignResponse{
				Campaign: testCampaign("c1", "X", statev1.CampaignStatus_COMPLETED),
			},
		}
		getCtx := func() Context { return Context{CampaignID: "c1"} }
		handler := CampaignEndHandler(client, getCtx, nil)
		_, _, err := handler(context.Background(), nil, CampaignStatusChangeInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("missing campaign_id", func(t *testing.T) {
		handler := CampaignEndHandler(&fakeCampaignClient{}, nil, nil)
		_, _, err := handler(context.Background(), nil, CampaignStatusChangeInput{})
		if err == nil {
			t.Fatal("expected error for missing campaign_id")
		}
	})
}

func TestCampaignArchiveHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &fakeCampaignClient{
			archiveResp: &statev1.ArchiveCampaignResponse{
				Campaign: testCampaign("c1", "X", statev1.CampaignStatus_ARCHIVED),
			},
		}
		handler := CampaignArchiveHandler(client, nil, nil)
		_, result, err := handler(context.Background(), nil, CampaignStatusChangeInput{CampaignID: "c1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != "ARCHIVED" {
			t.Errorf("expected status ARCHIVED, got %q", result.Status)
		}
	})

	t.Run("missing campaign_id", func(t *testing.T) {
		handler := CampaignArchiveHandler(&fakeCampaignClient{}, nil, nil)
		_, _, err := handler(context.Background(), nil, CampaignStatusChangeInput{})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestCampaignRestoreHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &fakeCampaignClient{
			restoreResp: &statev1.RestoreCampaignResponse{
				Campaign: testCampaign("c1", "X", statev1.CampaignStatus_DRAFT),
			},
		}
		handler := CampaignRestoreHandler(client, nil, nil)
		_, result, err := handler(context.Background(), nil, CampaignStatusChangeInput{CampaignID: "c1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != "DRAFT" {
			t.Errorf("expected status DRAFT, got %q", result.Status)
		}
	})

	t.Run("missing campaign_id", func(t *testing.T) {
		handler := CampaignRestoreHandler(&fakeCampaignClient{}, nil, nil)
		_, _, err := handler(context.Background(), nil, CampaignStatusChangeInput{})
		if err == nil {
			t.Fatal("expected error for missing campaign_id")
		}
	})

	t.Run("gRPC error", func(t *testing.T) {
		client := &fakeCampaignClient{
			restoreErr: fmt.Errorf("connection refused"),
		}
		handler := CampaignRestoreHandler(client, nil, nil)
		_, _, err := handler(context.Background(), nil, CampaignStatusChangeInput{CampaignID: "c1"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nil response", func(t *testing.T) {
		client := &fakeCampaignClient{
			restoreResp: &statev1.RestoreCampaignResponse{},
		}
		handler := CampaignRestoreHandler(client, nil, nil)
		_, _, err := handler(context.Background(), nil, CampaignStatusChangeInput{CampaignID: "c1"})
		if err == nil {
			t.Fatal("expected error for nil campaign in response")
		}
	})

	t.Run("falls back to context campaign_id", func(t *testing.T) {
		client := &fakeCampaignClient{
			restoreResp: &statev1.RestoreCampaignResponse{
				Campaign: testCampaign("c1", "X", statev1.CampaignStatus_DRAFT),
			},
		}
		getCtx := func() Context { return Context{CampaignID: "c1"} }
		handler := CampaignRestoreHandler(client, getCtx, nil)
		_, _, err := handler(context.Background(), nil, CampaignStatusChangeInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestSessionStartHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &fakeSessionClient{
			startResp: &statev1.StartSessionResponse{
				Session: testSession("s1", "c1", "Session 1", statev1.SessionStatus_SESSION_ACTIVE),
			},
		}
		handler := SessionStartHandler(client, nil)
		_, result, err := handler(context.Background(), nil, SessionStartInput{CampaignID: "c1", Name: "Session 1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != "s1" {
			t.Errorf("expected id %q, got %q", "s1", result.ID)
		}
		if result.Status != "ACTIVE" {
			t.Errorf("expected status ACTIVE, got %q", result.Status)
		}
	})

	t.Run("gRPC error", func(t *testing.T) {
		client := &fakeSessionClient{
			startErr: fmt.Errorf("error"),
		}
		handler := SessionStartHandler(client, nil)
		_, _, err := handler(context.Background(), nil, SessionStartInput{CampaignID: "c1"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nil response", func(t *testing.T) {
		client := &fakeSessionClient{
			startResp: &statev1.StartSessionResponse{},
		}
		handler := SessionStartHandler(client, nil)
		_, _, err := handler(context.Background(), nil, SessionStartInput{CampaignID: "c1"})
		if err == nil {
			t.Fatal("expected error for nil session in response")
		}
	})
}

func TestSessionEndHandler(t *testing.T) {
	t.Run("success with explicit IDs", func(t *testing.T) {
		client := &fakeSessionClient{
			endResp: &statev1.EndSessionResponse{
				Session: testSession("s1", "c1", "Session 1", statev1.SessionStatus_SESSION_ENDED),
			},
		}
		handler := SessionEndHandler(client, nil, nil)
		_, result, err := handler(context.Background(), nil, SessionEndInput{CampaignID: "c1", SessionID: "s1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != "ENDED" {
			t.Errorf("expected status ENDED, got %q", result.Status)
		}
	})

	t.Run("falls back to context", func(t *testing.T) {
		client := &fakeSessionClient{
			endResp: &statev1.EndSessionResponse{
				Session: testSession("s1", "c1", "Session 1", statev1.SessionStatus_SESSION_ENDED),
			},
		}
		getCtx := func() Context { return Context{CampaignID: "c1", SessionID: "s1"} }
		handler := SessionEndHandler(client, getCtx, nil)
		_, _, err := handler(context.Background(), nil, SessionEndInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("missing campaign_id", func(t *testing.T) {
		handler := SessionEndHandler(&fakeSessionClient{}, nil, nil)
		_, _, err := handler(context.Background(), nil, SessionEndInput{SessionID: "s1"})
		if err == nil {
			t.Fatal("expected error for missing campaign_id")
		}
	})

	t.Run("missing session_id", func(t *testing.T) {
		handler := SessionEndHandler(&fakeSessionClient{}, nil, nil)
		_, _, err := handler(context.Background(), nil, SessionEndInput{CampaignID: "c1"})
		if err == nil {
			t.Fatal("expected error for missing session_id")
		}
	})
}

func TestParticipantCreateHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &fakeParticipantClient{
			createResp: &statev1.CreateParticipantResponse{
				Participant: testParticipant("p1", "c1", "Alice", statev1.ParticipantRole_PLAYER),
			},
		}
		handler := ParticipantCreateHandler(client, nil, nil)
		_, result, err := handler(context.Background(), nil, ParticipantCreateInput{
			CampaignID:  "c1",
			DisplayName: "Alice",
			Role:        "PLAYER",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.DisplayName != "Alice" {
			t.Errorf("expected display name %q, got %q", "Alice", result.DisplayName)
		}
	})

	t.Run("gRPC error", func(t *testing.T) {
		client := &fakeParticipantClient{
			createErr: fmt.Errorf("connection refused"),
		}
		handler := ParticipantCreateHandler(client, nil, nil)
		_, _, err := handler(context.Background(), nil, ParticipantCreateInput{CampaignID: "c1", DisplayName: "X", Role: "GM"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nil response", func(t *testing.T) {
		client := &fakeParticipantClient{
			createResp: &statev1.CreateParticipantResponse{},
		}
		handler := ParticipantCreateHandler(client, nil, nil)
		_, _, err := handler(context.Background(), nil, ParticipantCreateInput{CampaignID: "c1", DisplayName: "X", Role: "GM"})
		if err == nil {
			t.Fatal("expected error for nil participant in response")
		}
	})

	t.Run("with controller", func(t *testing.T) {
		client := &fakeParticipantClient{
			createResp: &statev1.CreateParticipantResponse{
				Participant: testParticipant("p1", "c1", "Bot", statev1.ParticipantRole_PLAYER),
			},
		}
		handler := ParticipantCreateHandler(client, nil, nil)
		_, _, err := handler(context.Background(), nil, ParticipantCreateInput{
			CampaignID: "c1", DisplayName: "Bot", Role: "PLAYER", Controller: "AI",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestParticipantUpdateHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		name := "Bob"
		client := &fakeParticipantClient{
			updateResp: &statev1.UpdateParticipantResponse{
				Participant: testParticipant("p1", "c1", "Bob", statev1.ParticipantRole_PLAYER),
			},
		}
		handler := ParticipantUpdateHandler(client, nil, nil)
		_, result, err := handler(context.Background(), nil, ParticipantUpdateInput{
			CampaignID:    "c1",
			ParticipantID: "p1",
			DisplayName:   &name,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.DisplayName != "Bob" {
			t.Errorf("expected %q, got %q", "Bob", result.DisplayName)
		}
	})

	t.Run("missing campaign_id", func(t *testing.T) {
		name := "X"
		handler := ParticipantUpdateHandler(&fakeParticipantClient{}, nil, nil)
		_, _, err := handler(context.Background(), nil, ParticipantUpdateInput{ParticipantID: "p1", DisplayName: &name})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("missing participant_id", func(t *testing.T) {
		name := "X"
		handler := ParticipantUpdateHandler(&fakeParticipantClient{}, nil, nil)
		_, _, err := handler(context.Background(), nil, ParticipantUpdateInput{CampaignID: "c1", DisplayName: &name})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("no fields provided", func(t *testing.T) {
		handler := ParticipantUpdateHandler(&fakeParticipantClient{}, nil, nil)
		_, _, err := handler(context.Background(), nil, ParticipantUpdateInput{CampaignID: "c1", ParticipantID: "p1"})
		if err == nil {
			t.Fatal("expected error for no fields")
		}
	})

	t.Run("invalid role", func(t *testing.T) {
		role := "INVALID"
		handler := ParticipantUpdateHandler(&fakeParticipantClient{}, nil, nil)
		_, _, err := handler(context.Background(), nil, ParticipantUpdateInput{
			CampaignID:    "c1",
			ParticipantID: "p1",
			Role:          &role,
		})
		if err == nil {
			t.Fatal("expected error for invalid role")
		}
	})

	t.Run("invalid controller", func(t *testing.T) {
		ctrl := "INVALID"
		handler := ParticipantUpdateHandler(&fakeParticipantClient{}, nil, nil)
		_, _, err := handler(context.Background(), nil, ParticipantUpdateInput{
			CampaignID:    "c1",
			ParticipantID: "p1",
			Controller:    &ctrl,
		})
		if err == nil {
			t.Fatal("expected error for invalid controller")
		}
	})
}

func TestParticipantDeleteHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &fakeParticipantClient{
			deleteResp: &statev1.DeleteParticipantResponse{
				Participant: testParticipant("p1", "c1", "Alice", statev1.ParticipantRole_PLAYER),
			},
		}
		handler := ParticipantDeleteHandler(client, nil, nil)
		_, result, err := handler(context.Background(), nil, ParticipantDeleteInput{
			CampaignID:    "c1",
			ParticipantID: "p1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != "p1" {
			t.Errorf("expected id %q, got %q", "p1", result.ID)
		}
	})

	t.Run("missing campaign_id", func(t *testing.T) {
		handler := ParticipantDeleteHandler(&fakeParticipantClient{}, nil, nil)
		_, _, err := handler(context.Background(), nil, ParticipantDeleteInput{ParticipantID: "p1"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("missing participant_id", func(t *testing.T) {
		handler := ParticipantDeleteHandler(&fakeParticipantClient{}, nil, nil)
		_, _, err := handler(context.Background(), nil, ParticipantDeleteInput{CampaignID: "c1"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestCharacterCreateHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &fakeCharacterClient{
			createResp: &statev1.CreateCharacterResponse{
				Character: testCharacter("ch1", "c1", "Hero", statev1.CharacterKind_PC),
			},
		}
		handler := CharacterCreateHandler(client, nil)
		_, result, err := handler(context.Background(), nil, CharacterCreateInput{
			CampaignID: "c1", Name: "Hero", Kind: "PC",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Name != "Hero" || result.Kind != "PC" {
			t.Errorf("unexpected result: %+v", result)
		}
	})

	t.Run("gRPC error", func(t *testing.T) {
		client := &fakeCharacterClient{createErr: fmt.Errorf("error")}
		handler := CharacterCreateHandler(client, nil)
		_, _, err := handler(context.Background(), nil, CharacterCreateInput{CampaignID: "c1", Name: "X", Kind: "PC"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nil response", func(t *testing.T) {
		client := &fakeCharacterClient{createResp: &statev1.CreateCharacterResponse{}}
		handler := CharacterCreateHandler(client, nil)
		_, _, err := handler(context.Background(), nil, CharacterCreateInput{CampaignID: "c1", Name: "X", Kind: "PC"})
		if err == nil {
			t.Fatal("expected error for nil character in response")
		}
	})
}

func TestCharacterUpdateHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		name := "Updated"
		client := &fakeCharacterClient{
			updateResp: &statev1.UpdateCharacterResponse{
				Character: testCharacter("ch1", "c1", "Updated", statev1.CharacterKind_PC),
			},
		}
		handler := CharacterUpdateHandler(client, nil)
		_, result, err := handler(context.Background(), nil, CharacterUpdateInput{
			CampaignID: "c1", CharacterID: "ch1", Name: &name,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Name != "Updated" {
			t.Errorf("expected name %q, got %q", "Updated", result.Name)
		}
	})

	t.Run("missing campaign_id", func(t *testing.T) {
		name := "X"
		handler := CharacterUpdateHandler(&fakeCharacterClient{}, nil)
		_, _, err := handler(context.Background(), nil, CharacterUpdateInput{CharacterID: "ch1", Name: &name})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("missing character_id", func(t *testing.T) {
		name := "X"
		handler := CharacterUpdateHandler(&fakeCharacterClient{}, nil)
		_, _, err := handler(context.Background(), nil, CharacterUpdateInput{CampaignID: "c1", Name: &name})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("no fields provided", func(t *testing.T) {
		handler := CharacterUpdateHandler(&fakeCharacterClient{}, nil)
		_, _, err := handler(context.Background(), nil, CharacterUpdateInput{CampaignID: "c1", CharacterID: "ch1"})
		if err == nil {
			t.Fatal("expected error for no fields")
		}
	})

	t.Run("invalid kind", func(t *testing.T) {
		kind := "INVALID"
		handler := CharacterUpdateHandler(&fakeCharacterClient{}, nil)
		_, _, err := handler(context.Background(), nil, CharacterUpdateInput{
			CampaignID: "c1", CharacterID: "ch1", Kind: &kind,
		})
		if err == nil {
			t.Fatal("expected error for invalid kind")
		}
	})
}

func TestCharacterDeleteHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &fakeCharacterClient{
			deleteResp: &statev1.DeleteCharacterResponse{
				Character: testCharacter("ch1", "c1", "Hero", statev1.CharacterKind_PC),
			},
		}
		handler := CharacterDeleteHandler(client, nil)
		_, result, err := handler(context.Background(), nil, CharacterDeleteInput{
			CampaignID: "c1", CharacterID: "ch1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != "ch1" {
			t.Errorf("expected id %q, got %q", "ch1", result.ID)
		}
	})

	t.Run("missing campaign_id", func(t *testing.T) {
		handler := CharacterDeleteHandler(&fakeCharacterClient{}, nil)
		_, _, err := handler(context.Background(), nil, CharacterDeleteInput{CharacterID: "ch1"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("missing character_id", func(t *testing.T) {
		handler := CharacterDeleteHandler(&fakeCharacterClient{}, nil)
		_, _, err := handler(context.Background(), nil, CharacterDeleteInput{CampaignID: "c1"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestCharacterControlSetHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &fakeCharacterClient{
			controlResp: &statev1.SetDefaultControlResponse{
				CampaignId:    "c1",
				CharacterId:   "ch1",
				ParticipantId: wrapperspb.String("p1"),
			},
		}
		handler := CharacterControlSetHandler(client, nil)
		_, result, err := handler(context.Background(), nil, CharacterControlSetInput{
			CampaignID: "c1", CharacterID: "ch1", ParticipantID: "p1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ParticipantID != "p1" {
			t.Errorf("expected participant_id %q, got %q", "p1", result.ParticipantID)
		}
	})

	t.Run("gRPC error", func(t *testing.T) {
		client := &fakeCharacterClient{controlErr: fmt.Errorf("error")}
		handler := CharacterControlSetHandler(client, nil)
		_, _, err := handler(context.Background(), nil, CharacterControlSetInput{CampaignID: "c1", CharacterID: "ch1"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nil response", func(t *testing.T) {
		client := &fakeCharacterClient{}
		handler := CharacterControlSetHandler(client, nil)
		_, _, err := handler(context.Background(), nil, CharacterControlSetInput{CampaignID: "c1", CharacterID: "ch1"})
		if err == nil {
			t.Fatal("expected error for nil response")
		}
	})

	t.Run("nil participant_id in response", func(t *testing.T) {
		client := &fakeCharacterClient{
			controlResp: &statev1.SetDefaultControlResponse{
				CampaignId:  "c1",
				CharacterId: "ch1",
			},
		}
		handler := CharacterControlSetHandler(client, nil)
		_, result, err := handler(context.Background(), nil, CharacterControlSetInput{
			CampaignID: "c1", CharacterID: "ch1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ParticipantID != "" {
			t.Errorf("expected empty participant_id, got %q", result.ParticipantID)
		}
	})
}

func TestCharacterSheetGetHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &fakeCharacterClient{
			sheetResp: &statev1.GetCharacterSheetResponse{
				Character: testCharacter("ch1", "c1", "Hero", statev1.CharacterKind_PC),
				Profile:   &statev1.CharacterProfile{CharacterId: "ch1"},
				State:     &statev1.CharacterState{CharacterId: "ch1"},
			},
		}
		getCtx := func() Context { return Context{CampaignID: "c1"} }
		handler := CharacterSheetGetHandler(client, getCtx)
		_, result, err := handler(context.Background(), nil, CharacterSheetGetInput{CharacterID: "ch1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Character.Name != "Hero" {
			t.Errorf("expected name %q, got %q", "Hero", result.Character.Name)
		}
	})

	t.Run("missing campaign context", func(t *testing.T) {
		handler := CharacterSheetGetHandler(&fakeCharacterClient{}, func() Context { return Context{} })
		_, _, err := handler(context.Background(), nil, CharacterSheetGetInput{CharacterID: "ch1"})
		if err == nil {
			t.Fatal("expected error for missing campaign context")
		}
	})

	t.Run("gRPC error", func(t *testing.T) {
		client := &fakeCharacterClient{sheetErr: fmt.Errorf("error")}
		handler := CharacterSheetGetHandler(client, func() Context { return Context{CampaignID: "c1"} })
		_, _, err := handler(context.Background(), nil, CharacterSheetGetInput{CharacterID: "ch1"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nil response", func(t *testing.T) {
		client := &fakeCharacterClient{}
		handler := CharacterSheetGetHandler(client, func() Context { return Context{CampaignID: "c1"} })
		_, _, err := handler(context.Background(), nil, CharacterSheetGetInput{CharacterID: "ch1"})
		if err == nil {
			t.Fatal("expected error for nil response")
		}
	})
}

func TestEventListHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &fakeEventClient{
			listResp: &statev1.ListEventsResponse{
				Events: []*statev1.Event{
					{CampaignId: "c1", Seq: 1, Type: "test"},
				},
				TotalSize: 1,
			},
		}
		handler := EventListHandler(client, nil)
		_, result, err := handler(context.Background(), nil, EventListInput{CampaignID: "c1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.TotalSize != 1 {
			t.Errorf("expected total_size 1, got %d", result.TotalSize)
		}
	})

	t.Run("falls back to context campaign_id", func(t *testing.T) {
		client := &fakeEventClient{
			listResp: &statev1.ListEventsResponse{},
		}
		getCtx := func() Context { return Context{CampaignID: "c1"} }
		handler := EventListHandler(client, getCtx)
		_, _, err := handler(context.Background(), nil, EventListInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("missing campaign_id", func(t *testing.T) {
		handler := EventListHandler(&fakeEventClient{}, nil)
		_, _, err := handler(context.Background(), nil, EventListInput{})
		if err == nil {
			t.Fatal("expected error for missing campaign_id")
		}
	})

	t.Run("gRPC error", func(t *testing.T) {
		client := &fakeEventClient{listErr: fmt.Errorf("error")}
		handler := EventListHandler(client, nil)
		_, _, err := handler(context.Background(), nil, EventListInput{CampaignID: "c1"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nil response", func(t *testing.T) {
		client := &fakeEventClient{}
		handler := EventListHandler(client, nil)
		_, _, err := handler(context.Background(), nil, EventListInput{CampaignID: "c1"})
		if err == nil {
			t.Fatal("expected error for nil response")
		}
	})

	t.Run("with payload_json", func(t *testing.T) {
		client := &fakeEventClient{
			listResp: &statev1.ListEventsResponse{
				Events: []*statev1.Event{
					{CampaignId: "c1", Seq: 1, Type: "test", PayloadJson: []byte(`{"key":"value"}`)},
				},
				TotalSize: 1,
			},
		}
		handler := EventListHandler(client, nil)
		_, result, err := handler(context.Background(), nil, EventListInput{CampaignID: "c1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Events[0].PayloadJSON != `{"key":"value"}` {
			t.Errorf("expected payload JSON, got %q", result.Events[0].PayloadJSON)
		}
	})
}

func TestCampaignForkHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &fakeForkClient{
			forkResp: &statev1.ForkCampaignResponse{
				Campaign:     testCampaign("c2", "Forked", statev1.CampaignStatus_DRAFT),
				ForkEventSeq: 5,
				Lineage: &statev1.Lineage{
					CampaignId:       "c2",
					ParentCampaignId: "c1",
					OriginCampaignId: "c1",
					Depth:            1,
				},
			},
		}
		handler := CampaignForkHandler(client, nil)
		_, result, err := handler(context.Background(), nil, CampaignForkInput{
			SourceCampaignID: "c1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.CampaignID != "c2" {
			t.Errorf("expected campaign_id %q, got %q", "c2", result.CampaignID)
		}
		if result.Depth != 1 {
			t.Errorf("expected depth 1, got %d", result.Depth)
		}
	})

	t.Run("nil response", func(t *testing.T) {
		client := &fakeForkClient{
			forkResp: &statev1.ForkCampaignResponse{},
		}
		handler := CampaignForkHandler(client, nil)
		_, _, err := handler(context.Background(), nil, CampaignForkInput{SourceCampaignID: "c1"})
		if err == nil {
			t.Fatal("expected error for nil campaign in response")
		}
	})

	t.Run("gRPC error", func(t *testing.T) {
		client := &fakeForkClient{forkErr: fmt.Errorf("error")}
		handler := CampaignForkHandler(client, nil)
		_, _, err := handler(context.Background(), nil, CampaignForkInput{SourceCampaignID: "c1"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nil lineage uses fallback parent", func(t *testing.T) {
		client := &fakeForkClient{
			forkResp: &statev1.ForkCampaignResponse{
				Campaign: testCampaign("c2", "Forked", statev1.CampaignStatus_DRAFT),
			},
		}
		handler := CampaignForkHandler(client, nil)
		_, result, err := handler(context.Background(), nil, CampaignForkInput{SourceCampaignID: "c1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ParentCampaignID != "c1" {
			t.Errorf("expected fallback parent %q, got %q", "c1", result.ParentCampaignID)
		}
	})

	t.Run("with session_id fork point", func(t *testing.T) {
		client := &fakeForkClient{
			forkResp: &statev1.ForkCampaignResponse{
				Campaign:     testCampaign("c2", "Forked", statev1.CampaignStatus_DRAFT),
				ForkEventSeq: 10,
				Lineage: &statev1.Lineage{
					CampaignId:       "c2",
					ParentCampaignId: "c1",
					OriginCampaignId: "c1",
					Depth:            1,
				},
			},
		}
		handler := CampaignForkHandler(client, nil)
		_, result, err := handler(context.Background(), nil, CampaignForkInput{
			SourceCampaignID: "c1",
			SessionID:        "sess-1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ForkEventSeq != 10 {
			t.Errorf("expected fork_event_seq 10, got %d", result.ForkEventSeq)
		}
	})

	t.Run("with event_seq fork point", func(t *testing.T) {
		client := &fakeForkClient{
			forkResp: &statev1.ForkCampaignResponse{
				Campaign:     testCampaign("c2", "Forked", statev1.CampaignStatus_DRAFT),
				ForkEventSeq: 5,
			},
		}
		handler := CampaignForkHandler(client, nil)
		_, result, err := handler(context.Background(), nil, CampaignForkInput{
			SourceCampaignID: "c1",
			EventSeq:         5,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ForkEventSeq != 5 {
			t.Errorf("expected fork_event_seq 5, got %d", result.ForkEventSeq)
		}
	})
}

func TestCampaignLineageHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &fakeForkClient{
			lineageResp: &statev1.GetLineageResponse{
				Lineage: &statev1.Lineage{
					CampaignId:       "c1",
					OriginCampaignId: "c1",
					Depth:            0,
				},
			},
		}
		handler := CampaignLineageHandler(client)
		_, result, err := handler(context.Background(), nil, CampaignLineageInput{CampaignID: "c1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.IsOriginal {
			t.Error("expected IsOriginal to be true for root campaign")
		}
		if result.Depth != 0 {
			t.Errorf("expected depth 0, got %d", result.Depth)
		}
	})

	t.Run("gRPC error", func(t *testing.T) {
		client := &fakeForkClient{
			lineageErr: fmt.Errorf("error"),
		}
		handler := CampaignLineageHandler(client)
		_, _, err := handler(context.Background(), nil, CampaignLineageInput{CampaignID: "c1"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nil response", func(t *testing.T) {
		client := &fakeForkClient{
			lineageResp: &statev1.GetLineageResponse{},
		}
		handler := CampaignLineageHandler(client)
		_, _, err := handler(context.Background(), nil, CampaignLineageInput{CampaignID: "c1"})
		if err == nil {
			t.Fatal("expected error for nil lineage in response")
		}
	})

	t.Run("forked campaign", func(t *testing.T) {
		client := &fakeForkClient{
			lineageResp: &statev1.GetLineageResponse{
				Lineage: &statev1.Lineage{
					CampaignId:       "c2",
					ParentCampaignId: "c1",
					OriginCampaignId: "c1",
					ForkEventSeq:     5,
					Depth:            1,
				},
			},
		}
		handler := CampaignLineageHandler(client)
		_, result, err := handler(context.Background(), nil, CampaignLineageInput{CampaignID: "c2"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.IsOriginal {
			t.Error("expected IsOriginal to be false for forked campaign")
		}
		if result.Depth != 1 {
			t.Errorf("expected depth 1, got %d", result.Depth)
		}
		if result.ParentCampaignID != "c1" {
			t.Errorf("expected parent %q, got %q", "c1", result.ParentCampaignID)
		}
	})
}

func TestCharacterProfilePatchHandler(t *testing.T) {
	t.Run("success with daggerheart fields", func(t *testing.T) {
		hp := 25
		client := &fakeCharacterClient{
			profileResp: &statev1.PatchCharacterProfileResponse{
				Profile: &statev1.CharacterProfile{CharacterId: "ch1"},
			},
		}
		getCtx := func() Context { return Context{CampaignID: "c1"} }
		handler := CharacterProfilePatchHandler(client, getCtx, nil)
		_, _, err := handler(context.Background(), nil, CharacterProfilePatchInput{
			CharacterID: "ch1",
			HpMax:       &hp,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("missing campaign context", func(t *testing.T) {
		handler := CharacterProfilePatchHandler(&fakeCharacterClient{}, func() Context { return Context{} }, nil)
		_, _, err := handler(context.Background(), nil, CharacterProfilePatchInput{CharacterID: "ch1"})
		if err == nil {
			t.Fatal("expected error for missing campaign context")
		}
	})

	t.Run("gRPC error", func(t *testing.T) {
		client := &fakeCharacterClient{profileErr: fmt.Errorf("error")}
		handler := CharacterProfilePatchHandler(client, func() Context { return Context{CampaignID: "c1"} }, nil)
		hp := 25
		_, _, err := handler(context.Background(), nil, CharacterProfilePatchInput{CharacterID: "ch1", HpMax: &hp})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nil response", func(t *testing.T) {
		client := &fakeCharacterClient{profileResp: &statev1.PatchCharacterProfileResponse{}}
		handler := CharacterProfilePatchHandler(client, func() Context { return Context{CampaignID: "c1"} }, nil)
		hp := 25
		_, _, err := handler(context.Background(), nil, CharacterProfilePatchInput{CharacterID: "ch1", HpMax: &hp})
		if err == nil {
			t.Fatal("expected error for nil profile in response")
		}
	})

	t.Run("no daggerheart fields", func(t *testing.T) {
		client := &fakeCharacterClient{
			profileResp: &statev1.PatchCharacterProfileResponse{
				Profile: &statev1.CharacterProfile{CharacterId: "ch1"},
			},
		}
		handler := CharacterProfilePatchHandler(client, func() Context { return Context{CampaignID: "c1"} }, nil)
		_, _, err := handler(context.Background(), nil, CharacterProfilePatchInput{CharacterID: "ch1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestCharacterStatePatchHandler(t *testing.T) {
	t.Run("success with daggerheart fields", func(t *testing.T) {
		hp := 15
		client := &fakeSnapshotClient{
			patchStateResp: &statev1.PatchCharacterStateResponse{
				State: &statev1.CharacterState{CharacterId: "ch1"},
			},
		}
		getCtx := func() Context { return Context{CampaignID: "c1"} }
		handler := CharacterStatePatchHandler(client, getCtx, nil)
		_, _, err := handler(context.Background(), nil, CharacterStatePatchInput{
			CharacterID: "ch1",
			Hp:          &hp,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("missing campaign context", func(t *testing.T) {
		handler := CharacterStatePatchHandler(&fakeSnapshotClient{}, func() Context { return Context{} }, nil)
		_, _, err := handler(context.Background(), nil, CharacterStatePatchInput{CharacterID: "ch1"})
		if err == nil {
			t.Fatal("expected error for missing campaign context")
		}
	})

	t.Run("gRPC error", func(t *testing.T) {
		client := &fakeSnapshotClient{patchStateErr: fmt.Errorf("error")}
		handler := CharacterStatePatchHandler(client, func() Context { return Context{CampaignID: "c1"} }, nil)
		hp := 10
		_, _, err := handler(context.Background(), nil, CharacterStatePatchInput{CharacterID: "ch1", Hp: &hp})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nil response", func(t *testing.T) {
		client := &fakeSnapshotClient{patchStateResp: &statev1.PatchCharacterStateResponse{}}
		handler := CharacterStatePatchHandler(client, func() Context { return Context{CampaignID: "c1"} }, nil)
		hp := 10
		_, _, err := handler(context.Background(), nil, CharacterStatePatchInput{CharacterID: "ch1", Hp: &hp})
		if err == nil {
			t.Fatal("expected error for nil state in response")
		}
	})

	t.Run("no daggerheart fields", func(t *testing.T) {
		client := &fakeSnapshotClient{
			patchStateResp: &statev1.PatchCharacterStateResponse{
				State: &statev1.CharacterState{CharacterId: "ch1"},
			},
		}
		handler := CharacterStatePatchHandler(client, func() Context { return Context{CampaignID: "c1"} }, nil)
		_, _, err := handler(context.Background(), nil, CharacterStatePatchInput{CharacterID: "ch1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestParticipantUpdateHandler_EdgeCases(t *testing.T) {
	t.Run("gRPC error", func(t *testing.T) {
		name := "X"
		client := &fakeParticipantClient{updateErr: fmt.Errorf("error")}
		handler := ParticipantUpdateHandler(client, nil, nil)
		_, _, err := handler(context.Background(), nil, ParticipantUpdateInput{CampaignID: "c1", ParticipantID: "p1", DisplayName: &name})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nil response", func(t *testing.T) {
		name := "X"
		client := &fakeParticipantClient{updateResp: &statev1.UpdateParticipantResponse{}}
		handler := ParticipantUpdateHandler(client, nil, nil)
		_, _, err := handler(context.Background(), nil, ParticipantUpdateInput{CampaignID: "c1", ParticipantID: "p1", DisplayName: &name})
		if err == nil {
			t.Fatal("expected error for nil participant in response")
		}
	})

	t.Run("update role", func(t *testing.T) {
		role := "GM"
		client := &fakeParticipantClient{
			updateResp: &statev1.UpdateParticipantResponse{
				Participant: testParticipant("p1", "c1", "Alice", statev1.ParticipantRole_GM),
			},
		}
		handler := ParticipantUpdateHandler(client, nil, nil)
		_, result, err := handler(context.Background(), nil, ParticipantUpdateInput{CampaignID: "c1", ParticipantID: "p1", Role: &role})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Role != "GM" {
			t.Errorf("expected role GM, got %q", result.Role)
		}
	})

	t.Run("update controller", func(t *testing.T) {
		ctrl := "AI"
		client := &fakeParticipantClient{
			updateResp: &statev1.UpdateParticipantResponse{
				Participant: testParticipant("p1", "c1", "Bot", statev1.ParticipantRole_PLAYER),
			},
		}
		handler := ParticipantUpdateHandler(client, nil, nil)
		_, _, err := handler(context.Background(), nil, ParticipantUpdateInput{CampaignID: "c1", ParticipantID: "p1", Controller: &ctrl})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestCharacterUpdateHandler_EdgeCases(t *testing.T) {
	t.Run("gRPC error", func(t *testing.T) {
		name := "X"
		client := &fakeCharacterClient{updateErr: fmt.Errorf("error")}
		handler := CharacterUpdateHandler(client, nil)
		_, _, err := handler(context.Background(), nil, CharacterUpdateInput{CampaignID: "c1", CharacterID: "ch1", Name: &name})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nil response", func(t *testing.T) {
		name := "X"
		client := &fakeCharacterClient{updateResp: &statev1.UpdateCharacterResponse{}}
		handler := CharacterUpdateHandler(client, nil)
		_, _, err := handler(context.Background(), nil, CharacterUpdateInput{CampaignID: "c1", CharacterID: "ch1", Name: &name})
		if err == nil {
			t.Fatal("expected error for nil character in response")
		}
	})

	t.Run("update kind", func(t *testing.T) {
		kind := "NPC"
		client := &fakeCharacterClient{
			updateResp: &statev1.UpdateCharacterResponse{
				Character: testCharacter("ch1", "c1", "Villain", statev1.CharacterKind_NPC),
			},
		}
		handler := CharacterUpdateHandler(client, nil)
		_, result, err := handler(context.Background(), nil, CharacterUpdateInput{CampaignID: "c1", CharacterID: "ch1", Kind: &kind})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Kind != "NPC" {
			t.Errorf("expected kind NPC, got %q", result.Kind)
		}
	})

	t.Run("update notes", func(t *testing.T) {
		notes := "brave warrior"
		client := &fakeCharacterClient{
			updateResp: &statev1.UpdateCharacterResponse{
				Character: testCharacter("ch1", "c1", "Hero", statev1.CharacterKind_PC),
			},
		}
		handler := CharacterUpdateHandler(client, nil)
		_, _, err := handler(context.Background(), nil, CharacterUpdateInput{CampaignID: "c1", CharacterID: "ch1", Notes: &notes})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestCharacterDeleteHandler_EdgeCases(t *testing.T) {
	t.Run("gRPC error", func(t *testing.T) {
		client := &fakeCharacterClient{deleteErr: fmt.Errorf("error")}
		handler := CharacterDeleteHandler(client, nil)
		_, _, err := handler(context.Background(), nil, CharacterDeleteInput{CampaignID: "c1", CharacterID: "ch1"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nil response", func(t *testing.T) {
		client := &fakeCharacterClient{deleteResp: &statev1.DeleteCharacterResponse{}}
		handler := CharacterDeleteHandler(client, nil)
		_, _, err := handler(context.Background(), nil, CharacterDeleteInput{CampaignID: "c1", CharacterID: "ch1"})
		if err == nil {
			t.Fatal("expected error for nil character in response")
		}
	})
}

func TestCampaignEndHandler_EdgeCases(t *testing.T) {
	t.Run("nil response", func(t *testing.T) {
		client := &fakeCampaignClient{endResp: &statev1.EndCampaignResponse{}}
		handler := CampaignEndHandler(client, nil, nil)
		_, _, err := handler(context.Background(), nil, CampaignStatusChangeInput{CampaignID: "c1"})
		if err == nil {
			t.Fatal("expected error for nil campaign in response")
		}
	})

	t.Run("falls back to context campaign_id", func(t *testing.T) {
		client := &fakeCampaignClient{
			endResp: &statev1.EndCampaignResponse{
				Campaign: testCampaign("c1", "X", statev1.CampaignStatus_COMPLETED),
			},
		}
		getCtx := func() Context { return Context{CampaignID: "c1"} }
		handler := CampaignEndHandler(client, getCtx, nil)
		_, _, err := handler(context.Background(), nil, CampaignStatusChangeInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestCampaignArchiveHandler_EdgeCases(t *testing.T) {
	t.Run("gRPC error", func(t *testing.T) {
		client := &fakeCampaignClient{archiveErr: fmt.Errorf("connection refused")}
		handler := CampaignArchiveHandler(client, nil, nil)
		_, _, err := handler(context.Background(), nil, CampaignStatusChangeInput{CampaignID: "c1"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nil response", func(t *testing.T) {
		client := &fakeCampaignClient{archiveResp: &statev1.ArchiveCampaignResponse{}}
		handler := CampaignArchiveHandler(client, nil, nil)
		_, _, err := handler(context.Background(), nil, CampaignStatusChangeInput{CampaignID: "c1"})
		if err == nil {
			t.Fatal("expected error for nil campaign in response")
		}
	})

	t.Run("falls back to context campaign_id", func(t *testing.T) {
		client := &fakeCampaignClient{
			archiveResp: &statev1.ArchiveCampaignResponse{
				Campaign: testCampaign("c1", "X", statev1.CampaignStatus_ARCHIVED),
			},
		}
		getCtx := func() Context { return Context{CampaignID: "c1"} }
		handler := CampaignArchiveHandler(client, getCtx, nil)
		_, _, err := handler(context.Background(), nil, CampaignStatusChangeInput{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestParticipantDeleteHandler_EdgeCases(t *testing.T) {
	t.Run("gRPC error", func(t *testing.T) {
		client := &fakeParticipantClient{deleteErr: fmt.Errorf("error")}
		handler := ParticipantDeleteHandler(client, nil, nil)
		_, _, err := handler(context.Background(), nil, ParticipantDeleteInput{CampaignID: "c1", ParticipantID: "p1"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nil response", func(t *testing.T) {
		client := &fakeParticipantClient{deleteResp: &statev1.DeleteParticipantResponse{}}
		handler := ParticipantDeleteHandler(client, nil, nil)
		_, _, err := handler(context.Background(), nil, ParticipantDeleteInput{CampaignID: "c1", ParticipantID: "p1"})
		if err == nil {
			t.Fatal("expected error for nil participant in response")
		}
	})
}

func TestCharacterProfilePatchHandler_AllFields(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	t.Run("all daggerheart fields", func(t *testing.T) {
		client := &fakeCharacterClient{
			profileResp: &statev1.PatchCharacterProfileResponse{
				Profile: &statev1.CharacterProfile{CharacterId: "ch1"},
			},
		}
		getCtx := func() Context { return Context{CampaignID: "c1"} }
		handler := CharacterProfilePatchHandler(client, getCtx, nil)
		_, _, err := handler(context.Background(), nil, CharacterProfilePatchInput{
			CharacterID:     "ch1",
			HpMax:           intPtr(25),
			StressMax:       intPtr(6),
			Evasion:         intPtr(10),
			MajorThreshold:  intPtr(7),
			SevereThreshold: intPtr(14),
			Agility:         intPtr(2),
			Strength:        intPtr(1),
			Finesse:         intPtr(3),
			Instinct:        intPtr(0),
			Presence:        intPtr(2),
			Knowledge:       intPtr(1),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("notifies resource updates", func(t *testing.T) {
		client := &fakeCharacterClient{
			profileResp: &statev1.PatchCharacterProfileResponse{
				Profile: &statev1.CharacterProfile{CharacterId: "ch1"},
			},
		}
		var notified []string
		notify := func(ctx context.Context, uri string) { notified = append(notified, uri) }
		getCtx := func() Context { return Context{CampaignID: "c1"} }
		handler := CharacterProfilePatchHandler(client, getCtx, notify)
		hp := 25
		_, _, err := handler(context.Background(), nil, CharacterProfilePatchInput{CharacterID: "ch1", HpMax: &hp})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(notified) == 0 {
			t.Error("expected resource update notification")
		}
	})
}
