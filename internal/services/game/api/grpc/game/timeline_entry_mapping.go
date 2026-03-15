package game

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// timelineEntryFromEvent builds a timeline entry with projection context.
func timelineEntryFromEvent(ctx context.Context, resolver *timelineProjectionResolver, evt event.Event) (*campaignv1.TimelineEntry, error) {
	iconID, projection, err := resolver.resolve(ctx, evt)
	if err != nil {
		return nil, err
	}
	changeFields := timelineChangeFields(evt)
	if len(changeFields) > 0 {
		if projection == nil {
			projection = &campaignv1.ProjectionDisplay{}
		}
		projection.Fields = append(projection.Fields, changeFields...)
	}
	return &campaignv1.TimelineEntry{
		Seq:              evt.Seq,
		EventType:        string(evt.Type),
		EventTime:        timestamppb.New(evt.Timestamp),
		IconId:           iconID,
		Projection:       projection,
		EventPayloadJson: string(evt.PayloadJSON),
	}, nil
}

func timelineChangeFields(evt event.Event) []*campaignv1.ProjectionField {
	switch evt.Type {
	case handler.EventTypeDaggerheartCharacterStatePatched:
		return daggerheartStateChangeFields(evt.PayloadJSON)
	default:
		return nil
	}
}

func daggerheartStateChangeFields(payloadJSON []byte) []*campaignv1.ProjectionField {
	if len(payloadJSON) == 0 {
		return nil
	}
	var payload daggerheart.CharacterStatePatchedPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return nil
	}
	fields := make([]*campaignv1.ProjectionField, 0, 6)
	appendIntChange(&fields, "HP", nil, payload.HP)
	appendIntChange(&fields, "Hope", nil, payload.Hope)
	appendIntChange(&fields, "Hope Max", nil, payload.HopeMax)
	appendIntChange(&fields, "Stress", nil, payload.Stress)
	appendIntChange(&fields, "Armor", nil, payload.Armor)
	appendStringChange(&fields, "Life State", nil, payload.LifeState)
	return fields
}

func appendIntChange(fields *[]*campaignv1.ProjectionField, label string, before, after *int) {
	if after == nil {
		return
	}
	if before != nil {
		if *before == *after {
			return
		}
		*fields = append(*fields, &campaignv1.ProjectionField{
			Label: label,
			Value: strconv.Itoa(*before) + " -> " + strconv.Itoa(*after),
		})
		return
	}
	*fields = append(*fields, &campaignv1.ProjectionField{
		Label: label,
		Value: "= " + strconv.Itoa(*after),
	})
}

func appendStringChange(fields *[]*campaignv1.ProjectionField, label string, before, after *string) {
	if after == nil {
		return
	}
	afterValue := strings.TrimSpace(*after)
	if afterValue == "" {
		return
	}
	if before != nil {
		beforeValue := strings.TrimSpace(*before)
		if beforeValue == afterValue {
			return
		}
		if beforeValue != "" {
			*fields = append(*fields, &campaignv1.ProjectionField{
				Label: label,
				Value: beforeValue + " -> " + afterValue,
			})
			return
		}
	}
	*fields = append(*fields, &campaignv1.ProjectionField{
		Label: label,
		Value: "= " + afterValue,
	})
}

func eventDomainFromType(evtType event.Type) string {
	value := strings.TrimSpace(string(evtType))
	if value == "" {
		return ""
	}
	parts := strings.SplitN(value, ".", 2)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}
