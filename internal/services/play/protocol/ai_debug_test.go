package protocol

import (
	"testing"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAIDebugProtoMappers(t *testing.T) {
	t.Parallel()

	page := AIDebugTurnsPageFromProto(&aiv1.ListCampaignDebugTurnsResponse{
		Turns: []*aiv1.CampaignDebugTurnSummary{{
			Id:            " turn-1 ",
			TurnToken:     " token-1 ",
			ParticipantId: " participant-1 ",
			Provider:      aiv1.Provider_PROVIDER_OPENAI,
			Model:         " gpt-5.4 ",
			Status:        aiv1.CampaignDebugTurnStatus_CAMPAIGN_DEBUG_TURN_STATUS_RUNNING,
			LastError:     " delayed ",
			Usage: &aiv1.Usage{
				InputTokens:  10,
				OutputTokens: 5,
				TotalTokens:  15,
			},
			StartedAt:   timestamppb.New(timestamppb.Now().AsTime().UTC()),
			UpdatedAt:   timestamppb.New(timestamppb.Now().AsTime().UTC()),
			CompletedAt: timestamppb.New(timestamppb.Now().AsTime().UTC()),
			EntryCount:  2,
		}},
		NextPageToken: " next-token ",
	})
	if len(page.Turns) != 1 || page.Turns[0].ID != "turn-1" || page.Turns[0].Provider != "openai" || page.NextPageToken != "next-token" {
		t.Fatalf("page = %#v", page)
	}

	turn := AIDebugTurnFromProto(&aiv1.CampaignDebugTurn{
		Id:            " turn-2 ",
		TurnToken:     " token-2 ",
		ParticipantId: " participant-2 ",
		Provider:      aiv1.Provider_PROVIDER_OPENAI,
		Model:         " gpt-5.4-mini ",
		Status:        aiv1.CampaignDebugTurnStatus_CAMPAIGN_DEBUG_TURN_STATUS_FAILED,
		LastError:     " missing scene ",
		Entries: []*aiv1.CampaignDebugEntry{{
			Sequence:         7,
			Kind:             aiv1.CampaignDebugEntryKind_CAMPAIGN_DEBUG_ENTRY_KIND_TOOL_RESULT,
			ToolName:         " scene_create ",
			Payload:          `{"error":"missing scene"}`,
			PayloadTruncated: true,
			CallId:           " call-1 ",
			ResponseId:       " resp-1 ",
			IsError:          true,
			CreatedAt:        timestamppb.Now(),
			Usage:            &aiv1.Usage{ReasoningTokens: 4},
		}},
	})
	if turn.ID != "turn-2" || turn.Status != "failed" || turn.EntryCount != 1 {
		t.Fatalf("turn = %#v", turn)
	}
	if len(turn.Entries) != 1 || turn.Entries[0].Kind != "tool_result" || turn.Entries[0].ToolName != "scene_create" || turn.Entries[0].Usage == nil || turn.Entries[0].Usage.ReasoningTokens != 4 {
		t.Fatalf("entries = %#v", turn.Entries)
	}

	update := AIDebugTurnUpdateFromProto(&aiv1.CampaignDebugTurnUpdate{
		Turn: &aiv1.CampaignDebugTurnSummary{
			Id:         " turn-3 ",
			Status:     aiv1.CampaignDebugTurnStatus_CAMPAIGN_DEBUG_TURN_STATUS_SUCCEEDED,
			EntryCount: 3,
		},
		AppendedEntries: []*aiv1.CampaignDebugEntry{{
			Sequence: 3,
			Kind:     aiv1.CampaignDebugEntryKind_CAMPAIGN_DEBUG_ENTRY_KIND_MODEL_RESPONSE,
			Payload:  "Done",
		}},
	})
	if update.Turn.ID != "turn-3" || update.Turn.Status != "succeeded" || len(update.AppendedEntries) != 1 || update.AppendedEntries[0].Kind != "model_response" {
		t.Fatalf("update = %#v", update)
	}
}

func TestAIDebugProtoMappersHandleNilAndUnknownValues(t *testing.T) {
	t.Parallel()

	if got := AIDebugTurnsPageFromProto(nil); len(got.Turns) != 0 {
		t.Fatalf("nil page = %#v", got)
	}
	if got := AIDebugTurnFromProto(nil); len(got.Entries) != 0 {
		t.Fatalf("nil turn = %#v", got)
	}
	if got := AIDebugTurnUpdateFromProto(nil); len(got.AppendedEntries) != 0 {
		t.Fatalf("nil update = %#v", got)
	}
	if usage := aiDebugUsageFromProto(&aiv1.Usage{}); usage != nil {
		t.Fatalf("zero usage = %#v, want nil", usage)
	}
	if got := aiProviderString(aiv1.Provider_PROVIDER_UNSPECIFIED); got != "" {
		t.Fatalf("aiProviderString = %q, want empty", got)
	}
	if got := aiDebugTurnStatusString(aiv1.CampaignDebugTurnStatus_CAMPAIGN_DEBUG_TURN_STATUS_UNSPECIFIED); got != "" {
		t.Fatalf("aiDebugTurnStatusString = %q, want empty", got)
	}
	if got := aiDebugEntryKindString(aiv1.CampaignDebugEntryKind_CAMPAIGN_DEBUG_ENTRY_KIND_UNSPECIFIED); got != "" {
		t.Fatalf("aiDebugEntryKindString = %q, want empty", got)
	}
}
