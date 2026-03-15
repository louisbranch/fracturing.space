import type {
  CharacterCardProps,
  CharacterCardVariant,
  DaggerheartCharacterEquipment,
  DaggerheartCharacterSummary,
  DaggerheartCharacterTraits,
} from "./contract";

type LabeledValue = {
  label: string;
  value: string;
};

// CharacterCard is the stable Daggerheart card contract that future runtime
// adapters can target without depending on one permanent implementation.
export function CharacterCard({ character, variant }: CharacterCardProps) {
  const identityRows = buildIdentityRows(character, variant);
  const summary = character.daggerheart?.summary;
  const creationSummary = character.daggerheart?.creationSummary;
  const classSummary = summarizeClass(summary);
  const heritageSummary = summarizeHeritage(summary);
  const pronouns = character.identity?.pronouns?.trim();
  const statRows = buildStatRows(summary);
  const hopeValue = formatTrackValue(summary?.hope);
  const featureValue = summary?.feature?.trim();
  const traitBadges = buildTraitBadges(creationSummary?.traits);
  const equipmentBadges = buildEquipmentBadges(creationSummary?.equipment);
  const experiences = creationSummary?.experiences ?? [];
  const domainCards = creationSummary?.domainCards ?? [];

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
        <figure className={`character-card-media ${variant === "full" ? "character-card-media-fixed" : ""}`}>
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

            {identityRows.length > 0 ? (
              <dl className="space-y-2 text-sm" aria-label="Character identity">
                {identityRows.map((row) => (
                  <div key={`${row.label}-${row.value}`} className="flex flex-wrap gap-2">
                    <dt className="text-base-content/55">{row.label}:</dt>
                    <dd>{row.value}</dd>
                  </div>
                ))}
              </dl>
            ) : null}

            {statRows.length > 0 || hopeValue || featureValue ? (
              <div className="space-y-2">
                {statRows.length > 0 ? (
                  <section className="grid gap-2 text-sm text-base-content/85 sm:grid-cols-2" aria-label="Character statistics">
                    {statRows.map((row) => (
                      <div key={row.label} className="flex items-center justify-between gap-3 rounded-box border border-base-300/70 bg-base-200/45 px-3 py-2">
                        <span className="text-base-content/60">{row.label}</span>
                        <span className="font-medium text-base-content">{row.value}</span>
                      </div>
                    ))}
                  </section>
                ) : null}

                {hopeValue || featureValue ? (
                  <section aria-label="Character feature summary">
                    <div className="grid gap-2 text-sm text-base-content/85 sm:grid-cols-2">
                      {hopeValue ? (
                        <div className="flex items-center justify-between gap-3 rounded-box border border-base-300/70 bg-base-200/45 px-3 py-2">
                          <span className="text-base-content/60">Hope</span>
                          <span className="font-medium text-base-content">{hopeValue}</span>
                        </div>
                      ) : null}
                      {featureValue ? (
                        <div className="flex items-center justify-between gap-3 rounded-box border border-base-300/70 bg-base-200/45 px-3 py-2">
                          <span className="font-medium text-base-content/60">{featureValue}</span>
                        </div>
                      ) : null}
                    </div>
                  </section>
                ) : null}
              </div>
            ) : null}
          </header>

          {variant === "full" && hasFullContent(traitBadges, equipmentBadges, experiences, domainCards) ? (
            <section className="border-t border-base-300/80 pt-4" aria-label="Daggerheart full info">
              <div className="space-y-4">
                <SummaryBadgeGroup label="Traits" items={traitBadges} />
                <SummaryBadgeGroup label="Equipment" items={equipmentBadges} />
                <SummaryBadgeGroup
                  label="Experiences"
                  items={experiences.map((experience) => formatExperience(experience.name, experience.modifier))}
                />
                <SummaryBadgeGroup label="Domain Cards" items={domainCards} />
              </div>
            </section>
          ) : null}
        </div>
      </div>
    </article>
  );
}

function SummaryBadgeGroup(input: { label: string; items: string[] }) {
  if (input.items.length === 0) {
    return null;
  }

  return (
    <div className="space-y-2">
      <p className="character-card-section-label">{input.label}</p>
      <div className="flex flex-wrap gap-2">
        {input.items.map((item) => (
          <span key={`${input.label}-${item}`} className="badge badge-outline badge-sm">
            {item}
          </span>
        ))}
      </div>
    </div>
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

// characterCardVariants documents the supported display densities for stories,
// tests, and future runtime callers without duplicating labels in multiple files.
export const characterCardVariants: Array<{
  id: CharacterCardVariant;
  title: string;
  purpose: string;
}> = [
  {
    id: "portrait",
    title: "Portrait Only",
    purpose: "Spotlight or modal reveal with no supporting metadata on screen.",
  },
  {
    id: "basic",
    title: "Portrait + Basic Info",
    purpose: "Single-character card using the web campaign-card information hierarchy.",
  },
  {
    id: "full",
    title: "Portrait + Full Info",
    purpose: "Same card header plus the Daggerheart detail summary from the web character page.",
  },
];

function buildIdentityRows(character: CharacterCardProps["character"], variant: CharacterCardVariant): LabeledValue[] {
  const identity = character.identity;
  if (!identity) {
    return [];
  }

  const rows = [
    asLabeledValue("Kind", identity.kind),
    asLabeledValue("Controller", identity.controller),
    asLabeledValue("Aliases", joinValues(identity.aliases)),
  ].filter(isPresent);

  if (variant === "basic") {
    return [];
  }

  return rows;
}

function buildStatRows(summary: DaggerheartCharacterSummary | undefined): LabeledValue[] {
  return [
    asLabeledValue("HP", formatTrackValue(summary?.hp)),
    asLabeledValue("Stress", formatTrackValue(summary?.stress)),
    asLabeledValue("Evasion", summary?.evasion !== undefined ? String(summary.evasion) : undefined),
    asLabeledValue("Armor", formatTrackValue(summary?.armor)),
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

function buildEquipmentBadges(equipment: DaggerheartCharacterEquipment | undefined): string[] {
  if (!equipment) {
    return [];
  }

  return [equipment.primaryWeapon, equipment.secondaryWeapon, equipment.armor, equipment.potion].filter(isPresent);
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

function formatExperience(name: string, modifier: string | undefined): string {
  const trimmedModifier = modifier?.trim();
  if (!trimmedModifier) {
    return name;
  }
  return `${name} ${trimmedModifier.startsWith("+") || trimmedModifier.startsWith("-") ? trimmedModifier : `+${trimmedModifier}`}`;
}

function formatTrackValue(value: DaggerheartCharacterSummary["hp"] | undefined): string | undefined {
  if (!value) {
    return undefined;
  }
  return `${value.current}/${value.max}`;
}

function joinValues(values: Array<string | undefined> | undefined, separator = ", "): string {
  return values?.map((value) => value?.trim()).filter(isPresent).join(separator) ?? "";
}

function hasFullContent(
  traitBadges: string[],
  equipmentBadges: string[],
  experiences: Array<{ name: string; modifier?: string }>,
  domainCards: string[],
): boolean {
  return traitBadges.length > 0 || equipmentBadges.length > 0 || experiences.length > 0 || domainCards.length > 0;
}

function asLabeledValue(label: string, value: string | undefined): LabeledValue | undefined {
  const trimmed = value?.trim();
  if (!trimmed) {
    return undefined;
  }
  return { label, value: trimmed };
}

function isPresent<T>(value: T | null | undefined | ""): value is T {
  return value !== null && value !== undefined && value !== "";
}
