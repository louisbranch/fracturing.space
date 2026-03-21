import { StatIcon } from "../character-card/CharacterCard";
import type {
  CharacterSheetProps,
  DaggerheartArmor,
  DaggerheartCharacterSheetData,
  DaggerheartDomainCard,
  DaggerheartExperience,
  DaggerheartGold,
  DaggerheartTrait,
  DaggerheartWeapon,
} from "./contract";

// CharacterSheet is a read-only, display-oriented view of a Daggerheart
// character that follows the official paper sheet layout while using DaisyUI
// components for a better digital presentation.
export function CharacterSheet({ character }: CharacterSheetProps) {
  return (
    <article className="character-sheet">
      <SheetIdentity character={character} />

      {/* Row 1: Defense (25%) | Traits (75%) — mirrors PDF silhouette + trait columns */}
      <div className="grid grid-cols-[1fr_3fr] border-b border-base-300/60" aria-label="Character traits and defense">
        <SheetDefense character={character} />
        <SheetTraits traits={character.traits} />
      </div>

      {/* Row 2: Left (33%) | Right (66%) — mirrors PDF body columns */}
      <div className="grid sm:grid-cols-[1fr_2fr]">
        <div className="space-y-4 border-b border-base-300/60 p-4 sm:border-b-0 sm:border-r sm:p-5">
          <SheetDamageHealth character={character} />
          <SheetHope character={character} />
          <SheetExperiences experiences={character.experiences} />
          <SheetGold gold={character.gold} />
          <SheetClassFeature feature={character.classFeature} />
        </div>

        <div className="space-y-4 p-4 sm:p-5">
          <SheetActiveWeapons character={character} />
          <SheetActiveArmor armor={character.activeArmor} />
          <SheetDomainCards domainCards={character.domainCards} />
        </div>
      </div>

      {/* Full-width collapsed narrative sections */}
      <SheetNarrative
        description={character.description}
        background={character.background}
        connections={character.connections}
      />

      <SheetStatus character={character} />
    </article>
  );
}

// ---------------------------------------------------------------------------
// Identity — name, pronouns, heritage, subclass, level, portrait
// ---------------------------------------------------------------------------

function SheetIdentity({ character }: { character: DaggerheartCharacterSheetData }) {
  const classSummary = summarizeClass(character);
  const heritageSummary = summarizeHeritage(character);

  return (
    <div className="flex gap-4 border-b border-base-300/60 px-4 py-3 sm:px-5">
      <SheetPortrait character={character} />

      <div className="flex flex-1 flex-wrap items-start justify-between gap-x-6 gap-y-2">
        <div className="space-y-1">
          <div className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
            <h2 className="font-display text-2xl text-base-content sm:text-3xl">{character.name}</h2>
            {character.pronouns ? <span className="text-sm text-base-content/60">({character.pronouns})</span> : null}
          </div>
          <div className="flex flex-wrap gap-x-4 gap-y-0.5 text-sm">
            {heritageSummary ? (
              <div>
                <span className="sheet-field-label">Heritage </span>
                <span className="text-base-content/80">{heritageSummary}</span>
              </div>
            ) : null}
            {classSummary ? (
              <div>
                <span className="sheet-field-label">Subclass </span>
                <span className="text-base-content/80">{classSummary}</span>
              </div>
            ) : null}
          </div>
          {character.controller ? (
            <p className="text-xs text-base-content/40">Played by {character.controller}</p>
          ) : null}
        </div>

        {character.level !== undefined ? (
          <div className="text-center">
            <p className="sheet-field-label">Level</p>
            <p className="font-display text-2xl text-base-content">{character.level}</p>
          </div>
        ) : null}
      </div>
    </div>
  );
}

function SheetPortrait({ character }: { character: DaggerheartCharacterSheetData }) {
  if (character.portrait.src) {
    return (
      <figure className="hidden shrink-0 overflow-hidden rounded-box sm:block">
        <img
          alt={character.portrait.alt}
          className="h-20 w-14 object-cover"
          src={character.portrait.src}
        />
      </figure>
    );
  }

  const initials = character.name
    .split(/\s+/)
    .filter(Boolean)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .slice(0, 2)
    .join("");

  return (
    <figure className="hidden shrink-0 overflow-hidden rounded-box sm:block">
      <div
        aria-label={character.portrait.alt}
        className="flex h-20 w-14 items-center justify-center bg-base-300/60"
        role="img"
      >
        <span aria-hidden="true" className="font-display text-xl text-base-content/50">
          {initials || "?"}
        </span>
      </div>
    </figure>
  );
}

