local scn = Scenario.new("environment_gondor_court_gravity_of_empire")
local dh = scn:system("DAGGERHEART")

-- Model the golden opportunity and stress/temptation consequences.
scn:campaign{
  name = "Environment Gondor Court Gravity of Empire",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- The empire tempts a PC with a major offer.
scn:start_session("Gravity of Empire")
dh:gm_fear(1)

-- Narrative spotlight temptation — the reaction roll and conditions model the
-- mechanical pressure; no gold mechanic is involved.
dh:gm_spend_fear(1):spotlight("Gondor Court", { description = "golden_opportunity_offer" })
dh:reaction_roll{ actor = "Frodo", trait = "presence", difficulty = 20, outcome = "failure_fear" }
dh:apply_reaction_outcome{
  on_failure = {
    {kind = "apply_condition", target = "Frodo", add = {"VULNERABLE"}, source = "imperial_temptation_pressure"},
    {kind = "set_spotlight", type = "gm"},
  },
  on_success = {
    {kind = "apply_condition", target = "Frodo", add = {"VULNERABLE"}, source = "imperial_temptation_tax"},
  },
}
scn:set_spotlight{ target = "Frodo" }

scn:end_session()

return scn
