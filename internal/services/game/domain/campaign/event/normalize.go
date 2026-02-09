package event

import (
	"encoding/json"
	"fmt"
	"strings"
)

// NormalizeForAppend validates and normalizes an event before storage assigns sequencing.
func NormalizeForAppend(evt Event) (Event, error) {
	evt.CampaignID = strings.TrimSpace(evt.CampaignID)
	if evt.CampaignID == "" {
		return Event{}, fmt.Errorf("campaign id is required")
	}
	if evt.Seq != 0 {
		return Event{}, fmt.Errorf("event sequence must be assigned by storage")
	}
	if strings.TrimSpace(evt.Hash) != "" {
		return Event{}, fmt.Errorf("event hash must be assigned by storage")
	}
	if strings.TrimSpace(evt.PrevHash) != "" || strings.TrimSpace(evt.ChainHash) != "" {
		return Event{}, fmt.Errorf("event chain hashes must be assigned by storage")
	}
	if strings.TrimSpace(evt.SignatureKeyID) != "" || strings.TrimSpace(evt.Signature) != "" {
		return Event{}, fmt.Errorf("event signatures must be assigned by storage")
	}

	evt.Type = Type(strings.TrimSpace(string(evt.Type)))
	if !evt.Type.IsValid() {
		return Event{}, fmt.Errorf("event type is required")
	}

	evt.ActorType = ActorType(strings.TrimSpace(string(evt.ActorType)))
	if evt.ActorType == "" {
		evt.ActorType = ActorTypeSystem
	}
	switch evt.ActorType {
	case ActorTypeSystem, ActorTypeParticipant, ActorTypeGM:
		// allowed
	default:
		return Event{}, fmt.Errorf("actor type must be system, participant, or gm")
	}
	evt.ActorID = strings.TrimSpace(evt.ActorID)
	if (evt.ActorType == ActorTypeParticipant || evt.ActorType == ActorTypeGM) && evt.ActorID == "" {
		return Event{}, fmt.Errorf("actor id is required for participant or gm")
	}

	evt.SessionID = strings.TrimSpace(evt.SessionID)
	evt.RequestID = strings.TrimSpace(evt.RequestID)
	evt.InvocationID = strings.TrimSpace(evt.InvocationID)
	evt.EntityType = strings.TrimSpace(evt.EntityType)
	evt.EntityID = strings.TrimSpace(evt.EntityID)

	if len(evt.PayloadJSON) == 0 {
		evt.PayloadJSON = []byte("{}")
	}
	if !json.Valid(evt.PayloadJSON) {
		return Event{}, fmt.Errorf("payload json must be valid JSON")
	}

	return evt, nil
}
