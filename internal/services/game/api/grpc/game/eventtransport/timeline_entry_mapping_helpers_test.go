package eventtransport

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
)

func TestEventDomainFromType(t *testing.T) {
	tests := []struct {
		name    string
		evtType event.Type
		want    string
	}{
		{name: "empty", evtType: "", want: ""},
		{name: "trimmed", evtType: "  session.started  ", want: "session"},
		{name: "no dot", evtType: "session", want: "session"},
		{name: "system prefix", evtType: daggerheartpayload.EventTypeCharacterStatePatched, want: "sys"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := eventDomainFromType(tc.evtType); got != tc.want {
				t.Fatalf("eventDomainFromType(%q) = %q, want %q", string(tc.evtType), got, tc.want)
			}
		})
	}
}

func TestAppendIntChange(t *testing.T) {
	var fields []*campaignv1.ProjectionField

	appendIntChange(&fields, "HP", testIntPtr(3), testIntPtr(3))
	if len(fields) != 0 {
		t.Fatalf("expected no change for equal values, got %d", len(fields))
	}

	appendIntChange(&fields, "HP", testIntPtr(3), testIntPtr(5))
	if len(fields) != 1 || fields[0].GetValue() != "3 -> 5" {
		t.Fatalf("fields after before/after change = %#v, want single 3 -> 5", fields)
	}

	appendIntChange(&fields, "Armor", nil, testIntPtr(2))
	if len(fields) != 2 || fields[1].GetValue() != "= 2" {
		t.Fatalf("fields after add-only change = %#v, want second value '= 2'", fields)
	}

	appendIntChange(&fields, "Stress", testIntPtr(1), nil)
	if len(fields) != 2 {
		t.Fatalf("expected nil after to be ignored, fields = %#v", fields)
	}
}

func TestAppendStringChange(t *testing.T) {
	var fields []*campaignv1.ProjectionField

	appendStringChange(&fields, "Life State", testStringPtr("alive"), testStringPtr("alive"))
	if len(fields) != 0 {
		t.Fatalf("expected no change for equal strings, got %d", len(fields))
	}

	appendStringChange(&fields, "Life State", testStringPtr("alive"), testStringPtr("dying"))
	if len(fields) != 1 || fields[0].GetValue() != "alive -> dying" {
		t.Fatalf("fields after before/after string change = %#v, want single alive -> dying", fields)
	}

	appendStringChange(&fields, "Life State", testStringPtr("  "), testStringPtr(" stable "))
	if len(fields) != 2 || fields[1].GetValue() != "= stable" {
		t.Fatalf("fields after add-only string change = %#v, want second value '= stable'", fields)
	}

	appendStringChange(&fields, "Life State", testStringPtr("alive"), testStringPtr("   "))
	if len(fields) != 2 {
		t.Fatalf("expected blank after value to be ignored, fields = %#v", fields)
	}

	appendStringChange(&fields, "Life State", testStringPtr("alive"), nil)
	if len(fields) != 2 {
		t.Fatalf("expected nil after value to be ignored, fields = %#v", fields)
	}
}

func TestDaggerheartStateChangeFieldsInvalidPayload(t *testing.T) {
	if fields := daggerheartStateChangeFields([]byte("not-json")); fields != nil {
		t.Fatalf("fields = %#v, want nil", fields)
	}
}

func TestTimelineChangeFieldsUnknownType(t *testing.T) {
	fields := timelineChangeFields(event.Event{Type: event.Type("session.started")})
	if fields != nil {
		t.Fatalf("fields = %#v, want nil", fields)
	}
}

func TestTimelineEntryFromEventAddsChangeProjectionWithoutResolverDisplay(t *testing.T) {
	hp := 6
	payload := daggerheartpayload.CharacterStatePatchedPayload{HP: &hp}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	resolver := newTimelineProjectionResolver(timelineProjectionStores{})
	entry, err := timelineEntryFromEvent(context.Background(), resolver, event.Event{
		Seq:         1,
		Type:        daggerheartpayload.EventTypeCharacterStatePatched,
		Timestamp:   time.Now().UTC(),
		PayloadJSON: payloadJSON,
	})
	if err != nil {
		t.Fatalf("timelineEntryFromEvent returned error: %v", err)
	}
	if entry.GetProjection() == nil {
		t.Fatal("expected projection to be created for change fields")
	}
	if len(entry.GetProjection().GetFields()) != 1 {
		t.Fatalf("projection field count = %d, want 1", len(entry.GetProjection().GetFields()))
	}
	field := entry.GetProjection().GetFields()[0]
	if field.GetLabel() != "HP" || field.GetValue() != "= 6" {
		t.Fatalf("field = (%q,%q), want (HP, = 6)", field.GetLabel(), field.GetValue())
	}
}

func testIntPtr(value int) *int          { return &value }
func testStringPtr(value string) *string { return &value }
