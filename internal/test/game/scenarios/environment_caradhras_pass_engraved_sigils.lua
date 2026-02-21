local scene = Scenario.new("environment_caradhras_pass_engraved_sigils")

-- Model knowledge about sigils and advantage to dispel them.
scene:campaign{
  name = "Environment Caradhras Pass Engraved Sigils",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- The party studies the sigils carved into the pass.
scene:start_session("Engraved Sigils")

-- Missing DSL: apply advantage on dispel after critical knowledge success.
scene:action_roll{ actor = "Frodo", trait = "knowledge", difficulty = 15, outcome = "hope" }
scene:apply_roll_outcome{}

scene:end_session()

return scene
