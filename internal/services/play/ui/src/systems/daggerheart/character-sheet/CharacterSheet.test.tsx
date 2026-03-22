import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { CharacterSheet } from "./CharacterSheet";
import { characterSheetFixtures } from "./fixtures";

describe("CharacterSheet", () => {
  it("renders header with name, pronouns, heritage, subclass, and level", () => {
    render(<CharacterSheet character={characterSheetFixtures.full} />);

    expect(screen.getByRole("heading", { level: 2, name: "Mira" })).toBeInTheDocument();
    expect(screen.getByText("(she/her)")).toBeInTheDocument();
    expect(screen.getByText("Rogue / Nightwalker")).toBeInTheDocument();
    expect(screen.getByText("Human, Slyborne")).toBeInTheDocument();
    const levelSection = screen.getByText("Level").closest("div")!;
    expect(within(levelSection as HTMLElement).getByText("2")).toBeInTheDocument();
    expect(screen.getByText("Played by Mary")).toBeInTheDocument();
  });

  it("renders all six traits with values and skill names", () => {
    render(<CharacterSheet character={characterSheetFixtures.full} />);

    const traitsSection = screen.getByLabelText("Character traits and defense");
    const agilityCard = within(traitsSection).getByText("Agility").closest("[class*='rounded']")!;
    expect(within(agilityCard as HTMLElement).getByText("+2")).toBeInTheDocument();
    expect(within(agilityCard as HTMLElement).getByText("Sprint, Leap, Maneuver")).toBeInTheDocument();
    const knowledgeCard = within(traitsSection).getByText("Knowledge").closest("[class*='rounded']")!;
    expect(within(knowledgeCard as HTMLElement).getByText("-1")).toBeInTheDocument();
    expect(within(knowledgeCard as HTMLElement).getByText("Recall, Analyze, Comprehend")).toBeInTheDocument();
  });

  it("renders HP and stress tracks with current/max", () => {
    render(<CharacterSheet character={characterSheetFixtures.full} />);

    const health = screen.getByLabelText("Damage and health");
    expect(within(health).getByText("3/5")).toBeInTheDocument();
    expect(within(health).getByText("2/6")).toBeInTheDocument();
  });

  it("renders damage thresholds as a progression", () => {
    render(<CharacterSheet character={characterSheetFixtures.full} />);

    const health = screen.getByLabelText("Damage and health");
    expect(within(health).getByText("Minor")).toBeInTheDocument();
    expect(within(health).getByText("1")).toBeInTheDocument();
    expect(within(health).getByText("Major")).toBeInTheDocument();
    expect(within(health).getByText("5")).toBeInTheDocument();
    expect(within(health).getByText("Severe")).toBeInTheDocument();
  });

  it("renders defense stats: evasion and armor", () => {
    const { container } = render(
      <CharacterSheet
        character={{
          ...characterSheetFixtures.full,
          armor: { current: 4, max: 5 },
        }}
      />,
    );

    const defense = screen.getByLabelText("Defense");
    expect(within(defense).getByText("10")).toBeInTheDocument();
    expect(within(defense).getByText("5")).toBeInTheDocument();
    expect(within(defense).getByLabelText("Armor grid: 4 filled, 1 spent, 7 unavailable")).toBeInTheDocument();
    expect(container.querySelectorAll('[data-armor-state="filled"]')).toHaveLength(4);
    expect(container.querySelectorAll('[data-armor-state="spent"]')).toHaveLength(1);
    expect(container.querySelectorAll('[data-armor-state="muted"]')).toHaveLength(7);
  });

  it("renders spent armor slots as shield-off icons and preserves the 12-slot grid", () => {
    const { container } = render(<CharacterSheet character={characterSheetFixtures.damaged} />);

    const defense = screen.getByLabelText("Defense");
    expect(within(defense).getByText("4")).toBeInTheDocument();
    expect(within(defense).getByLabelText("Armor grid: 0 filled, 4 spent, 8 unavailable")).toBeInTheDocument();
    expect(container.querySelectorAll('[data-armor-state="filled"]')).toHaveLength(0);
    expect(container.querySelectorAll('[data-armor-state="spent"]')).toHaveLength(4);
    expect(container.querySelectorAll('[data-armor-state="muted"]')).toHaveLength(8);
  });

  it("renders the full 12-slot armor grid capacity for fortified characters", () => {
    const { container } = render(<CharacterSheet character={characterSheetFixtures.fortified} />);

    const defense = screen.getByLabelText("Defense");
    expect(within(defense).getByText("9")).toBeInTheDocument();
    expect(within(defense).getByText("12")).toBeInTheDocument();
    expect(within(defense).getByLabelText("Armor grid: 9 filled, 3 spent, 0 unavailable")).toBeInTheDocument();
    expect(container.querySelectorAll('[data-armor-state="filled"]')).toHaveLength(9);
    expect(container.querySelectorAll('[data-armor-state="spent"]')).toHaveLength(3);
    expect(container.querySelectorAll('[data-armor-state="muted"]')).toHaveLength(0);
  });

  it("renders hope section with diamonds and feature", () => {
    render(<CharacterSheet character={characterSheetFixtures.full} />);

    const hope = screen.getByLabelText("Hope");
    expect(within(hope).getByText("2/6")).toBeInTheDocument();
    expect(within(hope).getByText(/Rogue's Dodge/)).toBeInTheDocument();
  });

  it("renders equipment: weapon names, armor name, and proficiency", () => {
    render(<CharacterSheet character={characterSheetFixtures.full} />);

    const equipment = screen.getByLabelText("Equipment");
    expect(within(equipment).getByText("Sword")).toBeInTheDocument();
    expect(within(equipment).getByText("Dagger")).toBeInTheDocument();
    expect(within(equipment).getByText("Proficiency")).toBeInTheDocument();

    const armor = screen.getByLabelText("Active armor");
    expect(within(armor).getByText("Leather")).toBeInTheDocument();
  });

  it("renders experiences with modifiers", () => {
    render(<CharacterSheet character={characterSheetFixtures.full} />);

    const experiences = screen.getByLabelText("Experiences");
    expect(within(experiences).getByText("Wanderer")).toBeInTheDocument();
    expect(within(experiences).getByText("+2")).toBeInTheDocument();
    expect(within(experiences).getByText("Streetwise")).toBeInTheDocument();
    expect(within(experiences).getByText("Scholar")).toBeInTheDocument();
    expect(within(experiences).getByText("-1")).toBeInTheDocument();
  });

  it("renders expanded domain cards with feature text and preserves order", () => {
    const { container } = render(<CharacterSheet character={characterSheetFixtures.full} />);

    const domainCards = screen.getByLabelText("Domain cards");
    const scrollRegion = within(domainCards).getByLabelText("Domain card list");
    expect(within(domainCards).getByText("Vanishing Dodge")).toBeInTheDocument();
    expect(within(domainCards).getByText("Cloaking Blast")).toBeInTheDocument();
    expect(within(domainCards).getByText("Bolt Beacon")).toBeInTheDocument();
    expect(within(domainCards).getByText("Midnight")).toBeInTheDocument();
    expect(within(domainCards).getByText("Arcana")).toBeInTheDocument();
    expect(within(domainCards).getByText(/slip out of reach/)).toBeInTheDocument();
    expect(scrollRegion).toHaveClass("overflow-y-auto");

    const renderedCardIDs = Array.from(container.querySelectorAll("[data-domain-card-id]")).map((node) =>
      node.getAttribute("data-domain-card-id"),
    );
    expect(renderedCardIDs).toEqual([
      "domain_card.midnight-vanishing-dodge",
      "domain_card.arcana-cloaking-blast",
      "domain_card.splendor-bolt-beacon",
    ]);
  });

  it("renders gold values", () => {
    render(<CharacterSheet character={characterSheetFixtures.full} />);

    const gold = screen.getByLabelText("Gold");
    expect(within(gold).getByText("3")).toBeInTheDocument();
    expect(within(gold).getByText("1")).toBeInTheDocument();
    expect(within(gold).getByText("0")).toBeInTheDocument();
  });

  it("renders life state and condition badges", () => {
    render(<CharacterSheet character={characterSheetFixtures.damaged} />);

    const status = screen.getByLabelText("Status");
    const traitsAndDefense = screen.getByLabelText("Character traits and defense");
    expect(within(status).getByText("Unconscious")).toBeInTheDocument();
    expect(within(status).getByText("Frightened")).toBeInTheDocument();
    expect(within(status).getByText("Vulnerable")).toBeInTheDocument();
    expect(
      status.compareDocumentPosition(traitsAndDefense) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).not.toBe(0);
  });

  it("renders class feature text", () => {
    render(<CharacterSheet character={characterSheetFixtures.full} />);

    const feature = screen.getByLabelText("Class feature");
    expect(within(feature).getByText(/Sneak Attack/)).toBeInTheDocument();
  });
});
