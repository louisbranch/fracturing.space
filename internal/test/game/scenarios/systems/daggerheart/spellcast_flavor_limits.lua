local scene = Scenario.new("spellcast_flavor_limits")
local dh = scene:system("DAGGERHEART")

-- Capture the example where flavor doesn't grant extra effects.
scene:campaign{
  name = "Spellcast Flavor Limits",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "spellcast"
}

scene:pc("Gandalf")
dh:adversary("Saruman")

-- Flavoring a warding circle doesn't add extra damage.
scene:start_session("Flavor Limits")

-- Partial mapping: spellcast resolution and bounded damage profile are explicit.
-- Missing DSL: explicit `action.outcome.reject` emission for out-of-scope flavor effects.
dh:action_roll{ actor = "Gandalf", trait = "spellcast", difficulty = 12, outcome = "success_hope" }
dh:apply_roll_outcome{}
dh:attack{
  actor = "Gandalf",
  target = "Saruman",
  trait = "spellcast",
  difficulty = 0,
  outcome = "success_hope",
  damage_dice = {{count = 1, sides = 8}},
  damage_type = "magic"
}

scene:end_session()

return scene
