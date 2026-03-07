local scene = Scenario.new("environment_moria_ossuary_no_place_living")
local dh = scene:system("DAGGERHEART")

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
dh:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 10, outcome = "success_hope" }
dh:apply_roll_outcome{}

-- Partial mapping: represent the extra Hope spend explicitly before rest.
-- Missing DSL: direct coupling of Hope cost to healing/rest command semantics.
dh:action_roll{
  actor = "Frodo",
  trait = "instinct",
  difficulty = 12,
  outcome = "success_hope",
  modifiers = {
    Modifiers.hope("hope_feature")
  }
}
dh:apply_roll_outcome{}
dh:rest{ type = "short", party_size = 1 }

scene:end_session()

return scene
