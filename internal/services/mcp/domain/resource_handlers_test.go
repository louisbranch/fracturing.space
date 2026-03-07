package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func readReq(uri string) *mcp.ReadResourceRequest {
	return &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: uri},
	}
}

func TestCampaignListResourceHandler(t *testing.T) {
	t.Run("nil client", func(t *testing.T) {
		handler := CampaignListResourceHandler(nil)
		_, err := handler(context.Background(), readReq("campaigns://list"))
		if err == nil {
			t.Fatal("expected error for nil client")
		}
	})

	t.Run("empty list", func(t *testing.T) {
		client := &fakeCampaignClient{
			listResp: &statev1.ListCampaignsResponse{},
		}
		handler := CampaignListResourceHandler(client)
		result, err := handler(context.Background(), readReq("campaigns://list"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var payload CampaignListPayload
		if err := json.Unmarshal([]byte(result.Contents[0].Text), &payload); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if payload.Campaigns != nil {
			t.Errorf("expected nil campaigns, got %v", payload.Campaigns)
		}
	})

	t.Run("populated list", func(t *testing.T) {
		now := timestamppb.Now()
		client := &fakeCampaignClient{
			listResp: &statev1.ListCampaignsResponse{
				Campaigns: []*statev1.Campaign{
					{Id: "c1", Name: "First", Status: statev1.CampaignStatus_ACTIVE, GmMode: statev1.GmMode_HUMAN, CreatedAt: now},
					{Id: "c2", Name: "Second", Status: statev1.CampaignStatus_DRAFT, GmMode: statev1.GmMode_AI},
				},
			},
		}
		handler := CampaignListResourceHandler(client)
		result, err := handler(context.Background(), readReq("campaigns://list"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var payload CampaignListPayload
		if err := json.Unmarshal([]byte(result.Contents[0].Text), &payload); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(payload.Campaigns) != 2 {
			t.Fatalf("expected 2 campaigns, got %d", len(payload.Campaigns))
		}
		if payload.Campaigns[0].ID != "c1" || payload.Campaigns[0].Status != "ACTIVE" {
			t.Errorf("unexpected first campaign: %+v", payload.Campaigns[0])
		}
	})

	t.Run("list error", func(t *testing.T) {
		client := &fakeCampaignClient{
			listErr: fmt.Errorf("rpc error"),
		}
		handler := CampaignListResourceHandler(client)
		_, err := handler(context.Background(), readReq("campaigns://list"))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestCampaignResourceHandler(t *testing.T) {
	t.Run("nil client", func(t *testing.T) {
		handler := CampaignResourceHandler(nil)
		_, err := handler(context.Background(), readReq("campaign://c1"))
		if err == nil {
			t.Fatal("expected error for nil client")
		}
	})

	t.Run("nil request", func(t *testing.T) {
		handler := CampaignResourceHandler(&fakeCampaignClient{})
		_, err := handler(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error for nil request")
		}
	})

	t.Run("success", func(t *testing.T) {
		client := &fakeCampaignClient{
			getResp: &statev1.GetCampaignResponse{
				Campaign: testCampaign("c1", "My Campaign", statev1.CampaignStatus_ACTIVE),
			},
		}
		handler := CampaignResourceHandler(client)
		result, err := handler(context.Background(), readReq("campaign://c1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var payload CampaignPayload
		if err := json.Unmarshal([]byte(result.Contents[0].Text), &payload); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if payload.Campaign.ID != "c1" {
			t.Errorf("expected id %q, got %q", "c1", payload.Campaign.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		client := &fakeCampaignClient{
			getErr: fmt.Errorf("rpc error"),
		}
		handler := CampaignResourceHandler(client)
		_, err := handler(context.Background(), readReq("campaign://c-notfound"))
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("nil response", func(t *testing.T) {
		client := &fakeCampaignClient{
			getResp: &statev1.GetCampaignResponse{},
		}
		handler := CampaignResourceHandler(client)
		_, err := handler(context.Background(), readReq("campaign://c1"))
		if err == nil {
			t.Fatal("expected error for nil campaign in response")
		}
	})

	t.Run("empty URI", func(t *testing.T) {
		handler := CampaignResourceHandler(&fakeCampaignClient{})
		_, err := handler(context.Background(), readReq(""))
		if err == nil {
			t.Fatal("expected error for empty URI")
		}
	})

	t.Run("invalid URI format", func(t *testing.T) {
		handler := CampaignResourceHandler(&fakeCampaignClient{})
		_, err := handler(context.Background(), readReq("invalid://c1"))
		if err == nil {
			t.Fatal("expected error for invalid URI format")
		}
	})

	t.Run("URI with path segments", func(t *testing.T) {
		handler := CampaignResourceHandler(&fakeCampaignClient{})
		_, err := handler(context.Background(), readReq("campaign://c1/participants"))
		if err == nil {
			t.Fatal("expected error for URI with path segments")
		}
	})
}

func TestSessionListResourceHandler(t *testing.T) {
	t.Run("nil client", func(t *testing.T) {
		handler := SessionListResourceHandler(nil)
		_, err := handler(context.Background(), readReq("campaign://c1/sessions"))
		if err == nil {
			t.Fatal("expected error for nil client")
		}
	})

	t.Run("nil request", func(t *testing.T) {
		handler := SessionListResourceHandler(&fakeSessionClient{})
		_, err := handler(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error for nil request")
		}
	})

	t.Run("populated list", func(t *testing.T) {
		client := &fakeSessionClient{
			listResp: &statev1.ListSessionsResponse{
				Sessions: []*statev1.Session{
					testSession("s1", "c1", "Session 1", statev1.SessionStatus_SESSION_ACTIVE),
				},
			},
		}
		handler := SessionListResourceHandler(client)
		result, err := handler(context.Background(), readReq("campaign://c1/sessions"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var payload SessionListPayload
		if err := json.Unmarshal([]byte(result.Contents[0].Text), &payload); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(payload.Sessions) != 1 {
			t.Fatalf("expected 1 session, got %d", len(payload.Sessions))
		}
		if payload.Sessions[0].Status != "ACTIVE" {
			t.Errorf("expected status ACTIVE, got %q", payload.Sessions[0].Status)
		}
	})
}

func TestParticipantListResourceHandler(t *testing.T) {
	t.Run("nil client", func(t *testing.T) {
		handler := ParticipantListResourceHandler(nil)
		_, err := handler(context.Background(), readReq("campaign://c1/participants"))
		if err == nil {
			t.Fatal("expected error for nil client")
		}
	})

	t.Run("nil request", func(t *testing.T) {
		handler := ParticipantListResourceHandler(&fakeParticipantClient{})
		_, err := handler(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error for nil request")
		}
	})

	t.Run("populated list", func(t *testing.T) {
		client := &fakeParticipantClient{
			listResp: &statev1.ListParticipantsResponse{
				Participants: []*statev1.Participant{
					testParticipant("p1", "c1", "Alice", statev1.ParticipantRole_GM),
					testParticipant("p2", "c1", "Bob", statev1.ParticipantRole_PLAYER),
				},
			},
		}
		handler := ParticipantListResourceHandler(client)
		result, err := handler(context.Background(), readReq("campaign://c1/participants"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var payload ParticipantListPayload
		if err := json.Unmarshal([]byte(result.Contents[0].Text), &payload); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(payload.Participants) != 2 {
			t.Fatalf("expected 2 participants, got %d", len(payload.Participants))
		}
	})
}

func TestCharacterListResourceHandler(t *testing.T) {
	t.Run("nil client", func(t *testing.T) {
		handler := CharacterListResourceHandler(nil)
		_, err := handler(context.Background(), readReq("campaign://c1/characters"))
		if err == nil {
			t.Fatal("expected error for nil client")
		}
	})

	t.Run("nil request", func(t *testing.T) {
		handler := CharacterListResourceHandler(&fakeCharacterClient{})
		_, err := handler(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error for nil request")
		}
	})

	t.Run("populated list", func(t *testing.T) {
		client := &fakeCharacterClient{
			listResp: &statev1.ListCharactersResponse{
				Characters: []*statev1.Character{
					testCharacter("ch1", "c1", "Hero", statev1.CharacterKind_PC),
				},
			},
		}
		handler := CharacterListResourceHandler(client)
		result, err := handler(context.Background(), readReq("campaign://c1/characters"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var payload CharacterListPayload
		if err := json.Unmarshal([]byte(result.Contents[0].Text), &payload); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(payload.Characters) != 1 {
			t.Fatalf("expected 1 character, got %d", len(payload.Characters))
		}
		if payload.Characters[0].Kind != "PC" {
			t.Errorf("expected kind PC, got %q", payload.Characters[0].Kind)
		}
	})

	t.Run("gRPC error", func(t *testing.T) {
		client := &fakeCharacterClient{listErr: fmt.Errorf("error")}
		handler := CharacterListResourceHandler(client)
		_, err := handler(context.Background(), readReq("campaign://c1/characters"))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestEventsListResourceHandler(t *testing.T) {
	t.Run("nil client", func(t *testing.T) {
		handler := EventsListResourceHandler(nil)
		_, err := handler(context.Background(), readReq("campaign://c1/events"))
		if err == nil {
			t.Fatal("expected error for nil client")
		}
	})

	t.Run("nil request", func(t *testing.T) {
		handler := EventsListResourceHandler(&fakeEventClient{})
		_, err := handler(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error for nil request")
		}
	})

	t.Run("populated list", func(t *testing.T) {
		client := &fakeEventClient{
			listResp: &statev1.ListEventsResponse{
				Events: []*statev1.Event{
					{
						CampaignId: "c1",
						Seq:        1,
						Type:       "campaign.created",
						ActorType:  "system",
					},
				},
				TotalSize: 1,
			},
		}
		handler := EventsListResourceHandler(client)
		result, err := handler(context.Background(), readReq("campaign://c1/events"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var payload EventsListPayload
		if err := json.Unmarshal([]byte(result.Contents[0].Text), &payload); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if payload.TotalSize != 1 {
			t.Errorf("expected total_size 1, got %d", payload.TotalSize)
		}
		if len(payload.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(payload.Events))
		}
	})

	t.Run("gRPC error", func(t *testing.T) {
		client := &fakeEventClient{listErr: fmt.Errorf("error")}
		handler := EventsListResourceHandler(client)
		_, err := handler(context.Background(), readReq("campaign://c1/events"))
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("with payload_json", func(t *testing.T) {
		client := &fakeEventClient{
			listResp: &statev1.ListEventsResponse{
				Events: []*statev1.Event{
					{CampaignId: "c1", Seq: 1, Type: "test", PayloadJson: []byte(`{"k":"v"}`)},
				},
				TotalSize: 1,
			},
		}
		handler := EventsListResourceHandler(client)
		result, err := handler(context.Background(), readReq("campaign://c1/events"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var payload EventsListPayload
		if err := json.Unmarshal([]byte(result.Contents[0].Text), &payload); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if payload.Events[0].PayloadJSON != `{"k":"v"}` {
			t.Errorf("expected payload JSON, got %q", payload.Events[0].PayloadJSON)
		}
	})
}

func TestSessionListResourceHandler_gRPCError(t *testing.T) {
	client := &fakeSessionClient{listErr: fmt.Errorf("error")}
	handler := SessionListResourceHandler(client)
	_, err := handler(context.Background(), readReq("campaign://c1/sessions"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParticipantListResourceHandler_gRPCError(t *testing.T) {
	client := &fakeParticipantClient{listErr: fmt.Errorf("error")}
	handler := ParticipantListResourceHandler(client)
	_, err := handler(context.Background(), readReq("campaign://c1/participants"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCampaignListResourceHandler_nilResponse(t *testing.T) {
	client := &fakeCampaignClient{}
	handler := CampaignListResourceHandler(client)
	_, err := handler(context.Background(), readReq("campaigns://list"))
	if err == nil {
		t.Fatal("expected error for nil response")
	}
}

func TestSessionListResourceHandler_nilResponse(t *testing.T) {
	client := &fakeSessionClient{listResp: &statev1.ListSessionsResponse{}}
	handler := SessionListResourceHandler(client)
	// Empty response (no sessions) should succeed.
	result, err := handler(context.Background(), readReq("campaign://c1/sessions"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || len(result.Contents) == 0 {
		t.Fatal("expected non-nil result")
	}
}

func TestSessionListResourceHandler_invalidURI(t *testing.T) {
	client := &fakeSessionClient{}
	handler := SessionListResourceHandler(client)
	_, err := handler(context.Background(), readReq("invalid://no-campaign"))
	if err == nil {
		t.Fatal("expected error for invalid URI")
	}
}

func TestParticipantListResourceHandler_nilResponse(t *testing.T) {
	client := &fakeParticipantClient{}
	handler := ParticipantListResourceHandler(client)
	_, err := handler(context.Background(), readReq("campaign://c1/participants"))
	if err == nil {
		t.Fatal("expected error for nil response")
	}
}

func TestParticipantListResourceHandler_invalidURI(t *testing.T) {
	client := &fakeParticipantClient{}
	handler := ParticipantListResourceHandler(client)
	_, err := handler(context.Background(), readReq("invalid://no-campaign"))
	if err == nil {
		t.Fatal("expected error for invalid URI")
	}
}

func TestCharacterListResourceHandler_nilResponse(t *testing.T) {
	client := &fakeCharacterClient{}
	handler := CharacterListResourceHandler(client)
	_, err := handler(context.Background(), readReq("campaign://c1/characters"))
	if err == nil {
		t.Fatal("expected error for nil response")
	}
}

func TestCharacterListResourceHandler_invalidURI(t *testing.T) {
	client := &fakeCharacterClient{}
	handler := CharacterListResourceHandler(client)
	_, err := handler(context.Background(), readReq("invalid://no-campaign"))
	if err == nil {
		t.Fatal("expected error for invalid URI")
	}
}

func TestEventsListResourceHandler_invalidURI(t *testing.T) {
	client := &fakeEventClient{}
	handler := EventsListResourceHandler(client)
	_, err := handler(context.Background(), readReq("invalid://no-campaign"))
	if err == nil {
		t.Fatal("expected error for invalid URI")
	}
}

func TestEventsListResourceHandler_nilResponse(t *testing.T) {
	client := &fakeEventClient{}
	handler := EventsListResourceHandler(client)
	_, err := handler(context.Background(), readReq("campaign://c1/events"))
	if err == nil {
		t.Fatal("expected error for nil response")
	}
}

func TestCampaignResourceHandler_nilResponse(t *testing.T) {
	client := &fakeCampaignClient{
		getResp: &statev1.GetCampaignResponse{},
	}
	handler := CampaignResourceHandler(client)
	_, err := handler(context.Background(), readReq("campaign://c1"))
	if err == nil {
		t.Fatal("expected error for nil campaign in response")
	}
}
