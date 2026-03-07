local scn = Scenario.new("environment_caradhras_pass_engraved_sigils")
local dh = scn:system("DAGGERHEART")

-- Model knowledge about sigils and advantage to dispel them.
scn:campaign{
  name = "Environment Caradhras Pass Engraved Sigils",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- The party studies the sigils carved into the pass.
scn:start_session("Engraved Sigils")

-- Missing DSL: apply advantage on dispel after critical knowledge success.
dh:action_roll{ actor = "Frodo", trait = "knowledge", difficulty = 15, outcome = "hope" }
dh:apply_roll_outcome{}

scn:end_session()

return scn