// ---------------------------------------------------------------------------
// Defense — left 25% of the stats row
// ---------------------------------------------------------------------------

function SheetDefense({ character }: { character: DaggerheartCharacterSheetData }) {
  return (
    <section
      aria-label="Defense"
      className="flex items-center justify-center gap-3 border-r border-base-300/60 p-4"
    >
      {character.evasion !== undefined ? (
        <div className="flex flex-col items-center gap-0.5">
          <StatIcon name="evasion" />
          <span className="font-display text-xl font-bold text-base-content">{character.evasion}</span>
          <span className="sheet-field-label">Evasion</span>
        </div>
      ) : null}
      {character.armor !== undefined ? (
        <div className="flex flex-col items-center gap-0.5">
          <StatIcon name="armor" />
          <span className="font-display text-xl font-bold text-base-content">
            {character.armor.current}/{character.armor.max}
          </span>
          <span className="sheet-field-label">Armor</span>
        </div>
      ) : null}
    </section>
  );
}

// ---------------------------------------------------------------------------
// Traits — right 75% of the stats row
// ---------------------------------------------------------------------------

function SheetTraits({ traits }: { traits?: DaggerheartTrait[] }) {
  if (!traits || traits.length === 0) {
    return (
      <section aria-label="Character traits" className="p-4" />
    );
  }

  return (
    <section aria-label="Character traits" className="p-4">
      <div className="grid grid-cols-3 gap-1.5 sm:grid-cols-6">
        {traits.map((trait) => (
          <div key={trait.abbreviation} className="rounded border border-base-300/50 bg-base-200/25 px-2 py-1.5 text-center">
            <p className="text-[0.6rem] font-bold uppercase tracking-wider text-base-content/50">
              {trait.name}
            </p>
            <p className="font-display text-xl font-bold leading-tight text-base-content">
              {trait.value >= 0 ? `+${trait.value}` : trait.value}
            </p>
            <p className="mt-0.5 text-[0.6rem] leading-tight text-base-content/35">
              {(trait.skills ?? []).join(", ")}
            </p>
          </div>
        ))}
      </div>
    </section>
  );
}

// ---------------------------------------------------------------------------
// Damage & Health
// ---------------------------------------------------------------------------

function SheetDamageHealth({ character }: { character: DaggerheartCharacterSheetData }) {
  const hasThresholds = character.majorThreshold !== undefined || character.severeThreshold !== undefined;
  const hasHP = character.hp !== undefined;
  const hasStress = character.stress !== undefined;

  if (!hasThresholds && !hasHP && !hasStress) {
    return null;
  }

  return (
    <section aria-label="Damage and health" className="space-y-3">
      <SectionHeader>Damage &amp; Health</SectionHeader>

      {hasThresholds ? (
        <div className="flex items-stretch overflow-hidden rounded border border-base-300/50 text-sm">
          <ChevronSegment bg="bg-base-200/40" position="first" grow>
            <span className="text-center">
              <span className="block text-base-content/70">Minor</span>
              <span className="block text-[0.55rem] text-base-content/30">Mark 1 HP</span>
            </span>
          </ChevronSegment>
          <ChevronSegment bg="bg-base-content/10" position="middle">
            <span className="font-display font-bold text-base-content">1</span>
          </ChevronSegment>
          {character.majorThreshold !== undefined ? (
            <>
              <ChevronSegment bg="bg-base-200/40" position="middle" grow>
                <span className="text-center">
                  <span className="block text-base-content/70">Major</span>
                  <span className="block text-[0.55rem] text-base-content/30">Mark 2 HP</span>
                </span>
              </ChevronSegment>
              <ChevronSegment bg="bg-base-content/10" position="middle">
                <span className="font-display font-bold text-base-content">{character.majorThreshold}</span>
              </ChevronSegment>
            </>
          ) : null}
          {character.severeThreshold !== undefined ? (
            <ChevronSegment bg="bg-base-200/40" position="last" grow>
              <span className="text-center">
                <span className="block text-base-content/70">Severe</span>
                <span className="block text-[0.55rem] text-base-content/30">Mark 3 HP</span>
              </span>
            </ChevronSegment>
          ) : null}
        </div>
      ) : null}

      {hasHP ? (
        <div className="flex items-center gap-2">
          <span className="flex items-center gap-1 text-base-content/60">
            <StatIcon name="hp" />
            <span className="sheet-field-label">HP</span>
          </span>
          <div className="flex flex-wrap gap-0.5">
            {Array.from({ length: character.hp!.max }, (_, i) => (
              <span
                key={i}
                className={`inline-block h-3.5 w-3.5 rounded-sm border ${
                  i < character.hp!.current
                    ? "border-success/50 bg-success/30"
                    : "border-base-content/15 bg-transparent"
                }`}
              />
            ))}
          </div>
          <span className="text-xs text-base-content/45">{character.hp!.current}/{character.hp!.max}</span>
        </div>
      ) : null}

      {hasStress ? (
        <div className="flex items-center gap-2">
          <span className="flex items-center gap-1 text-base-content/60">
            <StatIcon name="stress" />
            <span className="sheet-field-label">Stress</span>
          </span>
          <div className="flex flex-wrap gap-0.5">
            {Array.from({ length: character.stress!.max }, (_, i) => (
              <span
                key={i}
                className={`inline-block h-3.5 w-3.5 rounded-sm border ${
                  i < character.stress!.current
                    ? "border-warning/50 bg-warning/30"
                    : "border-base-content/15 bg-transparent"
                }`}
              />
            ))}
          </div>
          <span className="text-xs text-base-content/45">{character.stress!.current}/{character.stress!.max}</span>
        </div>
      ) : null}
    </section>
  );
}

