import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AIDebugPanel } from "./AIDebugPanel";

describe("AIDebugPanel", () => {
  it("renders turn summaries and keeps payloads collapsed initially", () => {
    const onToggleTurn = vi.fn();
    render(
      <AIDebugPanel
        state={{
          phase: "ready",
          turns: [{
            id: "turn-1",
            model: "gpt-4.1-mini",
            provider: "openai",
            status: "failed",
            entry_count: 1,
            started_at: "2026-03-22T12:00:00Z",
            last_error: "scene_create failed",
          }],
          expandedTurnId: "turn-1",
          detailsByTurnId: {
            "turn-1": {
              id: "turn-1",
              model: "gpt-4.1-mini",
              provider: "openai",
              status: "failed",
              entries: [{
                sequence: 1,
                kind: "tool_result",
                tool_name: "scene_create",
                payload: "{\"error\":\"missing scene\"}",
                payload_truncated: false,
                is_error: true,
                created_at: "2026-03-22T12:00:01Z",
              }],
            },
          },
        }}
        onToggleTurn={onToggleTurn}
      />,
    );

    expect(screen.getByText("AI Debug")).toBeInTheDocument();
    expect(screen.getByText("scene_create failed")).toBeInTheDocument();
    expect(screen.getByText("Tool result (error)")).toBeInTheDocument();
    // Invariant: AI Debug should refresh from realtime/reconnect, not a manual control.
    expect(screen.queryByRole("button", { name: "Refresh" })).toBeNull();
    expect(screen.getByText("Payload").closest("details")).not.toHaveAttribute("open");
  });
});
