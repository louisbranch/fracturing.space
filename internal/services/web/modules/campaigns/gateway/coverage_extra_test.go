package gateway

import (
	"testing"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
)

func TestGatewayConstructorWrappersRequireOwnedDependencies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func() any
	}{
		{
			name: "authorization",
			run: func() any {
				if got := NewAuthorizationGateway(AuthorizationDeps{}); got != nil {
					t.Fatalf("NewAuthorizationGateway() = %#v, want nil", got)
				}
				return NewAuthorizationGateway(AuthorizationDeps{Client: contractAuthorizationClient{}})
			},
		},
		{
			name: "batch authorization",
			run: func() any {
				if got := NewBatchAuthorizationGateway(AuthorizationDeps{}); got != nil {
					t.Fatalf("NewBatchAuthorizationGateway() = %#v, want nil", got)
				}
				return NewBatchAuthorizationGateway(AuthorizationDeps{Client: contractAuthorizationClient{}})
			},
		},
		{
			name: "catalog read",
			run: func() any {
				if got := NewCatalogReadGateway(CatalogReadDeps{}, "https://cdn.example.test"); got != nil {
					t.Fatalf("NewCatalogReadGateway() = %#v, want nil", got)
				}
				return NewCatalogReadGateway(CatalogReadDeps{Campaign: &contractCampaignClient{}}, "https://cdn.example.test")
			},
		},
		{
			name: "catalog mutation",
			run: func() any {
				if got := NewCatalogMutationGateway(CatalogMutationDeps{}); got != nil {
					t.Fatalf("NewCatalogMutationGateway() = %#v, want nil", got)
				}
				return NewCatalogMutationGateway(CatalogMutationDeps{Campaign: &contractCampaignClient{}})
			},
		},
		{
			name: "workspace read",
			run: func() any {
				if got := NewWorkspaceReadGateway(WorkspaceReadDeps{}, "https://cdn.example.test"); got != nil {
					t.Fatalf("NewWorkspaceReadGateway() = %#v, want nil", got)
				}
				return NewWorkspaceReadGateway(WorkspaceReadDeps{Campaign: &contractCampaignClient{}}, "https://cdn.example.test")
			},
		},
		{
			name: "participant read",
			run: func() any {
				if got := NewParticipantReadGateway(ParticipantReadDeps{}, "https://cdn.example.test"); got != nil {
					t.Fatalf("NewParticipantReadGateway() = %#v, want nil", got)
				}
				return NewParticipantReadGateway(ParticipantReadDeps{Participant: &contractParticipantClient{}}, "https://cdn.example.test")
			},
		},
		{
			name: "participant mutation",
			run: func() any {
				if got := NewParticipantMutationGateway(ParticipantMutationDeps{}); got != nil {
					t.Fatalf("NewParticipantMutationGateway() = %#v, want nil", got)
				}
				return NewParticipantMutationGateway(ParticipantMutationDeps{Participant: &contractParticipantClient{}})
			},
		},
		{
			name: "character read",
			run: func() any {
				if got := NewCharacterReadGateway(CharacterReadDeps{}, "https://cdn.example.test"); got != nil {
					t.Fatalf("NewCharacterReadGateway() = %#v, want nil", got)
				}
				return NewCharacterReadGateway(CharacterReadDeps{
					Character:          &fakeCharacterWorkflowClient{},
					Participant:        &contractParticipantClient{},
					DaggerheartContent: &fakeDaggerheartContentClient{},
				}, "https://cdn.example.test")
			},
		},
		{
			name: "character mutation",
			run: func() any {
				if got := NewCharacterMutationGateway(CharacterMutationDeps{}); got != nil {
					t.Fatalf("NewCharacterMutationGateway() = %#v, want nil", got)
				}
				return NewCharacterMutationGateway(CharacterMutationDeps{Character: &fakeCharacterWorkflowClient{}})
			},
		},
		{
			name: "character ownership mutation",
			run: func() any {
				if got := NewCharacterOwnershipMutationGateway(CharacterOwnershipMutationDeps{}); got != nil {
					t.Fatalf("NewCharacterOwnershipMutationGateway() = %#v, want nil", got)
				}
				return NewCharacterOwnershipMutationGateway(CharacterOwnershipMutationDeps{Character: &fakeCharacterWorkflowClient{}})
			},
		},
		{
			name: "automation read",
			run: func() any {
				if got := NewAutomationReadGateway(AutomationReadDeps{}); got != nil {
					t.Fatalf("NewAutomationReadGateway() = %#v, want nil", got)
				}
				return NewAutomationReadGateway(AutomationReadDeps{Agent: &contractAgentClient{}})
			},
		},
		{
			name: "automation mutation",
			run: func() any {
				if got := NewAutomationMutationGateway(AutomationMutationDeps{}); got != nil {
					t.Fatalf("NewAutomationMutationGateway() = %#v, want nil", got)
				}
				return NewAutomationMutationGateway(AutomationMutationDeps{Campaign: &contractCampaignClient{}})
			},
		},
		{
			name: "session read",
			run: func() any {
				if got := NewSessionReadGateway(SessionReadDeps{}); got != nil {
					t.Fatalf("NewSessionReadGateway() = %#v, want nil", got)
				}
				return NewSessionReadGateway(SessionReadDeps{Session: &contractSessionClient{}, Campaign: &contractCampaignClient{}})
			},
		},
		{
			name: "session mutation",
			run: func() any {
				if got := NewSessionMutationGateway(SessionMutationDeps{}); got != nil {
					t.Fatalf("NewSessionMutationGateway() = %#v, want nil", got)
				}
				return NewSessionMutationGateway(SessionMutationDeps{Session: &contractSessionClient{}})
			},
		},
		{
			name: "invite read",
			run: func() any {
				if got := NewInviteReadGateway(InviteReadDeps{}); got != nil {
					t.Fatalf("NewInviteReadGateway() = %#v, want nil", got)
				}
				return NewInviteReadGateway(InviteReadDeps{
					Invite:      &contractInviteClient{},
					Participant: &contractParticipantClient{},
					Social:      &contractSocialClient{},
					Auth:        &contractAuthClient{},
				})
			},
		},
		{
			name: "invite mutation",
			run: func() any {
				if got := NewInviteMutationGateway(InviteMutationDeps{}); got != nil {
					t.Fatalf("NewInviteMutationGateway() = %#v, want nil", got)
				}
				return NewInviteMutationGateway(InviteMutationDeps{Invite: &contractInviteClient{}, Auth: &contractAuthClient{}})
			},
		},
		{
			name: "configuration mutation",
			run: func() any {
				if got := NewConfigurationMutationGateway(ConfigurationMutationDeps{}); got != nil {
					t.Fatalf("NewConfigurationMutationGateway() = %#v, want nil", got)
				}
				return NewConfigurationMutationGateway(ConfigurationMutationDeps{Campaign: &contractCampaignClient{}})
			},
		},
		{
			name: "creation read",
			run: func() any {
				if got := NewCharacterCreationReadGateway(CharacterCreationReadDeps{}, "https://cdn.example.test"); got != nil {
					t.Fatalf("NewCharacterCreationReadGateway() = %#v, want nil", got)
				}
				return NewCharacterCreationReadGateway(CharacterCreationReadDeps{
					Character:          &fakeCharacterWorkflowClient{},
					DaggerheartContent: &fakeDaggerheartContentClient{},
					DaggerheartAsset:   &fakeDaggerheartContentClient{},
				}, "https://cdn.example.test")
			},
		},
		{
			name: "creation mutation",
			run: func() any {
				if got := NewCharacterCreationMutationGateway(CharacterCreationMutationDeps{}); got != nil {
					t.Fatalf("NewCharacterCreationMutationGateway() = %#v, want nil", got)
				}
				return NewCharacterCreationMutationGateway(CharacterCreationMutationDeps{Character: &fakeCharacterWorkflowClient{}})
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.run(); got == nil {
				t.Fatalf("%s returned nil for ready deps", tc.name)
			}
		})
	}
}

