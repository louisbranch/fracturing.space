package filter

import (
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
