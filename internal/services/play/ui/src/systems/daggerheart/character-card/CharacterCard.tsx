import type {
  CharacterCardProps,
  DaggerheartCharacterSummary,
  DaggerheartCharacterTraits,
} from "./contract";

type LabeledValue = {
  label: string;
  value: string;
};

type StatIconName = "armor" | "evasion" | "hope" | "hp" | "stress";

type StatValue = LabeledValue & {
  icon: StatIconName;
};

// CharacterCard is the stable Daggerheart card contract that future runtime
// adapters can target without depending on one permanent implementation.
export function CharacterCard({ character, variant }: CharacterCardProps) {
  const summary = character.daggerheart?.summary;
  const classSummary = summarizeClass(summary);
  const heritageSummary = summarizeHeritage(summary);
  const pronouns = character.identity?.pronouns?.trim();
  const statRows = buildStatRows(summary);
  const hopeValue = asStatValue("Hope", formatTrackValue(summary?.hope), "hope");
  const traitBadges = buildTraitBadges(character.daggerheart?.traits);

  if (variant === "portrait") {
    return (
      <article className="character-card card character-card-portrait-only" data-variant={variant}>
        <figure className="character-card-media">
          <Portrait characterName={character.name} portrait={character.portrait} />
        </figure>
        <span className="sr-only">{character.name}</span>
      </article>
    );
  }

  return (
    <article className="character-card card" data-variant={variant}>
      <div className="grid gap-0 sm:grid-cols-[14rem_minmax(0,1fr)]">
        <figure className="character-card-media">
          <Portrait characterName={character.name} portrait={character.portrait} />
        </figure>

        <div className="card-body gap-4 p-4 sm:p-5">
          <header className="space-y-3">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div className="space-y-1.5">
                <div className="flex flex-wrap items-baseline gap-x-2 gap-y-1">
                  <h2 className="font-display text-2xl text-base-content sm:text-3xl">{character.name}</h2>
                  {pronouns ? <span className="text-sm text-base-content/70">({pronouns})</span> : null}
                </div>
                {classSummary ? <p className="text-base font-medium text-base-content/90">{classSummary}</p> : null}
                {heritageSummary ? <p className="text-sm text-base-content/70">{heritageSummary}</p> : null}
              </div>
              {summary?.level !== undefined ? (
                <span className="badge badge-outline badge-sm">Level {summary.level}</span>
              ) : null}
            </div>

            {statRows.length > 0 || hopeValue ? (
              <div className="space-y-2">
                {statRows.length > 0 ? (
                  <section className="grid gap-2 text-sm text-base-content/85 sm:grid-cols-2" aria-label="Character statistics">
                    {statRows.map((row) => (
                      <div key={row.label} className="flex items-center justify-between gap-3 rounded-box border border-base-300/70 bg-base-200/45 px-3 py-2">
                        <span className="flex items-center gap-2 text-base-content/60">
                          <StatIcon name={row.icon} />
                          <span>{row.label}</span>
                        </span>
                        <span className="font-medium text-base-content">{row.value}</span>
                      </div>
                    ))}
                  </section>
                ) : null}

                {hopeValue ? (
                  <section aria-label="Character hope summary">
                    <div className="grid gap-2 text-sm text-base-content/85 sm:grid-cols-2">
                      {hopeValue ? (
                        <div className="flex items-center justify-between gap-3 rounded-box border border-base-300/70 bg-base-200/45 px-3 py-2">
                          <span className="flex items-center gap-2 text-base-content/60">
                            <StatIcon name={hopeValue.icon} />
                            <span>{hopeValue.label}</span>
                          </span>
                          <span className="font-medium text-base-content">{hopeValue.value}</span>
                        </div>
                      ) : null}
                    </div>
                  </section>
                ) : null}
              </div>
            ) : null}
          </header>

          {traitBadges.length > 0 ? (
            <section className="border-t border-base-300/80 pt-4" aria-label="Character traits">
              <div className="flex flex-nowrap gap-1 overflow-hidden">
                {traitBadges.map((item) => (
                  <span key={`basic-trait-${item}`} className="badge badge-outline badge-sm shrink-0 text-[0.7rem]">
                    {item}
                  </span>
                ))}
              </div>
            </section>
          ) : null}
        </div>
      </div>
    </article>
  );
}

function Portrait(input: {
  characterName: string;
  portrait: CharacterCardProps["character"]["portrait"];
}) {
  if (input.portrait.src) {
    return (
      <img
        alt={input.portrait.alt}
        className="character-card-portrait"
        height={input.portrait.height}
        src={input.portrait.src}
        width={input.portrait.width}
      />
    );
  }

  const initials = input.characterName
    .split(/\s+/)
    .filter(Boolean)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .slice(0, 2)
    .join("");

  return (
    <div aria-label={input.portrait.alt} className="character-card-portrait character-card-portrait-placeholder" role="img">
      <span aria-hidden="true" className="font-display text-6xl text-base-content/75">
        {initials || "?"}
      </span>
    </div>
  );
}

