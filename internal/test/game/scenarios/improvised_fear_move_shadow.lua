local scene = Scenario.new("improvised_fear_move_shadow")

-- Showcase an improvised fear move that shifts the scene.
scene:campaign{
  name = "Improvised Fear Move Shadow",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "gm_fear"
}

scene:pc("Frodo")

-- The GM spends fear after a success-with-fear to escalate the chase.
scene:start_session("Fear Move")
scene:gm_fear(2)

scene:countdown_create{ name = "Looming Shadow", kind = "consequence", current = 0, max = 4, direction = "increase" }
scene:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 12, outcome = "success_fear" }

-- Example: the GM spends fear to introduce a looming shadow.
-- Partial mapping: fear spend plus deterministic pressure and condition effects.
-- Missing DSL: richer scene-wide separation effects from improvised fear moves.
scene:apply_roll_outcome{
  on_success_fear = {
    {kind = "gm_spend_fear", amount = 1, target = "Frodo", description = "looming_shadow_overtakes_path"},
    {kind = "countdown_update", name = "Looming Shadow", delta = 1, reason = "shadow_closes_in"},
    {kind = "apply_condition", target = "Frodo", add = {"VULNERABLE"}, source = "looming_shadow"},
    {kind = "set_spotlight", type = "gm"},
  },
}
scene:set_spotlight{ target = "Frodo" }

-- Close the session after the fear move.
scene:end_session()

return scene