func TestMapCharacterCreationStepsCoverRemainingOneofBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		step  *campaignapp.CampaignCharacterCreationStepInput
		check func(t *testing.T, mapped *daggerheartv1.DaggerheartCreationStepInput)
	}{
		{
			name: "heritage",
			step: &campaignapp.CampaignCharacterCreationStepInput{
				Heritage: &campaignapp.CampaignCharacterCreationStepHeritage{
					Heritage: campaignapp.CampaignCharacterCreationHeritageSelection{
						AncestryLabel:           "  Loreborne  ",
						FirstFeatureAncestryID:  " ancestry-1 ",
						SecondFeatureAncestryID: " ancestry-2 ",
						CommunityID:             " community-1 ",
					},
				},
			},
			check: func(t *testing.T, mapped *daggerheartv1.DaggerheartCreationStepInput) {
				t.Helper()
				input := mapped.GetHeritageInput().GetHeritage()
				if input.GetAncestryLabel() != "Loreborne" || input.GetCommunityId() != "community-1" {
					t.Fatalf("heritage input = %#v", input)
				}
			},
		},
		{
			name: "traits",
			step: &campaignapp.CampaignCharacterCreationStepInput{
				Traits: &campaignapp.CampaignCharacterCreationStepTraits{Agility: 1, Strength: 2, Finesse: 3, Instinct: 4, Presence: 5, Knowledge: 6},
			},
			check: func(t *testing.T, mapped *daggerheartv1.DaggerheartCreationStepInput) {
				t.Helper()
				input := mapped.GetTraitsInput()
				if input.GetKnowledge() != 6 || input.GetPresence() != 5 {
					t.Fatalf("traits input = %#v", input)
				}
			},
		},
		{
			name: "background",
			step: &campaignapp.CampaignCharacterCreationStepInput{
				Background: &campaignapp.CampaignCharacterCreationStepBackground{Background: "  Outcast courier  "},
			},
			check: func(t *testing.T, mapped *daggerheartv1.DaggerheartCreationStepInput) {
				t.Helper()
				input := mapped.GetBackgroundInput()
				if input.GetBackground() != "Outcast courier" {
					t.Fatalf("background input = %#v", input)
				}
			},
		},
		{
			name: "experiences",
			step: &campaignapp.CampaignCharacterCreationStepInput{
				Experiences: &campaignapp.CampaignCharacterCreationStepExperiences{
					Experiences: []campaignapp.CampaignCharacterCreationStepExperience{
						{Name: "  Cartography  ", Modifier: 2},
						{Name: "   ", Modifier: 1},
					},
				},
			},
			check: func(t *testing.T, mapped *daggerheartv1.DaggerheartCreationStepInput) {
				t.Helper()
				input := mapped.GetExperiencesInput()
				if len(input.GetExperiences()) != 1 || input.GetExperiences()[0].GetName() != "Cartography" {
					t.Fatalf("experiences input = %#v", input)
				}
			},
		},
		{
			name: "domain cards",
			step: &campaignapp.CampaignCharacterCreationStepInput{
				DomainCards: &campaignapp.CampaignCharacterCreationStepDomainCards{DomainCardIDs: []string{" card-1 ", "", " card-2 "}},
			},
			check: func(t *testing.T, mapped *daggerheartv1.DaggerheartCreationStepInput) {
				t.Helper()
				input := mapped.GetDomainCardsInput()
				if got := input.GetDomainCardIds(); len(got) != 2 || got[0] != "card-1" || got[1] != "card-2" {
					t.Fatalf("domain cards input = %#v", input)
				}
			},
		},
		{
			name: "connections",
			step: &campaignapp.CampaignCharacterCreationStepInput{
				Connections: &campaignapp.CampaignCharacterCreationStepConnections{Connections: "  Owes the ranger a favor  "},
			},
			check: func(t *testing.T, mapped *daggerheartv1.DaggerheartCreationStepInput) {
				t.Helper()
				input := mapped.GetConnectionsInput()
				if input.GetConnections() != "Owes the ranger a favor" {
					t.Fatalf("connections input = %#v", input)
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mapped, err := MapCampaignCharacterCreationStepToProto(tc.step)
			if err != nil {
				t.Fatalf("MapCampaignCharacterCreationStepToProto() error = %v", err)
			}
			tc.check(t, mapped)
		})
	}
}
