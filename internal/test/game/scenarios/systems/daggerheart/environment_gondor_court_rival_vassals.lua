local scn = Scenario.new("environment_gondor_court_rival_vassals")
local dh = scn:system("DAGGERHEART")

-- Capture the rival vassals social pressure in the court.
scn:campaign{
  name = "Environment Gondor Court Gondor Vassals",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
scn:npc("Gondor Vassals")

-- Courtiers compete for favor and feed intrigue.
scn:start_session("Gondor Vassals")
dh:gm_fear(1)

-- Partial mapping: rivalry pressure and social fallout are explicit by branch.
-- Missing DSL: first-class favor/debt economy between court factions and PCs.
dh:gm_spend_fear(1):spotlight("Gondor Vassals", { description = "court_favor_competition" })
dh:action_roll{ actor = "Frodo", trait = "presence", difficulty = 14, disadvantage = 1, outcome = "failure_fear" }
dh:apply_roll_outcome{
  on_failure_fear = {
    {kind = "apply_condition", target = "Frodo", add = {"VULNERABLE"}, source = "rival_vassal_pressure"},
    {kind = "set_spotlight", type = "gm"},
  },
  on_success = {
    {kind = "set_spotlight", target = "Frodo"},
  },
}
scn:set_spotlight{ target = "Frodo" }

scn:end_session()

return scn
