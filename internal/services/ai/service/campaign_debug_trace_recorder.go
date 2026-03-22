package service

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/louisbranch/fracturing.space/internal/services/ai/debugtrace"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

const campaignDebugPayloadMaxBytes = 64 * 1024

const campaignDebugPayloadSuffix = "\n\n[truncated by AI debug payload budget]"

// campaignDebugTraceRecorder persists a best-effort ordered trace for one GM turn.
type campaignDebugTraceRecorder struct {
	store  storage.DebugTraceStore
	clock  Clock
	broker *CampaignDebugUpdateBroker

	logger *slog.Logger

	mu       sync.Mutex
	turn     debugtrace.Turn
	sequence int
	disabled bool
}

// newCampaignDebugTraceRecorder initializes one running turn record when debug tracing is available.
func newCampaignDebugTraceRecorder(
	ctx context.Context,
	store storage.DebugTraceStore,
	clock Clock,
	broker *CampaignDebugUpdateBroker,
	idGenerator IDGenerator,
	logger *slog.Logger,
	turn debugtrace.Turn,
) *campaignDebugTraceRecorder {
	if store == nil {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	recorder := &campaignDebugTraceRecorder{
		store:  store,
		clock:  withDefaultClock(clock),
		broker: broker,
		logger: logger,
		turn:   turn,
	}
	if strings.TrimSpace(recorder.turn.ID) == "" {
		id, err := withDefaultIDGenerator(idGenerator)()
		if err != nil {
			logger.ErrorContext(ctx, "generate campaign debug turn id", "error", err)
			recorder.disabled = true
			return recorder
		}
		recorder.turn.ID = id
	}
	now := recorder.clock().UTC()
	if recorder.turn.StartedAt.IsZero() {
		recorder.turn.StartedAt = now
	}
	recorder.turn.UpdatedAt = now
	recorder.turn.Status = debugtrace.StatusRunning
	if err := recorder.store.PutCampaignDebugTurn(ctx, recorder.turn); err != nil {
		logger.ErrorContext(ctx, "persist campaign debug turn", "turn_id", recorder.turn.ID, "error", err)
		recorder.disabled = true
		return recorder
	}
	recorder.publishUpdate(nil)
	return recorder
}

// TurnID returns the persisted trace id when a turn record was initialized.
func (r *campaignDebugTraceRecorder) TurnID() string {
	if r == nil {
		return ""
	}
	return r.turn.ID
}

// RecordProviderStep appends model output and tool-call requests for one provider response.
func (r *campaignDebugTraceRecorder) RecordProviderStep(ctx context.Context, output orchestration.ProviderOutput) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.disabled {
		return
	}

	now := r.clock().UTC()
	r.turn.Usage = r.turn.Usage.Add(output.Usage)
	r.turn.UpdatedAt = now

	entries := make([]debugtrace.Entry, 0, len(output.ToolCalls)+1)
	usageRecorded := false
	if text := strings.TrimSpace(output.OutputText); text != "" {
		payload, truncated := truncateCampaignDebugPayload(text)
		entries = append(entries, debugtrace.Entry{
			Kind:             debugtrace.EntryKindModelResponse,
			Payload:          payload,
			PayloadTruncated: truncated,
			ResponseID:       strings.TrimSpace(output.ConversationID),
			CreatedAt:        now,
			Usage:            output.Usage,
		})
		usageRecorded = true
	}
	for _, call := range output.ToolCalls {
		payload, truncated := truncateCampaignDebugPayload(strings.TrimSpace(call.Arguments))
		entry := debugtrace.Entry{
			Kind:             debugtrace.EntryKindToolCall,
			ToolName:         strings.TrimSpace(call.Name),
			Payload:          payload,
			PayloadTruncated: truncated,
			CallID:           strings.TrimSpace(call.CallID),
			ResponseID:       strings.TrimSpace(output.ConversationID),
			CreatedAt:        now,
		}
		if !usageRecorded {
			entry.Usage = output.Usage
			usageRecorded = true
		}
		entries = append(entries, entry)
	}
	r.appendEntries(ctx, entries)
}

