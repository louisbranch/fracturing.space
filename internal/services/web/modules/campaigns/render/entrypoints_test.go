package render

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func TestExportedSectionFragmentsRenderOwnedMarkers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		component func() templ.Component
		marker    string
	}{
		{
			name: "overview",
			component: func() templ.Component {
				return OverviewFragment(OverviewPageView{
					CampaignDetailBaseView: CampaignDetailBaseView{
						CampaignID:      "camp-1",
						Name:            "Skyline",
						System:          "daggerheart",
						GMMode:          "human",
						Status:          "active",
						Locale:          "en-US",
						Intent:          "standard",
						AccessPolicy:    "private",
						Theme:           "Storm over the city",
						CanEditCampaign: true,
					},
					AIBindingStatus: "human",
				}, nil)
			},
			marker: `data-campaign-overview-name="Skyline"`,
		},
		{
			name: "participants",
			component: func() templ.Component {
				return ParticipantsFragment(ParticipantsPageView{
					CampaignDetailBaseView: CampaignDetailBaseView{
						CampaignID:            "camp-1",
						CanManageParticipants: true,
					},
					Participants: []ParticipantView{{
						ID:             "p-1",
						Name:           "Rook",
						Role:           "player",
						CampaignAccess: "full",
						Controller:     "human",
						CanEdit:        true,
					}},
				}, nil)
			},
			marker: `data-campaign-participant-card-id="p-1"`,
		},
		{
			name: "sessions",
			component: func() templ.Component {
				return SessionsFragment(SessionsPageView{
					CampaignDetailBaseView: CampaignDetailBaseView{CampaignID: "camp-1"},
					Sessions: []SessionView{{
						ID:     "s-1",
						Name:   "First Session",
						Status: "active",
					}},
				}, nil)
			},
			marker: `data-campaign-session-card-id="s-1"`,
		},
		{
			name: "session detail",
			component: func() templ.Component {
				return SessionDetailFragment(SessionDetailPageView{
					CampaignDetailBaseView: CampaignDetailBaseView{CampaignID: "camp-1"},
					SessionID:              "s-1",
					Sessions: []SessionView{{
						ID:        "s-1",
						Name:      "First Session",
						Status:    "active",
						StartedAt: "2026-03-21 10:00 UTC",
					}},
				}, nil)
			},
			marker: `data-campaign-session-detail-id="s-1"`,
		},
		{
			name: "session create",
			component: func() templ.Component {
				return SessionCreateFragment(SessionCreatePageView{
					CampaignDetailBaseView: CampaignDetailBaseView{CampaignID: "camp-1"},
					SessionReadiness:       SessionReadinessView{Ready: true},
				}, nil)
			},
			marker: `data-campaign-session-create-page="true"`,
		},
		{
			name: "campaign edit",
			component: func() templ.Component {
				return CampaignEditFragment(CampaignEditPageView{
					CampaignDetailBaseView: CampaignDetailBaseView{
						CampaignID:      "camp-1",
						Name:            "Skyline",
						System:          "daggerheart",
						GMMode:          "human",
						Status:          "active",
						LocaleValue:     "en-US",
						Intent:          "standard",
						AccessPolicy:    "private",
						Theme:           "Storm over the city",
						CanEditCampaign: true,
					},
				}, nil)
			},
			marker: `action="/app/campaigns/camp-1/edit"`,
		},
		{
			name: "campaign ai binding",
			component: func() templ.Component {
				return CampaignAIBindingFragment(CampaignAIBindingPageView{
					CampaignDetailBaseView: CampaignDetailBaseView{CampaignID: "camp-1"},
					AIBindingSettings: AIBindingSettingsView{
						Options: []AIAgentOptionView{{ID: "agent-1", Name: "Narrator", Selected: true}},
					},
				}, nil)
			},
			marker: `data-campaign-ai-binding-page="true"`,
		},
		{
			name: "invites",
			component: func() templ.Component {
				return InvitesFragment(InvitesPageView{
					CampaignDetailBaseView: CampaignDetailBaseView{
						CampaignID:       "camp-1",
						CanManageInvites: true,
					},
					Invites: []InviteView{{
						ID:              "invite-1",
						ParticipantID:   "p-1",
						ParticipantName: "Rook",
						Status:          "pending",
					}},
				}, nil)
			},
			marker: `data-campaign-invite-card-id="invite-1"`,
		},
		{
			name: "invite create",
			component: func() templ.Component {
				return InviteCreateFragment(InviteCreatePageView{
					CampaignDetailBaseView: CampaignDetailBaseView{CampaignID: "camp-1"},
					InviteSeatOptions:      []InviteSeatOptionView{{ParticipantID: "p-1", Label: "Rook"}},
				}, nil)
			},
			marker: `data-campaign-invite-create-page="true"`,
		},
		{
			name: "participant create",
			component: func() templ.Component {
				return ParticipantCreateFragment(ParticipantCreatePageView{
					CampaignDetailBaseView: CampaignDetailBaseView{CampaignID: "camp-1"},
					ParticipantCreator: ParticipantCreatorView{
						Name: "Rook",
					},
				}, nil)
			},
			marker: `data-campaign-participant-create-page="true"`,
		},
		{
			name: "participant edit",
			component: func() templ.Component {
				return ParticipantEditFragment(ParticipantEditPageView{
					CampaignDetailBaseView: CampaignDetailBaseView{CampaignID: "camp-1"},
					ParticipantID:          "p-1",
					ParticipantEditor: ParticipantEditorView{
						ID:   "p-1",
						Name: "Rook",
						Delete: ParticipantDeleteView{
							Visible: true,
							Enabled: true,
						},
					},
				}, nil)
			},
			marker: `data-campaign-participant-edit-page="true"`,
		},
		{
			name: "character list",
			component: func() templ.Component {
				return CharactersFragment(CharactersPageView{
					CampaignDetailBaseView: CampaignDetailBaseView{
						CampaignID:         "camp-1",
						CanCreateCharacter: true,
					},
					Characters: []CharacterView{{
						ID:   "char-1",
						Name: "Mira",
						Kind: "pc",
					}},
				}, nil)
			},
			marker: `data-campaign-character-card-id="char-1"`,
		},
		{
			name: "character create",
			component: func() templ.Component {
				return CharacterCreateFragment(CharacterCreatePageView{
					CampaignDetailBaseView: CampaignDetailBaseView{CampaignID: "camp-1"},
					CharacterEditor:        CharacterEditorView{Kind: "PC"},
				}, nil)
			},
			marker: `data-campaign-character-create-page="true"`,
		},
		{
			name: "character edit",
			component: func() templ.Component {
				return CharacterEditFragment(CharacterEditPageView{
					CampaignDetailBaseView: CampaignDetailBaseView{CampaignID: "camp-1"},
					CharacterID:            "char-1",
					Character:              CharacterView{ID: "char-1", Name: "Mira"},
					CharacterEditor:        CharacterEditorView{ID: "char-1", Name: "Mira", Kind: "PC"},
				}, nil)
			},
			marker: `data-campaign-character-edit-page="true"`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			if err := tt.component().Render(context.Background(), &buf); err != nil {
				t.Fatalf("Render() error = %v", err)
			}
			if got := buf.String(); !strings.Contains(got, tt.marker) {
				t.Fatalf("output missing marker %q: %s", tt.marker, got)
			}
		})
	}
}
