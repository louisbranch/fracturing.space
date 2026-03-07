local scn = Scenario.new("downtime_move_limit")
local dh = scn:system("DAGGERHEART")

-- Verify downtime move limits: 2 moves succeed, 3rd is rejected, rest resets.
scn:campaign{
  name = "Downtime Move Limit",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "downtime"
}

scn:pc("Frodo", { stress = 2 })

-- Two downtime moves should succeed within the limit.
scn:start_session("Downtime Limit")

-- First downtime move: clear all stress.
dh:downtime_move{
  target = "Frodo",
  move = "clear_all_stress",
  expect_stress_delta = -2,
}

-- Second downtime move: prepare (gains hope).
dh:downtime_move{
  target = "Frodo",
  move = "prepare",
  expect_hope_delta = 1,
}

-- UNRESOLVED: 3rd downtime move should be rejected (DOWNTIME_MOVE_LIMIT_HIT),
-- but the scenario DSL lacks an expect_rejection primitive to assert this.

-- Short rest resets the counter.
dh:rest{ type = "short", party_size = 1, seed = 42 }

-- After rest, another downtime move should succeed.
dh:downtime_move{
  target = "Frodo",
  move = "prepare",
  expect_hope_delta = 1,
}

scn:end_session()

return scn
