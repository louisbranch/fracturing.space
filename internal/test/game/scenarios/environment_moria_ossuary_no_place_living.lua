local scene = Scenario.new("environment_moria_ossuary_no_place_living")

-- Model the added Hope cost to clear HP in the ossuary.
scene:campaign{
  name = "Environment Ossuary No Place for the Living",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo", { hope = 2 })

-- Healing actions cost extra Hope here.
scene:start_session("No Place for the Living")

-- Build enough Hope for the explicit spend step.
scene:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 10, outcome = "success_hope" }
scene:apply_roll_outcome{}

-- Partial mapping: represent the extra Hope spend explicitly before rest.
-- Missing DSL: direct coupling of Hope cost to healing/rest command semantics.
scene:action_roll{
  actor = "Frodo",
  trait = "instinct",
  difficulty = 12,
  outcome = "success_hope",
  modifiers = {
    Modifiers.hope("hope_feature")
  }
}
scene:apply_roll_outcome{}
scene:rest{ type = "short", party_size = 1 }

scene:end_session()

return scene
