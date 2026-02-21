local scene = Scenario.new("environment_gondor_court_gravity_of_empire")

-- Model the golden opportunity and stress/temptation consequences.
scene:campaign{
  name = "Environment Gondor Court Gravity of Empire",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- The empire tempts a PC with a major offer.
scene:start_session("Gravity of Empire")
scene:gm_fear(1)

-- Missing DSL: apply Presence reaction and stress or acceptance on failure.
scene:gm_spend_fear(1):spotlight("Gondor Court", { description = "golden_opportunity_offer" })
scene:reaction_roll{ actor = "Frodo", trait = "presence", difficulty = 20, outcome = "failure_fear" }
scene:apply_reaction_outcome{
  on_failure = {
    {kind = "apply_condition", target = "Frodo", add = {"VULNERABLE"}, source = "imperial_temptation_pressure"},
    {kind = "set_spotlight", type = "gm"},
  },
  on_success = {
    {kind = "apply_condition", target = "Frodo", add = {"VULNERABLE"}, source = "imperial_temptation_tax"},
  },
}
scene:set_spotlight{ target = "Frodo" }

scene:end_session()

return scene
