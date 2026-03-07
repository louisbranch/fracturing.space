local scn = Scenario.new("fireball_orc_pack_multi")
local dh = scn:system("DAGGERHEART")

-- Capture the fireball example against multiple targets.
scn:campaign{
  name = "Fireball Orc Pack",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "battle"
}

scn:pc("Gandalf")
dh:adversary("Orc Pack A")
dh:adversary("Orc Pack B")

-- Gandalf casts Fireball to catch multiple orc packs at once.
scn:start_session("Fireball")

-- Example: one roll applied to multiple targets.
-- Missing DSL: assert per-target outcomes and damage tiers.
dh:multi_attack{
  actor = "Gandalf",
  targets = { "Orc Pack A", "Orc Pack B" },
  trait = "spellcast",
  difficulty = 0,
  outcome = "hope",
  damage_type = "magic",
  damage_dice = { { sides = 6, count = 2 } }
}

-- Close the session after the multi-target strike.
scn:end_session()

return scn
