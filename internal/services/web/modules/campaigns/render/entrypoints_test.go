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
			name: "invites",
			component: func() templ.Component {
				return InvitesFragment(InvitesPageView{
					CampaignDetailBaseView: CampaignDetailBaseView{
						CampaignID:       "camp-1",
						CanManageInvites: true,
					},
					InviteSeatOptions: []InviteSeatOptionView{{ParticipantID: "p-1", Label: "Rook"}},
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