// RecordToolResult appends one tool execution result as seen by the provider.
func (r *campaignDebugTraceRecorder) RecordToolResult(ctx context.Context, call orchestration.ProviderToolCall, result orchestration.ProviderToolResult) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.disabled {
		return
	}

	now := r.clock().UTC()
	r.turn.UpdatedAt = now
	payload, truncated := truncateCampaignDebugPayload(result.Output)
	r.appendEntries(ctx, []debugtrace.Entry{{
		Kind:             debugtrace.EntryKindToolResult,
		ToolName:         strings.TrimSpace(call.Name),
		Payload:          payload,
		PayloadTruncated: truncated,
		CallID:           strings.TrimSpace(result.CallID),
		IsError:          result.IsError,
		CreatedAt:        now,
	}})
}

// Finish marks the trace terminal so failed or completed turns remain inspectable.
func (r *campaignDebugTraceRecorder) Finish(ctx context.Context, err error) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.disabled {
		return
	}

	now := r.clock().UTC()
	r.turn.UpdatedAt = now
	r.turn.CompletedAt = &now
	r.turn.LastError = ""
	r.turn.Status = debugtrace.StatusSucceeded
	if err != nil {
		r.turn.Status = debugtrace.StatusFailed
		r.turn.LastError = strings.TrimSpace(err.Error())
	}
	if persistErr := r.store.PutCampaignDebugTurn(ctx, r.turn); persistErr != nil {
		r.logger.ErrorContext(ctx, "finalize campaign debug turn", "turn_id", r.turn.ID, "error", persistErr)
		r.disabled = true
		return
	}
	r.publishUpdate(nil)
}

// appendEntries persists one ordered batch and refreshes the turn summary only after entry writes succeed.
func (r *campaignDebugTraceRecorder) appendEntries(ctx context.Context, entries []debugtrace.Entry) {
	if len(entries) == 0 {
		if err := r.store.PutCampaignDebugTurn(ctx, r.turn); err != nil {
			r.logger.ErrorContext(ctx, "update campaign debug turn", "turn_id", r.turn.ID, "error", err)
			r.disabled = true
		}
		return
	}
	for i := range entries {
		r.sequence++
		entries[i].TurnID = r.turn.ID
		entries[i].Sequence = r.sequence
		if err := r.store.PutCampaignDebugTurnEntry(ctx, entries[i]); err != nil {
			r.logger.ErrorContext(ctx, "persist campaign debug turn entry", "turn_id", r.turn.ID, "sequence", r.sequence, "error", err)
			r.disabled = true
			return
		}
	}
	r.turn.EntryCount = r.sequence
	if err := r.store.PutCampaignDebugTurn(ctx, r.turn); err != nil {
		r.logger.ErrorContext(ctx, "update campaign debug turn", "turn_id", r.turn.ID, "error", err)
		r.disabled = true
		return
	}
	r.publishUpdate(entries)
}

func (r *campaignDebugTraceRecorder) publishUpdate(entries []debugtrace.Entry) {
	if r == nil || r.broker == nil {
		return
	}
	r.broker.Publish(r.turn.CampaignID, r.turn.SessionID, CampaignDebugTurnUpdate{
		Turn:            r.turn,
		AppendedEntries: entries,
	})
}

// truncateCampaignDebugPayload bounds stored payloads so traces remain readable and durable.
func truncateCampaignDebugPayload(payload string) (string, bool) {
	if len(payload) <= campaignDebugPayloadMaxBytes {
		return payload, false
	}
	suffix := campaignDebugPayloadSuffix
	if campaignDebugPayloadMaxBytes <= len(suffix) {
		return suffix[:campaignDebugPayloadMaxBytes], true
	}
	limit := campaignDebugPayloadMaxBytes - len(suffix)
	for limit > 0 && (payload[limit]&0xC0) == 0x80 {
		limit--
	}
	return payload[:limit] + suffix, true
}
