package generator

import (
	"context"
	"encoding/json"
	"fmt"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

// createSessions creates the specified number of sessions for a campaign.
// All but the last session are ended. Events are added to each session.
func (g *Generator) createSessions(ctx context.Context, campaignID string, count int, cfg PresetConfig, characters []*statev1.Character) error {
	if count < 1 {
		return nil
	}

	for i := 0; i < count; i++ {
		seq := i + 1
		sessionName := g.wb.SessionName(seq)

		resp, err := g.sessions.StartSession(ctx, &statev1.StartSessionRequest{
			CampaignId: campaignID,
			Name:       sessionName,
		})
		if err != nil {
			return fmt.Errorf("StartSession %d: %w", seq, err)
		}

		session := resp.Session

		// Add events to the session
		numEvents := g.randomRange(cfg.EventsMin, cfg.EventsMax)
		if err := g.addSessionEvents(ctx, campaignID, session.Id, numEvents, characters); err != nil {
			return fmt.Errorf("add events to session %d: %w", seq, err)
		}

		// End all sessions except the last one (unless we want ended sessions)
		isLastSession := i == count-1
		shouldEnd := !isLastSession || (cfg.IncludeEndedSessions && count > 1)

		if shouldEnd {
			_, err := g.sessions.EndSession(ctx, &statev1.EndSessionRequest{
				CampaignId: campaignID,
				SessionId:  session.Id,
			})
			if err != nil {
				return fmt.Errorf("EndSession %d: %w", seq, err)
			}
		}
	}

	return nil
}

// addSessionEvents adds the specified number of events to a session.
func (g *Generator) addSessionEvents(ctx context.Context, campaignID, sessionID string, count int, characters []*statev1.Character) error {
	for i := 0; i < count; i++ {
		eventType := g.pickEventType()
		if err := g.createEvent(ctx, campaignID, sessionID, eventType, characters); err != nil {
			return fmt.Errorf("create event %d: %w", i+1, err)
		}
	}
	return nil
}

// eventType represents the type of session event to create.
type eventType int

const (
	eventTypeNote eventType = iota
	eventTypeRoll
)

// pickEventType selects an event type with weighted probability.
func (g *Generator) pickEventType() eventType {
	// 60% notes, 40% rolls
	if g.rng.Float32() < 0.6 {
		return eventTypeNote
	}
	return eventTypeRoll
}

// createEvent creates a single session event.
func (g *Generator) createEvent(ctx context.Context, campaignID, sessionID string, et eventType, characters []*statev1.Character) error {
	switch et {
	case eventTypeNote:
		return g.createNoteEvent(ctx, campaignID, sessionID)
	case eventTypeRoll:
		return g.createRollEvent(ctx, campaignID, sessionID, characters)
	default:
		return nil
	}
}

// notePayload represents the JSON payload for NOTE_ADDED events.
type notePayload struct {
	Content string `json:"content"`
}

// createNoteEvent creates a NOTE_ADDED event.
func (g *Generator) createNoteEvent(ctx context.Context, campaignID, sessionID string) error {
	payload := notePayload{Content: g.wb.NoteContent()}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal note payload: %w", err)
	}

	_, err = g.events.AppendEvent(ctx, &statev1.AppendEventRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		Type:        "story.note_added",
		ActorType:   "system",
		EntityType:  "session",
		EntityId:    sessionID,
		PayloadJson: payloadJSON,
	})
	if err != nil {
		return fmt.Errorf("AppendEvent: %w", err)
	}
	return nil
}

// createRollEvent creates an action roll event.
// Note: Action rolls are more complex and may require character setup.
// For simplicity, we skip rolls if there are no characters.
func (g *Generator) createRollEvent(ctx context.Context, campaignID, sessionID string, characters []*statev1.Character) error {
	if len(characters) == 0 {
		// Fall back to a note event if no characters available
		return g.createNoteEvent(ctx, campaignID, sessionID)
	}

	// For now, just create note events since action rolls require more setup
	// (active session, valid character state, etc.)
	// In the future, this could be expanded to perform actual rolls
	return g.createNoteEvent(ctx, campaignID, sessionID)
}
