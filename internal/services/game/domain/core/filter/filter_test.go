package filter

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseEventFilter_TypeEquals(t *testing.T) {
	cond, err := ParseEventFilter(`type = "session.started"`)
	if err != nil {
		t.Fatalf("parse filter: %v", err)
	}
	if cond.Clause != "event_type = ?" {
		t.Errorf("expected 'event_type = ?', got %q", cond.Clause)
	}
	if len(cond.Params) != 1 {
		t.Fatalf("expected 1 param, got %d", len(cond.Params))
	}
	if cond.Params[0] != "session.started" {
		t.Errorf("expected 'session.started', got %v", cond.Params[0])
	}
}

func TestParseEventFilter_Empty(t *testing.T) {
	cond, err := ParseEventFilter(" ")
	if err != nil {
		t.Fatalf("parse filter: %v", err)
	}
	if cond.Clause != "" || cond.Params != nil {
		t.Fatalf("expected empty condition, got %+v", cond)
	}
}

func TestParseEventFilter_AndOr(t *testing.T) {
	cond, err := ParseEventFilter(`type = "session.started" AND actor_type = "gm"`)
	if err != nil {
		t.Fatalf("parse filter: %v", err)
	}
	if cond.Clause != "(event_type = ? AND actor_type = ?)" {
		t.Fatalf("Clause = %q", cond.Clause)
	}
	if !reflect.DeepEqual(cond.Params, []any{"session.started", "gm"}) {
		t.Fatalf("Params = %v", cond.Params)
	}

	cond, err = ParseEventFilter(`actor_type = "gm" OR actor_type = "participant"`)
	if err != nil {
		t.Fatalf("parse filter: %v", err)
	}
	if cond.Clause != "(actor_type = ? OR actor_type = ?)" {
		t.Fatalf("Clause = %q", cond.Clause)
	}
}

func TestParseEventFilter_NotEqualsAndNumeric(t *testing.T) {
	cond, err := ParseEventFilter(`actor_id != "p1"`)
	if err != nil {
		t.Fatalf("parse filter: %v", err)
	}
	if cond.Clause != "actor_id != ?" {
		t.Fatalf("Clause = %q", cond.Clause)
	}

	cond, err = ParseEventFilter(`ts > timestamp("2025-01-01T00:00:00Z")`)
	if err != nil {
		t.Fatalf("parse filter: %v", err)
	}
	if cond.Clause != "timestamp > ?" {
		t.Fatalf("Clause = %q", cond.Clause)
	}
	if len(cond.Params) != 1 {
		t.Fatalf("Params len = %d", len(cond.Params))
	}
	if !strings.HasPrefix(cond.Params[0].(string), "2025-01-01T00:00:00") {
		t.Fatalf("timestamp param = %v", cond.Params[0])
	}
}

func TestParseEventFilter_InvalidField(t *testing.T) {
	_, err := ParseEventFilter(`unknown = "x"`)
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestParseEventFilter_InvalidValueFunc(t *testing.T) {
	_, err := ParseEventFilter(`ts = duration("1h")`)
	if err == nil {
		t.Fatal("expected error for unsupported value function")
	}
}

func TestParseEventFilter_InvalidTimestamp(t *testing.T) {
	_, err := ParseEventFilter(`ts = timestamp("not-a-time")`)
	if err == nil {
		t.Fatal("expected error for invalid timestamp")
	}
}
