local scn = Scenario.new("improvised_fear_move_bandit_chain")
local dh = scn:system("DAGGERHEART")

-- Capture the bandit fear-move chain with multiple spotlights.
scn:campaign{
  name = "Improvised Fear Move Orc Chain",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "gm_fear"
}

scn:pc("Sam", { armor = 0 })
scn:pc("Frodo", { armor = 0 })
dh:adversary("Orc Captain")
dh:adversary("Orc Raider")
dh:adversary("Orc Minions")

-- The GM spends Fear to escalate the bandit ambush.
scn:start_session("Orc Ambush")
dh:gm_fear(5)

-- Example: spotlight Orc Captain with a sudden ambush move.
dh:gm_spend_fear(1):spotlight("Orc Captain")

-- Example: spotlight Orc Raider and swing with a multi-target action.
-- Partial mapping: fear-spend spotlight with deterministic per-target attacks.
-- Missing DSL: one shared roll that fans out to all targets in range.
dh:gm_spend_fear(1):spotlight("Orc Raider")
dh:adversary_attack{ actor = "Orc Raider", target = "Sam", difficulty = 0, seed = 51, damage_type = "physical" }
dh:adversary_attack{ actor = "Orc Raider", target = "Frodo", difficulty = 0, seed = 51, damage_type = "physical" }

-- Example: spotlight minions and spend Fear for a group attack.
-- Partial mapping: per-target minion attacks are explicit.
-- Missing DSL: combined group-attack damage as a single shared source.
dh:gm_spend_fear(1):spotlight("Orc Minions")
dh:adversary_attack{ actor = "Orc Minions", target = "Sam", difficulty = 0, seed = 52, damage_type = "physical" }
dh:adversary_attack{ actor = "Orc Minions", target = "Frodo", difficulty = 0, seed = 52, damage_type = "physical" }

-- Close the session after the bandit chain.
scn:end_session()

return scn
