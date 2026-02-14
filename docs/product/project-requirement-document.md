# Project Requirement Document: Daggerheart Core Mechanics (Setting-Agnostic)

## Purpose
Define the core, setting-agnostic mechanics of the Daggerheart system for digital implementation. This document captures what the system must support and why, without prescribing technical design or content catalogs. It is intended to enable the same mechanics to run in any setting by swapping content data.

## Source
- Primary source: `https://www.daggerheart.com/srd/` (Daggerheart System Reference Document v1.0).

## Scope
- Core resolution loop and action outcomes.
- Character model, traits, resources, thresholds, and conditions.
- Combat and damage rules.
- Rest/downtime cadence and recovery.
- Leveling and progression mechanics.
- Content archetype schemas (class, subclass, ancestry/community, domain cards, equipment, items, adversaries, environments) without enumerated lists.
- GM mechanics (Fear economy, GM moves, adversary actions, countdowns).

## Out of Scope
- Campaign frames and setting-specific mechanics.
- Full content lists (ancestries, spells, items, adversaries, environments).
- Implementation details (APIs, storage, UI/UX).

## Design Intent
- Preserve deterministic resolution for mechanical outcomes while allowing explicit GM judgment where the rules call for it.
- Keep the system portable by treating content as data plugged into stable schemas.
- Maintain player-facing transparency for resources and constraints (Hope, Stress, HP, Armor, etc.).

## Core Loop Requirements
- The system must support a conversational spotlight model: a single active focus at a time (PC or GM), with the spotlight shifting based on outcomes or fiction.
- The system must support action rolls that resolve to one of five outcomes: success/failure crossed with Hope/Fear, plus critical success.
- The system must allow the GM to introduce consequences and shifts in scene state following results, especially on Fear outcomes.

## Resolution System

### Duality Dice
- **What:** Two distinct d12s represent Hope and Fear.
- **Why:** The higher die determines narrative tone (Hope vs Fear) while the sum determines success.
- **Requirements:**
  - Action roll total = Hope die + Fear die + modifiers.
  - Difficulty comparison determines success/failure.
  - Outcome category is defined by higher die (or matching dice for critical success).
  - Critical success counts as a Hope outcome and grants an additional benefit.

### Roll Types
- **Action Roll:** Standard resolution for risky or meaningful actions.
- **Trait Roll:** Action roll that fixes the trait used.
- **Spellcast Roll:** Action roll using the character’s Spellcast trait; can also be an attack roll if it deals damage.
- **Attack Roll:** Action roll targeting an opponent; difficulty equals target’s Evasion (PCs) or Difficulty (adversaries).
- **Damage Roll:** Dice pool defined by weapon/spell; resolves damage amount and type.
- **Reaction Roll:** Defensive roll to avoid or mitigate effects; does not generate Hope/Fear, does not trigger GM moves, and cannot be aided.
- **Group Action Roll:** One leader action roll; supporting PCs make reaction rolls; leader gains +1 per success and −1 per failure.
- **Tag Team Roll:** Once per session per player to initiate; spend 3 Hope; two PCs roll and select one result for both actions; combined damage on attack.

### Advantage/Disadvantage
- **What:** Adds/subtracts a d6 to/from the roll total.
- **Why:** Encodes situational leverage without changing base difficulty.
- **Requirements:**
  - Advantage and disadvantage cancel one-for-one in the same pool.
  - Help an Ally grants an advantage die rolled by the helper; only the highest helper die applies.

## Character Model

### Core Fields (PCs and NPCs where relevant)
- Identity: name, kind (PC/NPC), notes.
- Level (1–10) and tier (1–4).
- Traits: Agility, Strength, Finesse, Instinct, Presence, Knowledge (integer modifiers; can be negative).
- Evasion (integer target number).
- Hit Points (current, max slots).
- Stress (current, max slots).
- Hope (0–6).
- Proficiency (integer; affects damage dice count).
- Damage thresholds (Major, Severe).
- Armor score and Armor slots (current, max; max cap 12).
- Experiences: list of named modifiers.

### Resource Rules
- Hope is a capped metacurrency (0–6) used to power features, aid allies, and special roll interactions.
- Stress is a capped resource (default 6, can grow) used to power features or absorb consequences.
- HP marks damage severity; at 0 HP remaining, a death move is required.
- Armor Slots reduce damage severity by one threshold per slot spent.

