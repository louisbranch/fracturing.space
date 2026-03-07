local scn = Scenario.new("environment_bruinen_ford_undertow")
local dh = scn:system("DAGGERHEART")

-- Model the Bruinen Ford undertow action and its consequences.
scn:campaign{
  name = "Environment Bruinen Ford Undertow",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- The river lashes out during a dangerous crossing.
scn:start_session("Bruinen Ford")
dh:gm_fear(1)

-- Example: spend Fear, Agility reaction, damage + movement + Vulnerable on failure.
-- River movement and conditional stress on success remain unresolved.
dh:gm_spend_fear(1):spotlight("Bruinen Ford")
dh:reaction_roll{ actor = "Frodo", trait = "agility", difficulty = 10, outcome = "failure_fear" }
dh:apply_reaction_outcome{
  on_failure = {
    {kind = "apply_condition", target = "Frodo", add = {"VULNERABLE"}, source = "undertow"},
  },
}

scn:end_session()

return scn
