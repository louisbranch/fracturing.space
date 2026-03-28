package protocol

import (
	"testing"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestProtocolHelperBranches(t *testing.T) {
	t.Parallel()

	if got := ProtoEnumToLower(gamev1.ParticipantRole_PLAYER, gamev1.ParticipantRole_ROLE_UNSPECIFIED, "PARTICIPANT_ROLE_"); got != "player" {
		t.Fatalf("ProtoEnumToLower(player) = %q, want %q", got, "player")
	}
	if got := ProtoEnumToLower(gamev1.ParticipantRole_ROLE_UNSPECIFIED, gamev1.ParticipantRole_ROLE_UNSPECIFIED, "PARTICIPANT_ROLE_"); got != "" {
		t.Fatalf("ProtoEnumToLower(unspecified) = %q, want empty", got)
	}

	if got := PronounsString(nil); got != "" {
		t.Fatalf("PronounsString(nil) = %q, want empty", got)
	}
	if got := PronounsString(&commonv1.Pronouns{
		Value: &commonv1.Pronouns_Kind{Kind: commonv1.Pronoun_PRONOUN_SHE_HER},
	}); got != "she/her" {
		t.Fatalf("PronounsString(kind) = %q, want %q", got, "she/her")
	}
	if got := PronounsString(&commonv1.Pronouns{
		Value: &commonv1.Pronouns_Custom{Custom: " xe/xem "},
	}); got != "xe/xem" {
		t.Fatalf("PronounsString(custom) = %q, want %q", got, "xe/xem")
	}
	if got := PronounsString(&commonv1.Pronouns{
		Value: &commonv1.Pronouns_Kind{Kind: commonv1.Pronoun_PRONOUN_UNSPECIFIED},
	}); got != "" {
		t.Fatalf("PronounsString(unspecified) = %q, want empty", got)
	}
}

func TestInteractionMappingAdditionalBranches(t *testing.T) {
	t.Parallel()

	if got := InteractionStateFromGameState(nil); got != (InteractionState{}) {
		t.Fatalf("InteractionStateFromGameState(nil) = %#v, want zero value", got)
	}
	if got := ViewerFromGameViewer(&gamev1.InteractionViewer{}); got != nil {
		t.Fatalf("ViewerFromGameViewer(empty) = %#v, want nil", got)
	}
	if got := SessionFromGameSession(&gamev1.InteractionSession{}); got != nil {
		t.Fatalf("SessionFromGameSession(empty) = %#v, want nil", got)
	}
	if got := SceneFromGameScene(&gamev1.InteractionScene{}); got != nil {
		t.Fatalf("SceneFromGameScene(empty) = %#v, want nil", got)
	}

	scene := SceneFromGameScene(&gamev1.InteractionScene{
		SceneId: " scene-1 ",
		InteractionHistory: []*gamev1.GMInteraction{
			nil,
			{},
			{
				InteractionId: " interaction-1 ",
				SceneId:       " scene-1 ",
				ParticipantId: " p1 ",
				CharacterIds:  []string{" ch-1 ", " ", "ch-2"},
				Illustration: &gamev1.GMInteractionIllustration{
					ImageUrl: " https://cdn.example.com/scene.png ",
					Alt:      " fog bank ",
					Caption:  " thick mist ",
				},
				Beats: []*gamev1.GMInteractionBeat{
					{BeatId: " beat-1 ", Type: gamev1.GMInteractionBeatType_GM_INTERACTION_BEAT_TYPE_CONSEQUENCE, Text: " danger rises "},
				},
				CreatedAt: &timestamppb.Timestamp{Seconds: 1710331200},
			},
		},
	})
	if scene == nil || len(scene.InteractionHistory) != 1 {
		t.Fatalf("SceneFromGameScene(history) = %#v", scene)
	}
	if scene.InteractionHistory[0].Illustration == nil || scene.InteractionHistory[0].Illustration.Alt != "fog bank" {
		t.Fatalf("illustration = %#v", scene.InteractionHistory[0].Illustration)
	}
	if len(scene.InteractionHistory[0].CharacterIDs) != 2 {
		t.Fatalf("CharacterIDs = %#v, want trimmed non-empty values", scene.InteractionHistory[0].CharacterIDs)
	}
	if scene.InteractionHistory[0].Beats[0].Type != "consequence" {
		t.Fatalf("beat type = %q, want consequence", scene.InteractionHistory[0].Beats[0].Type)
	}
}

func TestParticipantAndAIDebugAdditionalBranches(t *testing.T) {
	t.Parallel()

	if got := ParticipantFromGameParticipant("https://cdn.example.com/assets", nil); got.ID != "" || got.Name != "" || got.Role != "" || got.AvatarURL != "" || len(got.CharacterIDs) != 0 {
		t.Fatalf("ParticipantFromGameParticipant(nil) = %#v, want zero value fields", got)
	}

	fallbackToUser := ParticipantFromGameParticipant("https://cdn.example.com/assets", &gamev1.Participant{
		UserId:        " user-1 ",
		Name:          " Avery ",
		Role:          gamev1.ParticipantRole_PLAYER,
		AvatarSetId:   " avatar_set_v1 ",
		AvatarAssetId: " ceremonial_choir_lead ",
	})
	if fallbackToUser.Name != "Avery" || fallbackToUser.Role != "player" || fallbackToUser.AvatarURL == "" {
		t.Fatalf("fallbackToUser = %#v", fallbackToUser)
	}

	fallbackToCampaign := ParticipantFromGameParticipant("https://cdn.example.com/assets", &gamev1.Participant{
		CampaignId:    " camp-1 ",
		AvatarSetId:   " avatar_set_v1 ",
		AvatarAssetId: " ceremonial_choir_lead ",
	})
	if fallbackToCampaign.AvatarURL == "" {
		t.Fatalf("fallbackToCampaign.AvatarURL = empty, want asset-backed URL")
	}

	if got := aiDebugEntryFromProto(nil); got != (AIDebugEntry{}) {
		t.Fatalf("aiDebugEntryFromProto(nil) = %#v, want zero value", got)
	}
	if got := aiDebugUsageFromProto(&aiv1.Usage{}); got != nil {
		t.Fatalf("aiDebugUsageFromProto(zero) = %#v, want nil", got)
	}
	if got := aiProviderString(aiv1.Provider_PROVIDER_UNSPECIFIED); got != "" {
		t.Fatalf("aiProviderString(unspecified) = %q, want empty", got)
	}
	if got := aiDebugTurnStatusString(aiv1.CampaignDebugTurnStatus_CAMPAIGN_DEBUG_TURN_STATUS_UNSPECIFIED); got != "" {
		t.Fatalf("aiDebugTurnStatusString(unspecified) = %q, want empty", got)
	}
	if got := aiDebugEntryKindString(aiv1.CampaignDebugEntryKind_CAMPAIGN_DEBUG_ENTRY_KIND_UNSPECIFIED); got != "" {
		t.Fatalf("aiDebugEntryKindString(unspecified) = %q, want empty", got)
	}
}
