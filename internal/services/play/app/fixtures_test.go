package app

import (
	"net/http"
	"testing"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func playTestState() *gamev1.InteractionState {
	return &gamev1.InteractionState{
		CampaignId:   "c1",
		CampaignName: "The Guildhouse",
		Viewer:       &gamev1.InteractionViewer{ParticipantId: "p1", Name: "Avery", Role: gamev1.ParticipantRole_PLAYER},
		ActiveSession: &gamev1.InteractionSession{
			SessionId: "s1",
			Name:      "Session One",
		},
	}
}

func hubDepsFromServer(s *Server) realtimeHubDeps {
	return realtimeHubDeps{
		resolveUserID: s.resolvePlayUserID,
		application:   s.application,
		aiDebug:       s.deps.AIDebug,
		transcripts:   s.deps.Transcripts,
		events:        s.deps.CampaignUpdates,
	}
}

func newAuthedPlayServer(interaction *recordingInteractionClient, transcripts *scriptTranscriptStore) *Server {
	server := &Server{
		deps: Dependencies{
			Auth:               &fakePlayAuthClient{sessions: map[string]string{"ps-1": "user-1"}},
			AIDebug:            &fakePlayAIDebugClient{},
			Interaction:        interaction,
			Campaign:           fakePlayCampaignClient{response: &gamev1.GetCampaignResponse{}},
			System:             fakePlaySystemClient{response: &gamev1.GetGameSystemResponse{}},
			Participants:       fakePlayParticipantClient{response: &gamev1.ListParticipantsResponse{}},
			Characters:         fakePlayCharacterClient{listResponse: &gamev1.ListCharactersResponse{}},
			DaggerheartContent: &fakeDaggerheartContentClient{},
			Transcripts:        transcripts,
		},
		shellAssets:     shellAssets{devServerURL: "http://localhost:5173"},
		httpServer:      &http.Server{},
		webFallbackPort: "8080",
	}
	server.realtime = newRealtimeHub(server)
	return server
}

func testPlayLaunchGrantConfig(t *testing.T) playlaunchgrant.Config {
	t.Helper()
	now := time.Date(2026, time.March, 13, 12, 0, 0, 0, time.UTC)
	return playlaunchgrant.Config{
		Issuer:   "fracturing-space-web",
		Audience: "fracturing-space-play",
		HMACKey:  []byte("0123456789abcdef0123456789abcdef"),
		TTL:      2 * time.Minute,
		Now: func() time.Time {
			return now
		},
	}
}

func enrichedParticipantResponse() *gamev1.ListParticipantsResponse {
	return &gamev1.ListParticipantsResponse{
		Participants: []*gamev1.Participant{
			{Id: "p1", Name: "Avery", Role: gamev1.ParticipantRole_PLAYER},
			{Id: "p2", Name: "Guide", Role: gamev1.ParticipantRole_GM},
		},
	}
}

func enrichedCharacterResponse() *gamev1.ListCharactersResponse {
	return &gamev1.ListCharactersResponse{
		Characters: []*gamev1.Character{
			{
				Id:                 "char-1",
				CampaignId:         "c1",
				Name:               "Lark",
				Kind:               gamev1.CharacterKind_PC,
				OwnerParticipantId: &wrapperspb.StringValue{Value: "p1"},
			},
		},
	}
}

func enrichedCharacterSheetResponse() *gamev1.GetCharacterSheetResponse {
	return &gamev1.GetCharacterSheetResponse{
		Character: enrichedCharacterResponse().GetCharacters()[0],
		Profile: &gamev1.CharacterProfile{
			CampaignId:  "c1",
			CharacterId: "char-1",
			SystemProfile: &gamev1.CharacterProfile_Daggerheart{
				Daggerheart: &daggerheartv1.DaggerheartProfile{
					Level: 1,
					HpMax: 10,
					Heritage: &daggerheartv1.DaggerheartHeritageSelection{
						AncestryName:  "Human",
						CommunityName: "Slyborne",
					},
					ActiveClassFeatures: []*daggerheartv1.DaggerheartActiveClassFeature{
						{
							Name:        "Rogue's Dodge",
							Description: "Spend 3 Hope to gain +2 Evasion until an attack succeeds against you.",
							HopeFeature: true,
						},
						{
							Name:        "Sneak Attack",
							Description: "When you have advantage on a melee attack, deal an extra 1d8 damage.",
						},
					},
					PrimaryWeapon: &daggerheartv1.DaggerheartSheetWeaponSummary{
						Name:       "Sword",
						Trait:      "Finesse",
						Range:      "melee",
						DamageDice: "1d8",
						DamageType: "physical",
						Feature:    "Versatile",
					},
					ActiveArmor: &daggerheartv1.DaggerheartSheetArmorSummary{
						Name:      "Leather",
						BaseScore: 2,
						Feature:   "Quiet",
					},
					DomainCardIds: []string{
						"domain_card.valor-i-am-your-shield",
						"domain_card.blade-get-back-up",
					},
				},
			},
		},
	}
}
