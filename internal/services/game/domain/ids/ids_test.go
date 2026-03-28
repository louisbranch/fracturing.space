package ids

import "testing"

func TestIsZero(t *testing.T) {
	// Each ID type shares the same IsZero implementation, so we test them
	// all through a common table to catch any copy-paste drift.
	type zeroChecker interface {
		IsZero() bool
	}

	cases := []struct {
		name string
		id   zeroChecker
		want bool
	}{
		// CampaignID
		{"CampaignID empty", CampaignID(""), true},
		{"CampaignID spaces", CampaignID("  "), true},
		{"CampaignID tab", CampaignID("\t"), true},
		{"CampaignID valid", CampaignID("abc"), false},
		{"CampaignID padded", CampaignID(" abc "), false},

		// ParticipantID
		{"ParticipantID empty", ParticipantID(""), true},
		{"ParticipantID spaces", ParticipantID("  "), true},
		{"ParticipantID tab", ParticipantID("\t"), true},
		{"ParticipantID valid", ParticipantID("abc"), false},
		{"ParticipantID padded", ParticipantID(" abc "), false},

		// CharacterID
		{"CharacterID empty", CharacterID(""), true},
		{"CharacterID spaces", CharacterID("  "), true},
		{"CharacterID tab", CharacterID("\t"), true},
		{"CharacterID valid", CharacterID("abc"), false},
		{"CharacterID padded", CharacterID(" abc "), false},

		// SessionID
		{"SessionID empty", SessionID(""), true},
		{"SessionID spaces", SessionID("  "), true},
		{"SessionID tab", SessionID("\t"), true},
		{"SessionID valid", SessionID("abc"), false},
		{"SessionID padded", SessionID(" abc "), false},

		// SceneID
		{"SceneID empty", SceneID(""), true},
		{"SceneID spaces", SceneID("  "), true},
		{"SceneID tab", SceneID("\t"), true},
		{"SceneID valid", SceneID("abc"), false},
		{"SceneID padded", SceneID(" abc "), false},

		// InviteID
		{"InviteID empty", InviteID(""), true},
		{"InviteID spaces", InviteID("  "), true},
		{"InviteID tab", InviteID("\t"), true},
		{"InviteID valid", InviteID("abc"), false},
		{"InviteID padded", InviteID(" abc "), false},

		// UserID
		{"UserID empty", UserID(""), true},
		{"UserID spaces", UserID("  "), true},
		{"UserID tab", UserID("\t"), true},
		{"UserID valid", UserID("abc"), false},
		{"UserID padded", UserID(" abc "), false},

		// GateID
		{"GateID empty", GateID(""), true},
		{"GateID spaces", GateID("  "), true},
		{"GateID tab", GateID("\t"), true},
		{"GateID valid", GateID("abc"), false},
		{"GateID padded", GateID(" abc "), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.id.IsZero(); got != tc.want {
				t.Fatalf("IsZero() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestString(t *testing.T) {
	type stringer interface {
		String() string
	}

	cases := []struct {
		name string
		id   stringer
		want string
	}{
		{"CampaignID", CampaignID("camp-1"), "camp-1"},
		{"ParticipantID", ParticipantID("part-2"), "part-2"},
		{"CharacterID", CharacterID("char-3"), "char-3"},
		{"SessionID", SessionID("sess-4"), "sess-4"},
		{"SceneID", SceneID("scn-5"), "scn-5"},
		{"InviteID", InviteID("inv-6"), "inv-6"},
		{"UserID", UserID("usr-7"), "usr-7"},
		{"GateID", GateID("gate-8"), "gate-8"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.id.String(); got != tc.want {
				t.Fatalf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestErrCampaignIDRequired_IsNonNil(t *testing.T) {
	if ErrCampaignIDRequired == nil {
		t.Fatal("ErrCampaignIDRequired must not be nil")
	}
	if ErrCampaignIDRequired.Error() == "" {
		t.Fatal("ErrCampaignIDRequired must have a non-empty message")
	}
}
