package handler

import (
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TimestampOrNil converts a time pointer to a proto timestamp, returning nil
// when the input is nil.
func TimestampOrNil(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}
	return timestamppb.New(value.UTC())
}

// StructToMap converts a proto Struct to a Go map, returning nil when the
// input is nil.
func StructToMap(input *structpb.Struct) map[string]any {
	if input == nil {
		return nil
	}
	return input.AsMap()
}

// ValidateStructPayload rejects payloads containing empty keys.
func ValidateStructPayload(values map[string]any) error {
	for key := range values {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("payload keys must be non-empty")
		}
	}
	return nil
}