### Conditions
- Standard conditions: Hidden, Restrained, Vulnerable.
- Conditions cannot stack unless explicitly stated.
- Temporary conditions can be cleared by actions or GM-defined triggers.

## Progression

### Levels and Tiers
- 10 levels, divided into four tiers.
- Level-up flow includes tier achievements, advancements, threshold increases, and new domain cards.

### Advancements
- The system must support advancement choices that alter traits, HP, Stress, Experiences, Evasion, Proficiency, subclass features, and multiclass access.
- Trait increases are gated by tier; tiers reset trait-increase eligibility.

### Multiclassing
- Unlocks at level 5.
- Adds another class’s feature and access to one of its domains, with reduced domain card level access.

## Combat and Damage

### Damage Resolution
- Damage amount is compared against Major/Severe thresholds to determine HP marks (1, 2, or 3).
- Optional massive damage rule supports 4 HP on extreme hits.
- Damage types: physical and magic; some effects are direct damage (no armor reduction).

### Critical Damage
- On critical success during an attack roll, add the maximum possible dice result to the rolled damage total.

### Resistance/Immunity
- Resistance halves damage before armor reduction.
- Immunity negates damage of that type.
- Mixed damage requires resistance or immunity to both types to apply.

## Rest and Downtime
- Rest types: short or long; three short rests in a row require a long rest next.
- Each rest allows two downtime moves.
- Rest triggers resource refresh for features with per-rest usage.
- GM gains Fear on party rests (scaled by rest type).

## Death and Scars
- When HP hits zero, player chooses a death move:
  - **Blaze of Glory:** one final automatic critical success action, then death.
  - **Avoid Death:** unconscious; may gain a scar based on Hope die vs level; scars remove Hope slots.
  - **Risk It All:** duality roll determines survival and recovery vs death.
- Loss of all Hope slots ends the character’s journey.

## Content Archetypes (Schema Requirements)

### Class
- Fields: name, starting Evasion, starting HP, starting items, class features, class Hope feature (cost 3 Hope), domain access (two domains).
- Why: Defines baseline combat profile and resource hooks.

### Subclass
- Fields: name, Spellcast trait, foundation/specialization/mastery features.
- Why: Establishes role specialization and the spellcasting stat.

### Heritage (Ancestry + Community)
- Each provides feature(s) that modify traits, resources, or mechanics.
- Mixed heritage must allow selecting features across two heritages.

### Domain Cards
- Fields: level, domain, type (ability/spell/grimoire), recall cost, feature text, usage limits.
- Loadout system: max 5 active cards; vault holds the remainder; recall cost to swap outside downtime.

### Experiences
- Fields: name, modifier (typically +2 at creation; can grow).
- Utilization: spend Hope to add experience modifiers to relevant rolls; some features allow alternative costs.

### Equipment
- **Weapons:** category (primary/secondary), tier, trait, range, damage dice, damage type, burden, feature.
- **Armor:** tier, base thresholds, base armor score, feature.
- Equip/unequip rules: stress cost outside downtime for weapons; armor changes thresholds and slots; max burden 2 hands.

### Items and Consumables
- Items must support rarity, stack limits (consumables capped at 5 each), and one-time or persistent effects.
- Gold uses abstract denominations: handfuls, bags, chests (10:1 conversion per tier).

### Adversaries
- Stat block fields: name, tier, type, description, motives/tactics, Difficulty, thresholds, HP, Stress, attack modifier, standard attack (range, damage), Experiences (bonuses), features (actions/reactions/passives), Fear features.
- Minion and Horde types require group-resolution rules.

### Environments
- Stat block fields: name, tier, type (exploration/social/traversal/event), impulses, Difficulty, potential adversaries, features, questions/prompts.
- Environment features may use countdowns and GM moves.

## GM Mechanics

### Fear Economy
- Fear is a GM resource (0–12) gained from player Fear outcomes and rests; spent to interrupt, make extra moves, or fuel adversary/environment features.

### GM Moves
- The system must support structured GM responses to action outcomes and explicit triggers (fear rolls, failed rolls, golden opportunities).

### Countdowns
- Support standard countdowns (tick per action roll), dynamic countdowns (tick based on roll outcomes), and variants (looping, increasing/decreasing, linked progress/consequence).

### GM Dice
- Adversary attack rolls use a single d20 plus attack modifier.
- Adversary advantage/disadvantage uses roll-high/roll-low on d20.

