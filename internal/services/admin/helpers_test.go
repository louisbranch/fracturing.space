package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/i18n"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestFormatGmMode(t *testing.T) {
	loc := i18n.Printer(i18n.Default())
	tests := []struct {
		mode statev1.GmMode
		want string
	}{
		{statev1.GmMode_HUMAN, loc.Sprintf("label.human")},
		{statev1.GmMode_AI, loc.Sprintf("label.ai")},
		{statev1.GmMode_HYBRID, loc.Sprintf("label.hybrid")},
		{statev1.GmMode_GM_MODE_UNSPECIFIED, loc.Sprintf("label.unspecified")},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			if got := formatGmMode(tc.mode, loc); got != tc.want {
				t.Errorf("formatGmMode(%v) = %q, want %q", tc.mode, got, tc.want)
			}
		})
	}
}

func TestFormatGameSystem(t *testing.T) {
	loc := i18n.Printer(i18n.Default())
	tests := []struct {
		system commonv1.GameSystem
		want   string
	}{
		{commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, loc.Sprintf("label.daggerheart")},
		{commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, loc.Sprintf("label.unspecified")},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			if got := formatGameSystem(tc.system, loc); got != tc.want {
				t.Errorf("formatGameSystem(%v) = %q, want %q", tc.system, got, tc.want)
			}
		})
	}
}

