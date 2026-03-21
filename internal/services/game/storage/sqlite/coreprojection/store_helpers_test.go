package coreprojection

import (
	"database/sql"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
)

func TestMillisHelpers(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	value := time.Date(2026, 2, 1, 9, 0, 0, 0, loc)
	if sqliteutil.ToMillis(value) != value.UTC().UnixMilli() {
		t.Fatalf("expected millis to match UTC unix millis")
	}

	round := sqliteutil.FromMillis(sqliteutil.ToMillis(value))
	if !round.Equal(value.UTC()) {
		t.Fatalf("expected round trip UTC time, got %v", round)
	}
}

func TestNullMillisHelpers(t *testing.T) {
	if got := sqliteutil.ToNullMillis(nil); got.Valid {
		t.Fatal("expected nil time to produce invalid NullInt64")
	}
	if got := sqliteutil.FromNullMillis(sql.NullInt64{}); got != nil {
		t.Fatal("expected invalid NullInt64 to return nil time")
	}

	value := time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC)
	wrapped := sqliteutil.ToNullMillis(&value)
	if !wrapped.Valid {
		t.Fatal("expected valid null millis")
	}
	unwrapped := sqliteutil.FromNullMillis(wrapped)
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

func TestEnumToStorage(t *testing.T) {
	tests := []struct {
		name string
		val  string
		want string
	}{
		{"known value", string(bridge.SystemIDDaggerheart), "DAGGERHEART"},
		{"empty value", "", "UNSPECIFIED"},
		{"lowercase value", string(campaign.StatusActive), "ACTIVE"},
		{"mixed case", string(participant.RoleGM), "GM"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use bridge.SystemID as a representative ~string type for the first two,
			// and raw string conversion for the rest.
			got := enumToStorage(bridge.SystemID(tt.val))
			if got != tt.want {
				t.Fatalf("enumToStorage(%q) = %q, want %q", tt.val, got, tt.want)
			}
		})
	}
}

func TestEnumFromStorage(t *testing.T) {
	// Round-trip: known values
	if got := enumFromStorage("DAGGERHEART", bridge.NormalizeSystemID); got != bridge.SystemIDDaggerheart {
		t.Fatalf("expected daggerheart, got %q", got)
	}
	if got := enumFromStorage("ACTIVE", campaign.NormalizeStatus); got != campaign.StatusActive {
		t.Fatalf("expected active, got %q", got)
	}
	if got := enumFromStorage("GM", participant.NormalizeRole); got != participant.RoleGM {
		t.Fatalf("expected gm, got %q", got)
	}
	if got := enumFromStorage("PC", character.NormalizeKind); got != character.KindPC {
		t.Fatalf("expected pc, got %q", got)
	}
	if got := enumFromStorage("ENDED", session.NormalizeStatus); got != session.StatusEnded {
		t.Fatalf("expected ended, got %q", got)
	}

	// Unknown values fall back to zero value
	if got := enumFromStorage("bogus", bridge.NormalizeSystemID); got != bridge.SystemIDUnspecified {
		t.Fatalf("expected unspecified, got %q", got)
	}
	if got := enumFromStorage("bogus", campaign.NormalizeStatus); got != campaign.StatusUnspecified {
		t.Fatalf("expected unspecified, got %q", got)
	}
}

func TestBoolIntHelpers(t *testing.T) {
	if boolToInt(true) != 1 {
		t.Fatal("expected boolToInt(true) = 1")
	}
	if boolToInt(false) != 0 {
		t.Fatal("expected boolToInt(false) = 0")
	}
	if !intToBool(1) {
		t.Fatal("expected intToBool(1) = true")
	}
	if intToBool(0) {
		t.Fatal("expected intToBool(0) = false")
	}
}

func TestToNullString(t *testing.T) {
	if got := sqliteutil.ToNullString(""); got.Valid {
		t.Fatal("expected empty string to produce invalid NullString")
	}
	if got := sqliteutil.ToNullString("  "); got.Valid {
		t.Fatal("expected whitespace-only string to produce invalid NullString")
	}
	got := sqliteutil.ToNullString("hello")
	if !got.Valid || got.String != "hello" {
		t.Fatalf("expected valid NullString with value 'hello', got valid=%v string=%q", got.Valid, got.String)
	}
}

type fakeErr string

func (f fakeErr) Error() string {
	return string(f)
}
