package campaign

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/fold"
)

// foldRouter is the registration-based fold dispatcher. Handled types are
// derived from registered handlers, eliminating sync-drift between the switch
// and the type list.
var foldRouter = newFoldRouter()

func newFoldRouter() *fold.CoreFoldRouter[State] {
	r := fold.NewCoreFoldRouter[State]()
	r.Handle(EventTypeCreated, foldCreated)
	r.Handle(EventTypeUpdated, foldUpdated)
	r.Handle(EventTypeAIBound, foldAIBound)
	r.Handle(EventTypeAIUnbound, foldAIUnbound)
	r.Handle(EventTypeAIAuthRotated, foldAIAuthRotated)
	r.Handle(EventTypeForked, foldForked)
	return r
}

// FoldHandledTypes returns the event types handled by the campaign fold function.
// Derived from registered handlers via the fold router.
func FoldHandledTypes() []event.Type {
	return foldRouter.FoldHandledTypes()
}

// Fold applies an event to campaign state. Returns an error for unhandled
// event types and for recognized events with unparseable payloads.
func Fold(state State, evt event.Event) (State, error) {
	return foldRouter.Fold(state, evt)
}

func foldCreated(state State, evt event.Event) (State, error) {
	state.Created = true
	var payload CreatePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("campaign fold %s: %w", evt.Type, err)
	}
	state.Name = payload.Name
	state.Locale = normalizeCampaignLocale(payload.Locale)
	state.GameSystem = GameSystem(payload.GameSystem)
	state.GmMode = GmMode(payload.GmMode)
	state.Status = StatusDraft
	state.ThemePrompt = strings.TrimSpace(payload.ThemePrompt)
	state.CoverAssetID = strings.TrimSpace(payload.CoverAssetID)
	state.CoverSetID = strings.TrimSpace(payload.CoverSetID)
	return state, nil
}

func foldUpdated(state State, evt event.Event) (State, error) {
	var payload UpdatePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("campaign fold %s: %w", evt.Type, err)
	}
	for key, value := range payload.Fields {
		switch key {
		case "name":
			state.Name = strings.TrimSpace(value)
		case "status":
			state.Status = Status(strings.TrimSpace(value))
		case "theme_prompt":
			state.ThemePrompt = strings.TrimSpace(value)
		case "locale":
			state.Locale = normalizeCampaignLocale(value)
		case "cover_asset_id":
			state.CoverAssetID = strings.TrimSpace(value)
		case "cover_set_id":
			state.CoverSetID = strings.TrimSpace(value)
		}
	}
	return state, nil
}

func foldAIBound(state State, evt event.Event) (State, error) {
	var payload AIBindPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("campaign fold %s: %w", evt.Type, err)
	}
	state.AIAgentID = strings.TrimSpace(payload.AIAgentID)
	return state, nil
}

func foldAIUnbound(state State, _ event.Event) (State, error) {
	state.AIAgentID = ""
	return state, nil
}

func foldAIAuthRotated(state State, evt event.Event) (State, error) {
	var payload AIAuthRotatePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("campaign fold %s: %w", evt.Type, err)
	}
	state.AIAuthEpoch = payload.EpochAfter
	return state, nil
}

// foldForked acknowledges the event for fold coverage validation. Fork lineage
// metadata does not affect campaign aggregate state.
func foldForked(state State, _ event.Event) (State, error) {
	return state, nil
}