func TestParseSystemID(t *testing.T) {
	tests := []struct {
		input string
		want  commonv1.GameSystem
	}{
		{"", commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED},
		{"daggerheart", commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART},
		{"DAGGERHEART", commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART},
		{" Daggerheart ", commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART},
		{"GAME_SYSTEM_DAGGERHEART", commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART},
		{"unknown", commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			if got := parseSystemID(tc.input); got != tc.want {
				t.Errorf("parseSystemID(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestFormatSessionStatus(t *testing.T) {
	loc := i18n.Printer(i18n.Default())
	tests := []struct {
		status statev1.SessionStatus
		want   string
	}{
		{statev1.SessionStatus_SESSION_ACTIVE, loc.Sprintf("label.active")},
		{statev1.SessionStatus_SESSION_ENDED, loc.Sprintf("label.ended")},
		{statev1.SessionStatus_SESSION_STATUS_UNSPECIFIED, loc.Sprintf("label.unspecified")},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			if got := formatSessionStatus(tc.status, loc); got != tc.want {
				t.Errorf("formatSessionStatus(%v) = %q, want %q", tc.status, got, tc.want)
			}
		})
	}
}

func TestFormatInviteStatus(t *testing.T) {
	loc := i18n.Printer(i18n.Default())
	tests := []struct {
		status  statev1.InviteStatus
		want    string
		variant string
	}{
		{statev1.InviteStatus_PENDING, loc.Sprintf("label.invite_pending"), "warning"},
		{statev1.InviteStatus_CLAIMED, loc.Sprintf("label.invite_claimed"), "success"},
		{statev1.InviteStatus_REVOKED, loc.Sprintf("label.invite_revoked"), "error"},
		{statev1.InviteStatus_INVITE_STATUS_UNSPECIFIED, loc.Sprintf("label.unspecified"), "secondary"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			label, variant := formatInviteStatus(tc.status, loc)
			if label != tc.want {
				t.Errorf("label = %q, want %q", label, tc.want)
			}
			if variant != tc.variant {
				t.Errorf("variant = %q, want %q", variant, tc.variant)
			}
		})
	}
}

func TestFormatCreatedDate(t *testing.T) {
	t.Run("nil timestamp", func(t *testing.T) {
		if got := formatCreatedDate(nil); got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("valid timestamp", func(t *testing.T) {
		ts := timestamppb.Now()
		result := formatCreatedDate(ts)
		if result == "" {
			t.Error("expected non-empty date string")
		}
		// Format should be YYYY-MM-DD
		if len(result) != 10 {
			t.Errorf("expected 10-char date, got %q", result)
		}
	})
}

func TestFormatTimestamp(t *testing.T) {
	t.Run("nil timestamp", func(t *testing.T) {
		if got := formatTimestamp(nil); got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("valid timestamp", func(t *testing.T) {
		ts := timestamppb.Now()
		result := formatTimestamp(ts)
		if result == "" {
			t.Error("expected non-empty timestamp string")
		}
	})
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		limit int
		want  string
	}{
		{"empty text", "", 10, ""},
		{"zero limit", "hello", 0, ""},
		{"under limit", "hi", 10, "hi"},
		{"at limit", "hello", 5, "hello"},
		{"over limit", "hello world", 5, "hello..."},
		{"unicode", "日本語テスト", 3, "日本語..."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := truncateText(tc.text, tc.limit); got != tc.want {
				t.Errorf("truncateText(%q, %d) = %q, want %q", tc.text, tc.limit, got, tc.want)
			}
		})
	}
}

func TestFormatParticipantRole(t *testing.T) {
	loc := i18n.Printer(i18n.Default())
	tests := []struct {
		role    statev1.ParticipantRole
		want    string
		variant string
	}{
		{statev1.ParticipantRole_GM, loc.Sprintf("label.gm"), "info"},
		{statev1.ParticipantRole_PLAYER, loc.Sprintf("label.player"), "success"},
		{statev1.ParticipantRole_ROLE_UNSPECIFIED, loc.Sprintf("label.unspecified"), "secondary"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			label, variant := formatParticipantRole(tc.role, loc)
			if label != tc.want {
				t.Errorf("label = %q, want %q", label, tc.want)
			}
			if variant != tc.variant {
				t.Errorf("variant = %q, want %q", variant, tc.variant)
			}
		})
	}
}

func TestFormatParticipantController(t *testing.T) {
	loc := i18n.Printer(i18n.Default())
	tests := []struct {
		controller statev1.Controller
		want       string
		variant    string
	}{
		{statev1.Controller_CONTROLLER_HUMAN, loc.Sprintf("label.human"), "success"},
		{statev1.Controller_CONTROLLER_AI, loc.Sprintf("label.ai"), "info"},
		{statev1.Controller_CONTROLLER_UNSPECIFIED, loc.Sprintf("label.unspecified"), "secondary"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			label, variant := formatParticipantController(tc.controller, loc)
			if label != tc.want {
				t.Errorf("label = %q, want %q", label, tc.want)
			}
			if variant != tc.variant {
				t.Errorf("variant = %q, want %q", variant, tc.variant)
			}
		})
	}
}

func TestFormatParticipantAccess(t *testing.T) {
	loc := i18n.Printer(i18n.Default())
	tests := []struct {
		access  statev1.CampaignAccess
		want    string
		variant string
	}{
		{statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER, loc.Sprintf("label.member"), "secondary"},
		{statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER, loc.Sprintf("label.manager"), "info"},
		{statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER, loc.Sprintf("label.owner"), "warning"},
		{statev1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED, loc.Sprintf("label.unspecified"), "secondary"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			label, variant := formatParticipantAccess(tc.access, loc)
			if label != tc.want {
				t.Errorf("label = %q, want %q", label, tc.want)
			}
			if variant != tc.variant {
				t.Errorf("variant = %q, want %q", variant, tc.variant)
			}
		})
	}
}

func TestFormatCharacterKind(t *testing.T) {
	loc := i18n.Printer(i18n.Default())
	tests := []struct {
		kind statev1.CharacterKind
		want string
	}{
		{statev1.CharacterKind_PC, loc.Sprintf("label.pc")},
		{statev1.CharacterKind_NPC, loc.Sprintf("label.npc")},
		{statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED, loc.Sprintf("label.unspecified")},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			if got := formatCharacterKind(tc.kind, loc); got != tc.want {
				t.Errorf("formatCharacterKind(%v) = %q, want %q", tc.kind, got, tc.want)
			}
		})
	}
}

