package sqlite

import (
	"database/sql"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
)

func TestMillisHelpers(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	value := time.Date(2026, 2, 1, 9, 0, 0, 0, loc)
	if toMillis(value) != value.UTC().UnixMilli() {
		t.Fatalf("expected millis to match UTC unix millis")
	}

	round := fromMillis(toMillis(value))
	if !round.Equal(value.UTC()) {
		t.Fatalf("expected round trip UTC time, got %v", round)
	}
}

func TestNullMillisHelpers(t *testing.T) {
	if got := toNullMillis(nil); got.Valid {
		t.Fatal("expected nil time to produce invalid NullInt64")
	}
	if got := fromNullMillis(sql.NullInt64{}); got != nil {
		t.Fatal("expected invalid NullInt64 to return nil time")
	}

	value := time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC)
	wrapped := toNullMillis(&value)
	if !wrapped.Valid {
		t.Fatal("expected valid null millis")
	}
	unwrapped := fromNullMillis(wrapped)
	if unwrapped == nil || !unwrapped.Equal(value) {
		t.Fatalf("expected round trip time, got %v", unwrapped)
	}
}

func TestExtractUpMigration(t *testing.T) {
	content := "-- +migrate Up\nCREATE TABLE test(id text);\n-- +migrate Down\nDROP TABLE test;"
	up := extractUpMigration(content)
	if up == "" || up == content {
		t.Fatal("expected up migration subset")
	}

	plain := "CREATE TABLE test(id text);"
	if extractUpMigration(plain) != plain {
		t.Fatal("expected full content when no markers present")
	}
}

func TestIsAlreadyExistsError(t *testing.T) {
	if isAlreadyExistsError(sql.ErrNoRows) {
		t.Fatal("unexpected already exists match")
	}
	if !isAlreadyExistsError(fakeErr("table already exists")) {
		t.Fatal("expected already exists match")
	}
}

func TestConversionHelpers(t *testing.T) {
	if gameSystemToString(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART) != "DAGGERHEART" {
		t.Fatal("expected daggerheart game system string")
	}
	if gameSystemToString(commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED) != "UNSPECIFIED" {
		t.Fatal("expected unspecified game system string")
	}
	if stringToGameSystem("unknown") != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatal("expected fallback game system daggerheart")
	}

	if campaignStatusToString(campaign.CampaignStatusArchived) != "ARCHIVED" {
		t.Fatal("expected archived status string")
	}
	if stringToCampaignStatus("ACTIVE") != campaign.CampaignStatusActive {
		t.Fatal("expected active status")
	}
	if stringToCampaignStatus("bogus") != campaign.CampaignStatusUnspecified {
		t.Fatal("expected unspecified status fallback")
	}

	if gmModeToString(campaign.GmModeHybrid) != "HYBRID" {
		t.Fatal("expected hybrid gm mode")
	}
	if stringToGmMode("AI") != campaign.GmModeAI {
		t.Fatal("expected AI gm mode")
	}
	if stringToGmMode("bogus") != campaign.GmModeUnspecified {
		t.Fatal("expected unspecified gm mode")
	}

	if participantRoleToString(participant.ParticipantRoleGM) != "GM" {
		t.Fatal("expected GM role")
	}
	if stringToParticipantRole("PLAYER") != participant.ParticipantRolePlayer {
		t.Fatal("expected player role")
	}
	if stringToParticipantRole("bogus") != participant.ParticipantRoleUnspecified {
		t.Fatal("expected unspecified role")
	}

	if participantControllerToString(participant.ControllerAI) != "AI" {
		t.Fatal("expected AI controller")
	}
	if stringToParticipantController("HUMAN") != participant.ControllerHuman {
		t.Fatal("expected human controller")
	}
	if stringToParticipantController("bogus") != participant.ControllerUnspecified {
		t.Fatal("expected unspecified controller")
	}

	if participantAccessToString(participant.CampaignAccessOwner) != "OWNER" {
		t.Fatal("expected owner access")
	}
	if stringToParticipantAccess("MEMBER") != participant.CampaignAccessMember {
		t.Fatal("expected member access")
	}
	if stringToParticipantAccess("bogus") != participant.CampaignAccessUnspecified {
		t.Fatal("expected unspecified access")
	}

	if characterKindToString(character.CharacterKindNPC) != "NPC" {
		t.Fatal("expected NPC kind")
	}
	if stringToCharacterKind("PC") != character.CharacterKindPC {
		t.Fatal("expected PC kind")
	}
	if stringToCharacterKind("bogus") != character.CharacterKindUnspecified {
		t.Fatal("expected unspecified kind")
	}

	if sessionStatusToString(session.SessionStatusEnded) != "ENDED" {
		t.Fatal("expected ended session status")
	}
	if stringToSessionStatus("ACTIVE") != session.SessionStatusActive {
		t.Fatal("expected active session status")
	}
	if stringToSessionStatus("bogus") != session.SessionStatusUnspecified {
		t.Fatal("expected unspecified session status")
	}
}

type fakeErr string

func (f fakeErr) Error() string {
	return string(f)
}
