local scn = Scenario.new("gm_move_examples")
local dh = scn:system("DAGGERHEART")

-- Showcase typed direct GM move spends tied to roll outcomes.
scn:campaign{
  name = "GM Move Examples",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "gm_move"
}

scn:pc("Gandalf")
dh:adversary("Nazgul")

-- The GM responds to fear and failure with narrative moves.
scn:start_session("GM Moves")
dh:gm_fear(2)

-- Example: an additional move can still be recorded explicitly without opening
-- a new interruption gate.
dh:gm_spend_fear(1):move("custom", { description = "Ash and embers swirl through the chamber." })

-- Example: roll with Fear triggers an interrupt-style move that opens the GM
-- consequence path.
dh:action_roll{ actor = "Gandalf", trait = "presence", difficulty = 12, outcome = "success_hope" }
dh:apply_roll_outcome{}
dh:gm_spend_fear(1):move("reveal_danger", { description = "The Nazgul closes the distance in a blur." })

-- Close the session after the GM move sequence.
scn:end_session()

return scn
