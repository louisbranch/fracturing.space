local scene = Scenario.new("environment_gondor_court_rival_vassals")

-- Capture the rival vassals social pressure in the court.
scene:campaign{
  name = "Environment Gondor Court Gondor Vassals",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
scene:npc("Gondor Vassals")

-- Courtiers compete for favor and feed intrigue.
scene:start_session("Gondor Vassals")
scene:gm_fear(1)

-- Partial mapping: rivalry pressure and social fallout are explicit by branch.
-- Missing DSL: first-class favor/debt economy between court factions and PCs.
scene:gm_spend_fear(1):spotlight("Gondor Vassals", { description = "court_favor_competition" })
scene:action_roll{ actor = "Frodo", trait = "presence", difficulty = 14, disadvantage = 1, outcome = "failure_fear" }
scene:apply_roll_outcome{
  on_failure_fear = {
    {kind = "apply_condition", target = "Frodo", add = {"VULNERABLE"}, source = "rival_vassal_pressure"},
    {kind = "set_spotlight", type = "gm"},
  },
  on_success = {
    {kind = "set_spotlight", target = "Frodo"},
  },
}
scene:set_spotlight{ target = "Frodo" }

scene:end_session()

return scene