## Optional Rules (Must Be Togglable)
- Spotlight tracker tokens.
- Defined range grid conversions.
- Massive damage.
- Fate rolls.
- Underwater combat and breath countdowns.
- PC vs PC conflict resolution.
- Gold coins as a lower denomination.

## Determinism and Judgment Boundaries
- Dice outcomes and resource changes are deterministic.
- Difficulty selection, narrative consequences, and some feature triggers are GM judgment and must be explicitly modeled as inputs or GM-side actions.
- Content data must not hardcode setting-specific names, only structural roles and effects.

## Phase 2 Planning (Implementation Focus)
Phase 2 focuses on wiring the Phase 1 mechanics into event-driven gameplay flows, keeping internal mechanics authoritative and MCP deferred.

### Phase 2 Goals
- Convert mechanical helpers into event-driven state changes (damage, rest, downtime, loadout swaps).
- Define combat event payloads and projection handling for HP/Stress/Hope/Armor changes.
- Establish cadence events (rests, downtime moves) with GM Fear adjustments.
- Add outcome application flows for attacks, damage, and reactions.

### Phase 2 Deliverables
- Event payloads for damage application and mitigation (HP before/after, severity, armor spend).
- Rest/downtime events with refresh semantics and GM Fear changes.
- Ability usage events for stress/hope spend and recall swaps.
- Projection validation for new event types.

### Phase 2 Out of Scope
- Full content catalogs (items, adversaries, spells).
- MCP exposure and UI/UX.
- Campaign frames or setting-specific features.

## Example Behavior Snippets (Clarifying Only)
- **Action roll result:** If total ≥ Difficulty and Hope die > Fear die, success with Hope; gain 1 Hope.
- **Armor reduction:** Mark 1 Armor Slot to reduce damage severity by one threshold.
- **Loadout swap:** Move a domain card from vault to loadout outside downtime by paying its recall cost in Stress.

## Glossary (System-Agnostic)
- **Action Roll:** Core resolution check using the system’s primary dice and modifiers to determine success and tone.
- **Advantage/Disadvantage:** A single die modifier that raises or lowers a roll without changing the base difficulty.
- **Armor Slots:** Limited mitigation charges that reduce damage severity.
- **Character Traits:** Core numeric attributes that modify rolls.
- **Damage Thresholds:** Two breakpoints that convert damage totals into HP marks.
- **Difficulty:** Target number to meet or beat for success.
- **Domain Card:** Modular ability definition with level, type, cost, and effect text.
- **Evasion:** Defensive target number for attacks against a character.
- **Experience:** Contextual skill label with a numeric modifier that can be applied by spending a resource.
- **Fear (GM Resource):** GM-side currency that fuels complications and special adversary/environment effects.
- **Hope (Player Resource):** Player-side currency that fuels special actions, aid, and abilities.
- **Loadout/Vault:** Active vs inactive ability sets; swapping can incur a cost.
- **Progress Countdown:** A tracked meter that advances toward a positive outcome.
- **Consequence Countdown:** A tracked meter that advances toward a negative outcome.
- **Reaction Roll:** Defensive roll made to avoid or mitigate an incoming effect.
- **Rest (Short/Long):** Recovery windows with structured refresh and downtime choices.
- **Spotlight:** The current focus of action; determines who acts next.
- **Stress:** Endurance/strain resource that powers effects and absorbs consequences.
- **Tag Team Roll:** Coordinated multi-actor action that selects one result for both actors.
- **Tier:** Power band that gates content and scales thresholds.

## Domain Language Mapping (Daggerheart Terms → System-Agnostic)
- **Duality Dice → Tone Dice Pair**
- **Hope/Fear → Player Edge / GM Pressure**
- **Evasion → Defense Target**
- **Damage Thresholds → Severity Breakpoints**
- **Hit Points → Harm Capacity**
- **Stress → Strain Capacity**
- **Armor Slots → Mitigation Charges**
- **Domain Cards → Ability Modules**
- **Loadout/Vault → Active/Inactive Ability Set**
- **Spellcast Trait → Casting Attribute**
- **Experience → Context Skill Tag**
- **GM Move → Adjudication Move**
- **Countdown → Progress Track**
- **Adversary → Hostile Actor**
- **Environment → Scene State / Scene Hazards**