// ---------------------------------------------------------------------------
// Hope
// ---------------------------------------------------------------------------

function SheetHope({ character }: { character: DaggerheartCharacterSheetData }) {
  if (!character.hope) {
    return null;
  }

  return (
    <section aria-label="Hope" className="space-y-2">
      <SectionHeader>Hope</SectionHeader>
      <p className="text-[0.6rem] text-base-content/30">
        Spend a Hope to use an experience or help an ally.
      </p>
      <div className="flex items-center gap-2">
        <StatIcon name="hope" />
        <div className="flex gap-0.5" aria-label={`${character.hope.current} of ${character.hope.max} hope`}>
          {Array.from({ length: character.hope.max }, (_, i) => (
            <span
              key={i}
              className={`inline-block text-xl leading-none ${
                i < character.hope!.current ? "text-amber-400" : "text-base-content/15"
              }`}
              aria-hidden="true"
            >
              ◆
            </span>
          ))}
        </div>
        <span className="text-xs text-base-content/45">{character.hope.current}/{character.hope.max}</span>
      </div>
      {character.hopeFeature ? (
        <div>
          <p className="sheet-field-label">Hope Feature</p>
          <p className="text-sm text-base-content/70">{character.hopeFeature}</p>
        </div>
      ) : null}
    </section>
  );
}

// ---------------------------------------------------------------------------
// Experiences
// ---------------------------------------------------------------------------

function SheetExperiences({ experiences }: { experiences?: DaggerheartExperience[] }) {
  if (!experiences || experiences.length === 0) {
    return null;
  }

  return (
    <section aria-label="Experiences" className="space-y-2">
      <SectionHeader>Experience</SectionHeader>
      <div className="flex flex-wrap gap-1.5">
        {experiences.map((exp) => (
          <span key={exp.name} className="badge badge-outline badge-sm">
            {exp.name}
            {exp.modifier !== undefined ? (
              <span className="ml-1 text-base-content/55">
                {exp.modifier >= 0 ? `+${exp.modifier}` : exp.modifier}
              </span>
            ) : null}
          </span>
        ))}
      </div>
    </section>
  );
}

// ---------------------------------------------------------------------------
// Gold
// ---------------------------------------------------------------------------

function SheetGold({ gold }: { gold?: DaggerheartGold }) {
  if (!gold) {
    return null;
  }

  return (
    <section aria-label="Gold" className="space-y-2">
      <SectionHeader>Gold</SectionHeader>
      <div className="flex gap-4 text-sm">
        <div>
          <span className="sheet-field-label">Handfuls </span>
          <span className="font-medium text-base-content">{gold.handfuls}</span>
        </div>
        <div>
          <span className="sheet-field-label">Bags </span>
          <span className="font-medium text-base-content">{gold.bags}</span>
        </div>
        <div>
          <span className="sheet-field-label">Chests </span>
          <span className="font-medium text-base-content">{gold.chests}</span>
        </div>
      </div>
    </section>
  );
}

// ---------------------------------------------------------------------------
// Class Feature
// ---------------------------------------------------------------------------

function SheetClassFeature({ feature }: { feature?: string }) {
  if (!feature) {
    return null;
  }

  return (
    <section aria-label="Class feature" className="space-y-2">
      <SectionHeader>Class Feature</SectionHeader>
      <p className="text-sm leading-relaxed text-base-content/70">{feature}</p>
    </section>
  );
}

// ---------------------------------------------------------------------------
// Active Weapons
// ---------------------------------------------------------------------------