function buildStatRows(summary: DaggerheartCharacterSummary | undefined): StatValue[] {
  return [
    asStatValue("HP", formatTrackValue(summary?.hp), "hp"),
    asStatValue("Stress", formatTrackValue(summary?.stress), "stress"),
    asStatValue("Evasion", summary?.evasion !== undefined ? String(summary.evasion) : undefined, "evasion"),
    asStatValue("Armor", formatTrackValue(summary?.armor), "armor"),
  ].filter(isPresent);
}

function buildTraitBadges(traits: DaggerheartCharacterTraits | undefined): string[] {
  if (!traits) {
    return [];
  }

  return [
    formatBadge("AGI", traits.agility),
    formatBadge("STR", traits.strength),
    formatBadge("FIN", traits.finesse),
    formatBadge("INS", traits.instinct),
    formatBadge("PRE", traits.presence),
    formatBadge("KNO", traits.knowledge),
  ].filter(isPresent);
}

function summarizeClass(summary: DaggerheartCharacterSummary | undefined): string {
  if (!summary?.className) {
    return "";
  }
  if (!summary.subclassName) {
    return summary.className;
  }
  return `${summary.className} / ${summary.subclassName}`;
}

function summarizeHeritage(summary: DaggerheartCharacterSummary | undefined): string {
  if (!summary?.ancestryName) {
    return "";
  }
  if (!summary.communityName) {
    return summary.ancestryName;
  }
  return `${summary.ancestryName}, ${summary.communityName}`;
}

function formatBadge(label: string, value: string | undefined): string | undefined {
  if (!value) {
    return undefined;
  }
  return `${label} ${value}`;
}

function formatTrackValue(value: DaggerheartCharacterSummary["hp"] | undefined): string | undefined {
  if (!value) {
    return undefined;
  }
  return `${value.current}/${value.max}`;
}

function asLabeledValue(label: string, value: string | undefined): LabeledValue | undefined {
  const trimmed = value?.trim();
  if (!trimmed) {
    return undefined;
  }
  return { label, value: trimmed };
}

function asStatValue(label: string, value: string | undefined, icon: StatIconName): StatValue | undefined {
  const labeled = asLabeledValue(label, value);
  if (!labeled) {
    return undefined;
  }
  return { ...labeled, icon };
}

function isPresent<T>(value: T | null | undefined | ""): value is T {
  return value !== null && value !== undefined && value !== "";
}

export function StatIcon(input: { name: StatIconName }) {
  const commonProps = {
    "aria-hidden": true,
    "data-icon": input.name,
    className: "h-4 w-4 shrink-0 text-base-content/60",
    fill: "none",
    stroke: "currentColor",
    strokeLinecap: "round" as const,
    strokeLinejoin: "round" as const,
    strokeWidth: 2,
    viewBox: "0 0 24 24",
  };

  switch (input.name) {
    case "hp":
      return (
        <svg {...commonProps}>
          <path d="M2 9.5a5.5 5.5 0 0 1 9.591-3.676.56.56 0 0 0 .818 0A5.49 5.49 0 0 1 22 9.5c0 2.29-1.5 4-3 5.5l-5.492 5.313a2 2 0 0 1-3 .019L5 15c-1.5-1.5-3-3.2-3-5.5" />
        </svg>
      );
    case "stress":
      return (
        <svg {...commonProps}>
          <path d="M12.409 5.824c-.702.792-1.15 1.496-1.415 2.166l2.153 2.156a.5.5 0 0 1 0 .707l-2.293 2.293a.5.5 0 0 0 0 .707L12 15" />
          <path d="M13.508 20.313a2 2 0 0 1-3 .019L5 15c-1.5-1.5-3-3.2-3-5.5a5.5 5.5 0 0 1 9.591-3.677.6.6 0 0 0 .818.001A5.5 5.5 0 0 1 22 9.5c0 2.29-1.5 4-3 5.5z" />
        </svg>
      );
    case "evasion":
      return (
        <svg {...commonProps}>
          <circle cx="12" cy="17" r="1" />
          <path d="M21 7v6h-6" />
          <path d="M3 17a9 9 0 0 1 9-9 9 9 0 0 1 6 2.3l3 2.7" />
        </svg>
      );
    case "armor":
      return (
        <svg {...commonProps}>
          <path d="M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z" />
        </svg>
      );
    case "hope":
      return (
        <svg {...commonProps}>
          <path d="M9.937 15.5A2 2 0 0 0 8.5 14.063l-6.135-1.582a.5.5 0 0 1 0-.962L8.5 9.936A2 2 0 0 0 9.937 8.5l1.582-6.135a.5.5 0 0 1 .963 0L14.063 8.5A2 2 0 0 0 15.5 9.937l6.135 1.581a.5.5 0 0 1 0 .964L15.5 14.063a2 2 0 0 0-1.437 1.437l-1.582 6.135a.5.5 0 0 1-.963 0z" />
          <path d="M20 3v4" />
          <path d="M22 5h-4" />
          <path d="M4 17v2" />
          <path d="M5 18H3" />
        </svg>
      );
  }
}
