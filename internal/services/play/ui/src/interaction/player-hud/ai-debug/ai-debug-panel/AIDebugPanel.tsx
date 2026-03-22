import type { WireAIDebugEntry, WireAIDebugTurn, WireUsage } from "../../../../api/types";
import type { AIDebugPanelState } from "../shared/contract";

type AIDebugPanelProps = {
  state: AIDebugPanelState;
  onLoadMore?: () => void;
  onToggleTurn?: (turnId: string) => void;
};

function formatTimestamp(value?: string): string {
  if (!value) {
    return "Unknown time";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

function formatUsage(usage?: WireUsage): string {
  if (!usage) {
    return "";
  }
  const parts = [
    usage.input_tokens ? `in ${usage.input_tokens}` : "",
    usage.output_tokens ? `out ${usage.output_tokens}` : "",
    usage.reasoning_tokens ? `reason ${usage.reasoning_tokens}` : "",
    usage.total_tokens ? `total ${usage.total_tokens}` : "",
  ].filter(Boolean);
  return parts.join(" • ");
}

function statusTone(status?: string): string {
  switch (status) {
    case "running":
      return "badge-warning";
    case "failed":
      return "badge-error";
    case "succeeded":
      return "badge-success";
    default:
      return "badge-ghost";
  }
}

function entryLabel(entry: WireAIDebugEntry): string {
  switch (entry.kind) {
    case "model_response":
      return "Model response";
    case "tool_call":
      return "Tool call";
    case "tool_result":
      return entry.is_error ? "Tool result (error)" : "Tool result";
    default:
      return "Trace entry";
  }
}

function EntryRow({ entry }: { entry: WireAIDebugEntry }) {
  const usage = formatUsage(entry.usage);
  return (
    <article className="rounded-box border border-base-300 bg-base-100/70 p-3">
      <header className="flex flex-wrap items-center gap-2 text-sm">
        <span className="font-semibold">{entryLabel(entry)}</span>
        {entry.tool_name ? <span className="badge badge-outline">{entry.tool_name}</span> : null}
        <span className="text-base-content/60">{formatTimestamp(entry.created_at)}</span>
        {usage ? <span className="text-base-content/60">{usage}</span> : null}
      </header>
      <details className="mt-3 rounded-box bg-base-200/60 p-2">
        <summary className="cursor-pointer text-sm font-medium">
          Payload{entry.payload_truncated ? " (truncated)" : ""}
        </summary>
        <pre className="mt-2 overflow-x-auto whitespace-pre-wrap break-words text-xs">{entry.payload || "(empty)"}</pre>
      </details>
    </article>
  );
}

function TurnDetails({ turn }: { turn: WireAIDebugTurn }) {
  return (
    <div className="mt-4 space-y-3">
      {turn.entries.map((entry) => <EntryRow key={`${turn.id}-${entry.sequence}`} entry={entry} />)}
      {turn.entries.length === 0 ? (
        <div className="rounded-box border border-dashed border-base-300 p-4 text-sm text-base-content/70">
          No trace entries were captured for this turn.
        </div>
      ) : null}
    </div>
  );
}

export function AIDebugPanel({ state, onLoadMore, onToggleTurn }: AIDebugPanelProps) {
  return (
    <section aria-label="AI Debug panel" className="flex min-h-0 flex-1 flex-col overflow-hidden">
      <header className="border-b border-base-300 px-4 py-3">
        <div>
          <h2 className="text-lg font-semibold">AI Debug</h2>
          <p className="text-sm text-base-content/60">Tool calls, model responses, and provider usage for this session.</p>
        </div>
      </header>

      <div className="flex-1 overflow-y-auto p-4">
        {state.phase === "loading" && state.turns.length === 0 ? (
          <div className="flex h-full items-center justify-center">
            <span className="loading loading-spinner loading-md" />
          </div>
        ) : null}

        {state.errorMessage ? (
          <div className="alert alert-error mb-4">
            <span>{state.errorMessage}</span>
          </div>
        ) : null}

        {state.turns.length === 0 && state.phase !== "loading" ? (
          <div className="rounded-box border border-dashed border-base-300 p-6 text-sm text-base-content/70">
            No AI GM debug traces exist for this session yet.
          </div>
        ) : null}

        <div className="space-y-4">
          {state.turns.map((turn) => {
            const usage = formatUsage(turn.usage);
            const expanded = state.expandedTurnId === turn.id;
            const detail = state.detailsByTurnId[turn.id];
            const isLoadingDetail = state.loadingTurnId === turn.id && !detail;
            return (
              <article key={turn.id} className="rounded-box border border-base-300 bg-base-200/40 p-4">
                <button
                  type="button"
                  aria-expanded={expanded}
                  className="flex w-full cursor-pointer flex-col items-start gap-3 rounded-box px-1 py-1 text-left transition-colors hover:bg-base-100/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/70"
                  onClick={() => onToggleTurn?.(turn.id)}
                >
                  <div className="flex w-full flex-wrap items-center gap-2">
                    <span className={`badge ${statusTone(turn.status)}`}>{turn.status || "unknown"}</span>
                    <span className="font-semibold">{turn.model || "Unknown model"}</span>
                    {turn.provider ? <span className="text-sm text-base-content/60">{turn.provider}</span> : null}
                    <span className="text-sm text-base-content/60">{formatTimestamp(turn.started_at)}</span>
                  </div>
                  <div className="flex flex-wrap gap-x-4 gap-y-1 text-sm text-base-content/70">
                    <span>{turn.entry_count ?? 0} entries</span>
                    {usage ? <span>{usage}</span> : null}
                    {turn.completed_at ? <span>Completed {formatTimestamp(turn.completed_at)}</span> : null}
                    {turn.last_error ? <span className="text-error">{turn.last_error}</span> : null}
                  </div>
                </button>
                {expanded ? (
                  isLoadingDetail ? (
                    <div className="mt-4 flex items-center gap-2 text-sm text-base-content/70">
                      <span className="loading loading-spinner loading-sm" />
                      Loading turn trace…
                    </div>
                  ) : detail ? (
                    <TurnDetails turn={detail} />
                  ) : null
                ) : null}
              </article>
            );
          })}
        </div>

        {state.nextPageToken ? (
          <div className="mt-4 flex justify-center">
            <button type="button" className="btn btn-sm btn-outline" onClick={onLoadMore}>
              Load older turns
            </button>
          </div>
        ) : null}
      </div>
    </section>
  );
}