function SheetActiveWeapons({ character }: { character: DaggerheartCharacterSheetData }) {
  const hasWeapons = character.primaryWeapon || character.secondaryWeapon;

  if (!hasWeapons) {
    return null;
  }

  return (
    <section aria-label="Equipment" className="space-y-3">
      <SectionHeader>Active Weapons</SectionHeader>

      {character.proficiency !== undefined ? (
        <div className="flex items-center gap-2">
          <span className="sheet-field-label">Proficiency</span>
          <div className="flex gap-0.5">
            {Array.from({ length: 6 }, (_, i) => (
              <span
                key={i}
                className={`inline-block text-xs ${
                  i < character.proficiency! ? "text-base-content/70" : "text-base-content/15"
                }`}
              >
                ●
              </span>
            ))}
          </div>
        </div>
      ) : null}

      {character.primaryWeapon ? <WeaponBlock weapon={character.primaryWeapon} label="Primary" /> : null}
      {character.secondaryWeapon ? <WeaponBlock weapon={character.secondaryWeapon} label="Secondary" /> : null}
    </section>
  );
}

function WeaponBlock({ weapon, label }: { weapon: DaggerheartWeapon; label: string }) {
  return (
    <div className="space-y-1 rounded border border-base-300/40 bg-base-200/20 px-3 py-2">
      <p className="text-[0.65rem] font-bold uppercase tracking-wider text-base-content/45">{label}</p>
      <div className="flex flex-wrap gap-x-4 gap-y-0.5 text-sm">
        <div>
          <span className="sheet-field-label">Name </span>
          <span className="text-base-content">{weapon.name}</span>
        </div>
        {weapon.trait || weapon.range ? (
          <div>
            <span className="sheet-field-label">Trait &amp; Range </span>
            <span className="text-base-content/75">
              {[weapon.trait, weapon.range].filter(Boolean).join(" / ")}
            </span>
          </div>
        ) : null}
        {weapon.damageDice ? (
          <div>
            <span className="sheet-field-label">Damage </span>
            <span className="text-base-content/75">
              {weapon.damageDice}{weapon.damageType ? ` ${weapon.damageType}` : ""}
            </span>
          </div>
        ) : null}
      </div>
      {weapon.feature ? (
        <div className="text-sm">
          <span className="sheet-field-label">Feature </span>
          <span className="text-base-content/60">{weapon.feature}</span>
        </div>
      ) : null}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Active Armor
// ---------------------------------------------------------------------------

function SheetActiveArmor({ armor }: { armor?: DaggerheartArmor }) {
  if (!armor) {
    return null;
  }

  return (
    <section aria-label="Active armor" className="space-y-2">
      <SectionHeader>Active Armor</SectionHeader>
      <div className="space-y-1 rounded border border-base-300/40 bg-base-200/20 px-3 py-2">
        <div className="flex flex-wrap gap-x-4 text-sm">
          <div>
            <span className="sheet-field-label">Name </span>
            <span className="text-base-content">{armor.name}</span>
          </div>
          {armor.baseThresholds !== undefined ? (
            <div>
              <span className="sheet-field-label">Base Thresholds </span>
              <span className="text-base-content/75">+{armor.baseThresholds}</span>
            </div>
          ) : null}
          {armor.baseScore !== undefined ? (
            <div>
              <span className="sheet-field-label">Base Score </span>
              <span className="text-base-content/75">{armor.baseScore}</span>
            </div>
          ) : null}
        </div>
        {armor.feature ? (
          <div className="text-sm">
            <span className="sheet-field-label">Feature </span>
            <span className="text-base-content/60">{armor.feature}</span>
          </div>
        ) : null}
      </div>
    </section>
  );
}

// ---------------------------------------------------------------------------
// Domain Cards
// ---------------------------------------------------------------------------

function SheetDomainCards({ domainCards }: { domainCards?: DaggerheartDomainCard[] }) {
  if (!domainCards || domainCards.length === 0) {
    return null;
  }

  const grouped = new Map<string, string[]>();
  for (const card of domainCards) {
    const domain = card.domain ?? "Other";
    const existing = grouped.get(domain) ?? [];
    existing.push(card.name);
    grouped.set(domain, existing);
  }

  return (
    <section aria-label="Domain cards" className="space-y-2">
      <SectionHeader>Domain Cards</SectionHeader>
      <div className="space-y-1.5">
        {Array.from(grouped.entries()).map(([domain, names]) => (
          <div key={domain} className="flex flex-wrap items-center gap-1.5">
            <span className="text-xs font-medium text-base-content/45">{domain}:</span>
            {names.map((name) => (
              <span key={name} className="badge badge-outline badge-sm">{name}</span>
            ))}
          </div>
        ))}
      </div>
    </section>
  );
}

// ---------------------------------------------------------------------------
// Narrative — full-width collapsed sections at the bottom
// ---------------------------------------------------------------------------

function SheetNarrative({ description, background, connections }: {
  description?: string;
  background?: string;
  connections?: string;
}) {
  if (!description && !background && !connections) {
    return null;
  }

  return (
    <section aria-label="Narrative" className="space-y-1.5 border-t border-base-300/60 px-4 py-4 sm:px-5">
      {description ? (
        <div className="collapse collapse-arrow rounded border border-base-300/40 bg-base-200/20">
          <input type="checkbox" />
          <div className="collapse-title py-2 text-sm font-medium">Description</div>
          <div className="collapse-content text-sm leading-relaxed text-base-content/65">
            <p>{description}</p>
          </div>
        </div>
      ) : null}
      {background ? (
        <div className="collapse collapse-arrow rounded border border-base-300/40 bg-base-200/20">
          <input type="checkbox" />
          <div className="collapse-title py-2 text-sm font-medium">Background</div>
          <div className="collapse-content text-sm leading-relaxed text-base-content/65">
            <p>{background}</p>
          </div>
        </div>
      ) : null}
      {connections ? (
        <div className="collapse collapse-arrow rounded border border-base-300/40 bg-base-200/20">
          <input type="checkbox" />
          <div className="collapse-title py-2 text-sm font-medium">Connections</div>
          <div className="collapse-content text-sm leading-relaxed text-base-content/65">
            <p>{connections}</p>
          </div>
        </div>
      ) : null}
    </section>
  );
}

// ---------------------------------------------------------------------------
// Status (life state + conditions) — conditional footer
// ---------------------------------------------------------------------------

function SheetStatus({ character }: { character: DaggerheartCharacterSheetData }) {
  const hasLifeState = character.lifeState && character.lifeState !== "alive";
  const hasConditions = character.conditions && character.conditions.length > 0;

  if (!hasLifeState && !hasConditions) {
    return null;
  }

  const lifeStateBadgeClass: Record<string, string> = {
    unconscious: "badge-warning",
    blaze_of_glory: "badge-error",
    dead: "badge-neutral",
  };

  return (
    <section
      aria-label="Status"
      className="flex flex-wrap items-center gap-2 border-t border-base-300/60 px-4 py-3 sm:px-5"
    >
      <span className="sheet-field-label">Status</span>
      {hasLifeState ? (
        <span className={`badge ${lifeStateBadgeClass[character.lifeState!] ?? "badge-neutral"} badge-sm`}>
          {formatLifeState(character.lifeState!)}
        </span>
      ) : null}
      {hasConditions
        ? character.conditions!.map((condition) => (
            <span key={condition} className="badge badge-outline badge-warning badge-sm">
              {condition}
            </span>
          ))
        : null}
    </section>
  );
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

const chevronClip = {
  first: "polygon(0 0, calc(100% - 7px) 0, 100% 50%, calc(100% - 7px) 100%, 0 100%)",
  middle: "polygon(0 0, calc(100% - 7px) 0, 100% 50%, calc(100% - 7px) 100%, 0 100%, 7px 50%)",
  last: "polygon(0 0, 100% 0, 100% 100%, 0 100%, 7px 50%)",
};

function ChevronSegment({ children, bg, position, grow }: {
  children: React.ReactNode;
  bg: string;
  position: "first" | "middle" | "last";
  grow?: boolean;
}) {
  return (
    <span
      className={`flex items-center justify-center px-3 py-1.5 ${bg} ${grow ? "flex-1" : ""}`}
      style={{ clipPath: chevronClip[position], marginLeft: position === "first" ? 0 : "-3px" }}
    >
      {children}
    </span>
  );
}

function SectionHeader({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex items-center gap-2">
      <div className="h-px flex-1 bg-base-content/15" />
      <h3 className="text-[0.65rem] font-bold uppercase tracking-[0.2em] text-base-content/55">{children}</h3>
      <div className="h-px flex-1 bg-base-content/15" />
    </div>
  );
}

function summarizeClass(character: DaggerheartCharacterSheetData): string {
  if (!character.className) {
    return "";
  }
  if (!character.subclassName) {
    return character.className;
  }
  return `${character.className} / ${character.subclassName}`;
}

function summarizeHeritage(character: DaggerheartCharacterSheetData): string {
  if (!character.ancestryName) {
    return "";
  }
  if (!character.communityName) {
    return character.ancestryName;
  }
  return `${character.ancestryName}, ${character.communityName}`;
}

function formatLifeState(state: string): string {
  return state
    .split("_")
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(" ");
}
