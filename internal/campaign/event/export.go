package event

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// ExportHumanReadable writes events to w in a human-readable format for debugging and auditing.
func ExportHumanReadable(events []Event, w io.Writer) error {
	for i, evt := range events {
		if i > 0 {
			fmt.Fprintln(w)
		}
		if err := writeEvent(evt, w); err != nil {
			return fmt.Errorf("write event %d: %w", evt.Seq, err)
		}
	}
	return nil
}

func writeEvent(evt Event, w io.Writer) error {
	// Header line with timestamp and type
	fmt.Fprintf(w, "[%s] %s\n", evt.Timestamp.UTC().Format(time.RFC3339), evt.Type)

	// Hash
	if evt.Hash != "" {
		fmt.Fprintf(w, "  hash: %s\n", evt.Hash)
	}

	// Campaign
	fmt.Fprintf(w, "  campaign: %s\n", evt.CampaignID)

	// Sequence
	fmt.Fprintf(w, "  seq: %d\n", evt.Seq)

	// Session (optional)
	if evt.SessionID != "" {
		fmt.Fprintf(w, "  session: %s\n", evt.SessionID)
	}

	// Correlation IDs (optional)
	if evt.RequestID != "" {
		fmt.Fprintf(w, "  request: %s\n", evt.RequestID)
	}
	if evt.InvocationID != "" {
		fmt.Fprintf(w, "  invocation: %s\n", evt.InvocationID)
	}

	// Actor
	actorStr := string(evt.ActorType)
	if evt.ActorID != "" {
		actorStr = fmt.Sprintf("%s/%s", evt.ActorType, evt.ActorID)
	}
	fmt.Fprintf(w, "  actor: %s\n", actorStr)

	// Entity (optional)
	if evt.EntityType != "" {
		entityStr := evt.EntityType
		if evt.EntityID != "" {
			entityStr = fmt.Sprintf("%s/%s", evt.EntityType, evt.EntityID)
		}
		fmt.Fprintf(w, "  entity: %s\n", entityStr)
	}

	// Payload
	if len(evt.PayloadJSON) > 0 {
		fmt.Fprintln(w, "  payload:")
		if err := writeIndentedJSON(evt.PayloadJSON, w, "    "); err != nil {
			// Fall back to raw bytes if JSON parsing fails
			fmt.Fprintf(w, "    %s\n", string(evt.PayloadJSON))
		}
	}

	return nil
}

func writeIndentedJSON(data []byte, w io.Writer, prefix string) error {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	// Format with indentation
	formatted, err := json.MarshalIndent(v, prefix, "  ")
	if err != nil {
		return err
	}

	// Write with prefix on first line
	fmt.Fprintf(w, "%s%s\n", prefix, formatted)
	return nil
}
