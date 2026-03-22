local scn = Scenario.new("head_guard_on_my_signal")
local dh = scn:system("DAGGERHEART")

-- Capture the leader reaction that starts an archer countdown.
scn:campaign{
  name = "Gondor Captain On My Signal",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary"
}

scn:pc("Frodo")
dh:adversary("Gondor Captain")
dh:adversary("Gondor Archers")

-- The head guard signals archers to attack with advantage.
scn:start_session("On My Signal")

-- Example: reaction starts a countdown for coordinated archer fire.
-- Partial mapping: explicit countdown tick and advantaged archer volley are represented.
-- Missing DSL: automatic trigger of countdown ticks from qualifying PC attack outcomes.
dh:scene_countdown_create{ name = "On My Signal", kind = "consequence", current = 0, max = 3, direction = "increase" }
dh:scene_countdown_update{ name = "On My Signal", delta = 1, reason = "pc_attack_trigger" }
dh:adversary_attack{
  actor = "Gondor Archers",
  target = "Frodo",
  difficulty = 10,
  advantage = 1,
  damage_type = "physical"
}

scn:end_session()

return scn
