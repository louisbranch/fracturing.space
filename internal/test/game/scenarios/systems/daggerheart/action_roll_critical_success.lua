local scn = Scenario.new("action_roll_critical_success")
local dh = scn:system("DAGGERHEART")

-- Capture the critical success benefits from the example action roll.
scn:campaign{
  name = "Action Roll Critical Success",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "core"
}

scn:pc("Sam")
dh:adversary("Nazgul")

-- A critical success grants Hope, clears Stress, and offers an extra benefit.
scn:start_session("Critical Success")

dh:action_roll{ actor = "Sam", trait = "agility", difficulty = 14, outcome = "critical" }
dh:apply_roll_outcome{
  on_critical = {
    {kind = "apply_condition", target = "Nazgul", add = { "vulnerable" }},
  },
}

scn:end_session()

return scn
