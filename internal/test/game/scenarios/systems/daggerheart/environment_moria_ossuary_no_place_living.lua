local scn = Scenario.new("environment_moria_ossuary_no_place_living")
local dh = scn:system("DAGGERHEART")

-- Model the added Hope cost to clear HP in the ossuary.
scn:campaign{
  name = "Environment Ossuary No Place for the Living",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo", { hope = 2 })

-- Healing actions cost extra Hope here.
scn:start_session("No Place for the Living")

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
  hope_spends = {
    Modifiers.hope("hope_feature")
  }
}
dh:apply_roll_outcome{}
dh:rest{ type = "short", party_size = 1 }

scn:end_session()

return scn
