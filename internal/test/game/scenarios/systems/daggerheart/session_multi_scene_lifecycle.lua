-- Session lifecycle: multi-scene progression with interactions and session end.
-- Models a starter campaign flow: village → caves → lighthouse.
--
-- Note: scene_transition requires the same auth context as scene activation,
-- which conflicts with explicit GM authority in the current auth model.
-- This scenario uses implicit authority (campaign owner) for the full flow
-- until the game-service auth model supports GM participants with session
-- management capability. The session_conclusion_flow scenario demonstrates
-- the explicit GM authority pattern for interaction-only flows.
local scn = Scenario.new("session_multi_scene_lifecycle")

scn:campaign{
  name = "Multi-Scene Lifecycle",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "starter lifecycle"
}

scn:pc("Kael")

scn:start_session("Opening Night")

-- Act 1: Village.
scn:create_scene{
  name = "Brinewall Village",
  description = "A coastal fishing village beneath a darkened lighthouse.",
  characters = {"Kael"}
}

-- Act 2: Transition to caves atomically.
scn:scene_transition{
  scene = "Brinewall Village",
  name = "Sea Caves",
  description = "A narrow tunnel system connecting the hidden cove to the lighthouse basement."
}

-- Act 3: Transition to lighthouse.
scn:scene_transition{
  scene = "Sea Caves",
  name = "Lantern Room",
  description = "The circular lantern room at the top of the Ivory Beacon."
}

-- End scene and session cleanly.
scn:end_scene{name = "Lantern Room"}
scn:end_session()

return scn
