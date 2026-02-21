local scene = Scenario.new("head_guard_on_my_signal")

-- Capture the leader reaction that starts an archer countdown.
scene:campaign{
  name = "Gondor Captain On My Signal",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary"
}

scene:pc("Frodo")
scene:adversary("Gondor Captain")
scene:adversary("Gondor Archers")

-- The head guard signals archers to attack with advantage.
scene:start_session("On My Signal")

-- Example: reaction starts a countdown for coordinated archer fire.
-- Partial mapping: explicit countdown tick and advantaged archer volley are represented.
-- Missing DSL: automatic trigger of countdown ticks from qualifying PC attack outcomes.
scene:countdown_create{ name = "On My Signal", kind = "consequence", current = 0, max = 3, direction = "increase" }
scene:countdown_update{ name = "On My Signal", delta = 1, reason = "pc_attack_trigger" }
scene:adversary_attack{
  actor = "Gondor Archers",
  target = "Frodo",
  difficulty = 10,
  advantage = 1,
  damage_type = "physical"
}

scene:end_session()

return scene
