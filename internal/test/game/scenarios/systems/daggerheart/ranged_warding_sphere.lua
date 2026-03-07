local scn = Scenario.new("ranged_warding_sphere")
local dh = scn:system("DAGGERHEART")

-- Model the Saruman's Warding Sphere reaction.
scn:campaign{
  name = "Ranged Warding Sphere",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary"
}

scn:pc("Frodo")
dh:adversary("Saruman")

-- The wizard reacts to close-range damage with a magical backlash.
scn:start_session("Warding Sphere")

-- Example: when hit within Close range, the attacker takes 2d6 magic damage.
-- Missing DSL: apply reactive damage and cooldown on the reaction.
dh:attack{ actor = "Frodo", target = "Saruman", trait = "instinct", difficulty = 0, outcome = "hope", damage_type = "physical" }

scn:end_session()

return scn
