import { ChevronLeft, ChevronRight, BookOpen, CircleHelp, Compass, Dices, Zap } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import type { ComponentType } from "react";
import { PlayerHUDStatusPill } from "../../shared/PlayerHUDStatusPill";
import type { PlayerHUDStatusBadge } from "../../shared/view-models";
import type { OnStageGMBeatType, OnStageGMInteractionIllustration } from "../shared/contract";
import type { OnStageGMInteractionCardProps } from "./contract";

type IconProps = { className?: string };

const beatIcon: Record<OnStageGMBeatType, ComponentType<IconProps>> = {
  fiction: BookOpen,
  prompt: CircleHelp,
  resolution: Dices,
  consequence: Zap,
  guidance: Compass,
};

const beatLabel: Record<OnStageGMBeatType, string> = {
  fiction: "Fiction",
  prompt: "Prompt",
  resolution: "Resolution",
  consequence: "Consequence",
  guidance: "Guidance",
};

const concludedStatus: PlayerHUDStatusBadge = {
  className: "badge-ghost",
  indicator: "none",
  label: "Concluded",
  tooltip: "This interaction is part of the resolved scene history.",
};

const sizeHintClass: Record<NonNullable<OnStageGMInteractionIllustration["sizeHint"]>, string> = {
  compact: "max-w-[20%]",
  wide: "max-w-[40%]",
};

const beatTextClass = (type: OnStageGMBeatType): string => {
  switch (type) {
    case "guidance":
      return "whitespace-pre-line text-sm leading-7 italic text-base-content/60";
    default:
      return "whitespace-pre-line text-sm leading-7 text-base-content/80";
  }
};

export function OnStageGMInteractionCard({
  currentInteraction,
  interactionHistory,
  currentStatus,
}: OnStageGMInteractionCardProps) {
  const interactions = useMemo(
    () => (currentInteraction ? [currentInteraction, ...interactionHistory] : interactionHistory),
    [currentInteraction, interactionHistory],
  );
  const [interactionIndex, setInteractionIndex] = useState(0);

  useEffect(() => {
    setInteractionIndex(0);
  }, [currentInteraction?.id, interactionHistory[0]?.id, interactionHistory.length]);

  const interaction = interactions[interactionIndex];
  const canGoOlder = interactionIndex < interactions.length - 1;
  const canGoNewer = interactionIndex > 0;
  const status = interactionIndex === 0 ? currentStatus : concludedStatus;

  if (!interaction) {
    return (
      <section
        aria-label="On-stage GM interaction"
        className="mx-3 mt-4 min-w-0 rounded-box border border-dashed border-base-300/70 bg-base-100/70 px-4 py-4 text-sm text-base-content/65"
      >
        No GM interactions are available for this scene yet.
      </section>
    );
  }

  const illustration = interaction.illustration;

  return (
    <section aria-label="On-stage GM interaction" className="mx-3 mt-4 min-w-0">
      <article className="min-w-0 rounded-box border border-base-300/70 bg-base-100 px-4 py-4 shadow-sm">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="flex min-w-0 items-center gap-2">
            {canGoOlder ? (
              <button
                type="button"
                className="btn btn-ghost btn-sm btn-circle"
                aria-label="Show older interaction"
                onClick={() => setInteractionIndex((current) => Math.min(interactions.length - 1, current + 1))}
              >
                <ChevronLeft className="h-4 w-4" />
              </button>
            ) : null}
            <h3 className="min-w-0 truncate text-base font-semibold text-base-content">{interaction.title}</h3>
            {canGoNewer ? (
              <button
                type="button"
                className="btn btn-ghost btn-sm btn-circle"
                aria-label="Show newer interaction"
                onClick={() => setInteractionIndex((current) => Math.max(0, current - 1))}
              >
                <ChevronRight className="h-4 w-4" />
              </button>
            ) : null}
          </div>

          <PlayerHUDStatusPill
            ariaLabel={`On-stage interaction status: ${status.label}`}
            status={status}
          />
        </div>

        <div className="mt-2 min-w-0 overflow-hidden space-y-3 text-base-content">
          {illustration ? (
            <figure className={`float-end mb-3 ml-4 ${sizeHintClass[illustration.sizeHint ?? "wide"]}`}>
              <img
                src={illustration.imageUrl}
                alt={illustration.alt}
                className="w-full rounded-box object-cover"
              />
              {illustration.caption ? (
                <figcaption className="mt-1 text-xs text-base-content/60">{illustration.caption}</figcaption>
              ) : null}
            </figure>
          ) : null}
          {interaction.beats.map((beat, index) => {
            const Icon = beatIcon[beat.type];
            return (
              <div key={beat.id} className="min-w-0">
                <div className={`divider ${index === 0 ? "mt-0 mb-1" : "my-3"}`} aria-hidden="true">
                  <span className="tooltip tooltip-bottom" data-tip={beatLabel[beat.type]}>
                    <Icon className="size-5 shrink-0 text-base-content/7" />
                  </span>
                </div>
                <section aria-label={`${beat.type} beat`} className="min-w-0 space-y-3">
                  <p className={beatTextClass(beat.type)}>{beat.text}</p>
                </section>
              </div>
            );
          })}
          <div className="clear-both" />
        </div>
      </article>
    </section>
  );
}