func TestUsersPageRoute(t *testing.T) {
	handler := NewHandler(nil)

	t.Run("full page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/users", nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
		assertContains(t, recorder.Body.String(), "Users")
	})

	t.Run("htmx", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/users", nil)
		req.Header.Set("HX-Request", "true")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
		assertHTMXFragmentInvariant(t, recorder.Body.String())
	})
}

func TestUsersTableRoute(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/users/table", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	// Should show service unavailable error when no client
	assertContains(t, recorder.Body.String(), "unavailable")
}

func TestCampaignsTableRoute(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/table", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	assertContains(t, recorder.Body.String(), "unavailable")
}

func TestCampaignsTableWithClient(t *testing.T) {
	campaignClient := &testCampaignClient{}
	provider := testClientProvider{campaign: campaignClient}
	handler := NewHandler(provider)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/table", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestSystemDetailRoute(t *testing.T) {
	systemClient := &testSystemClient{
		getResponse: &statev1.GetGameSystemResponse{
			System: &statev1.GameSystemInfo{
				Id:                  commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
				Name:                "Daggerheart",
				Version:             "1.0.0",
				ImplementationStage: commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PARTIAL,
				OperationalStatus:   commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL,
				AccessLevel:         commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_BETA,
			},
		},
	}
	handler := NewHandler(testClientProvider{system: systemClient})

	t.Run("full page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/systems/GAME_SYSTEM_DAGGERHEART", nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
		assertContains(t, recorder.Body.String(), "Daggerheart")
	})

	t.Run("htmx", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/systems/GAME_SYSTEM_DAGGERHEART", nil)
		req.Header.Set("HX-Request", "true")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
		assertHTMXFragmentInvariant(t, recorder.Body.String())
	})
}

func TestSystemDetailNoClient(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/systems/GAME_SYSTEM_DAGGERHEART", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	assertContains(t, recorder.Body.String(), "unavailable")
}

func TestParticipantsListRoute(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-123/participants", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestParticipantsTableRoute(t *testing.T) {
	t.Run("no client", func(t *testing.T) {
		handler := NewHandler(nil)
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-123/participants/table", nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
		assertContains(t, recorder.Body.String(), "unavailable")
	})

	t.Run("with client", func(t *testing.T) {
		participantClient := &testParticipantClient{
			participants: []*statev1.Participant{
				{Id: "p1", CampaignId: "camp-123", Name: "Alice", Role: statev1.ParticipantRole_GM},
			},
		}
		handler := NewHandler(testClientProvider{participant: participantClient})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-123/participants/table", nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
		assertContains(t, recorder.Body.String(), "Alice")
	})
}

func TestCharactersListRoute(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-123/characters", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestInvitesListRoute(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-123/invites", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestDashboardRoute(t *testing.T) {
	handler := NewHandler(nil)

	t.Run("not found for unknown path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/unknown-path", nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", recorder.Code)
		}
	})

	t.Run("dashboard content", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/dashboard/content", nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", recorder.Code)
		}
	})
}

func TestEventLogRoute(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-123/events", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestEventLogTableRoute(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-123/events/table", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	assertContains(t, recorder.Body.String(), "unavailable")
}

func TestFormatEventDescription(t *testing.T) {
	loc := i18n.Printer(i18n.Default())

	t.Run("nil event", func(t *testing.T) {
		if got := formatEventDescription(nil, loc); got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("campaign created", func(t *testing.T) {
		event := &statev1.Event{Type: "campaign.created"}
		result := formatEventDescription(event, loc)
		// Just ensure it doesn't panic and returns something
		_ = result
	})

	t.Run("session started", func(t *testing.T) {
		event := &statev1.Event{Type: "session.started", SessionId: "s1"}
		result := formatEventDescription(event, loc)
		_ = result
	})

	t.Run("unknown type", func(t *testing.T) {
		event := &statev1.Event{Type: "unknown.event"}
		result := formatEventDescription(event, loc)
		_ = result
	})
}
