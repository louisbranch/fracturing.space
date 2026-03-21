import { OnStageCharacterAvatarStack } from "../on-stage-character-avatar-stack/OnStageCharacterAvatarStack";
import type { OnStageSlotCardProps } from "./contract";

function formatTime(iso: string | undefined): string | undefined {
  if (!iso) {
    return undefined;
  }
  return new Date(iso).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  });
}

function reviewDisplay(reviewState: OnStageSlotCardProps["slot"]["reviewState"]): {
  label: string;
} {
  switch (reviewState) {
    case "under-review":
      return { label: "Under Review" };
    case "accepted":
      return { label: "Accepted" };
    case "changes-requested":
      return { label: "Changes Requested" };
    default:
      return { label: "Open" };
  }
}

function joinCharacterNames(names: string[]): string {
  if (names.length === 0) {
    return "No acting characters";
  }
  return names.join(", ");
}

export function OnStageSlotCard({
  slot,
  participant,
  isViewer,
  onCharacterInspect,
}: OnStageSlotCardProps) {
  const characters = slot.characters.length > 0 ? slot.characters : participant.characters;
  const review = reviewDisplay(slot.reviewState);
  const body = slot.body?.trim();
  const time = formatTime(slot.updatedAt);
  const characterNames = joinCharacterNames(characters.map((character) => character.name));
  const statusLabel = slot.yielded && slot.reviewState === "open" ? "Yielded" : review.label;

  return (
    <article
      aria-label={`On-stage slot by ${participant.name}`}
      className={`rounded-box border border-base-300/70 bg-base-100/90 p-3 shadow-sm ${
        isViewer ? "ring-2 ring-primary/70 ring-offset-2 ring-offset-base-100" : ""
      }`}
    >
      <header className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2">
          <OnStageCharacterAvatarStack
            characters={characters}
            onCharacterInspect={(characterId) =>
              onCharacterInspect?.(participant.id, characterId)
            }
          />
          <div className="min-w-0 flex items-center gap-2 text-sm text-base-content">
            <h3 className="min-w-0 font-semibold">{participant.name}</h3>
            {isViewer ? <span className="badge badge-primary badge-soft badge-sm">You</span> : null}
            <span className="font-normal text-base-content/75">as {characterNames}</span>
          </div>
        </div>

        <div className="shrink-0">
          <span className="badge badge-sm badge-ghost">{statusLabel}</span>
        </div>
      </header>

      <div className="mt-2.5 space-y-2.5">
        <div className="flex items-end justify-between gap-2">
          <p className={`text-sm leading-relaxed ${body ? "text-base-content/85" : "italic text-base-content/55"}`}>
            {body || "No committed post yet."}
          </p>
          {time ? (
            <span className="shrink-0 text-xs font-medium uppercase tracking-wide text-base-content/45">
              {time}
            </span>
          ) : null}
        </div>

        {slot.reviewReason ? (
          <section className="rounded-box border border-warning/30 bg-warning/10 px-2.5 py-2" aria-label="Revision request">
            <div className="text-xs font-semibold uppercase tracking-wide text-base-content/60">
              Revision Request
            </div>
            <p className="mt-1 text-sm text-base-content/80">{slot.reviewReason}</p>
          </section>
        ) : null}
      </div>
    </article>
  );
}
