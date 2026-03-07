-- Scene transition: atomic move of all characters to a new scene.
local scn = Scenario.new("scene_transition")

scn:campaign{
  name = "Scene Transition Campaign",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "basics"
}

scn:pc("Frodo")
scn:pc("Sam")

scn:start_session("Session")

scn:create_scene{name = "Room A", characters = {"Frodo", "Sam"}}

-- Transition atomically moves all characters from Room A to Room B.
scn:scene_transition{scene = "Room A", name = "Room B", description = "Through the door"}

scn:end_scene{name = "Room B"}
scn:end_session()

return scn
