package game

import (
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Timestamp helpers.
func timestampOrNil(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}
	return timestamppb.New(value.UTC())
}

func structToMap(input *structpb.Struct) map[string]any {
	if input == nil {
		return nil
	}
	return input.AsMap()
}

func validateStructPayload(values map[string]any) error {
	for key := range values {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("payload keys must be non-empty")
		}
	}
	return nil
}
