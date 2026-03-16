local scn = Scenario.new("downtime_move_limit")
local dh = scn:system("DAGGERHEART")

-- Verify the atomic rest workflow still allows up to two downtime moves per participant.
scn:campaign{
  name = "Downtime Move Limit",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "downtime"
}

scn:pc("Frodo", { stress = 2 })

scn:start_session("Downtime Limit")

dh:rest{
  type = "long",
  participants = {
    {
      character = "Frodo",
      downtime_moves = {
        { move = "clear_all_stress" },
        { move = "prepare", expect_stress_delta = -2, expect_hope_delta = 1 },
      },
    },
  },
}

scn:end_session()

return scn
