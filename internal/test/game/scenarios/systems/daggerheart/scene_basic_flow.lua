-- Scene lifecycle: create a scene with characters, then end it.
local scn = Scenario.new("scene_basic_flow")

scn:campaign{
  name = "Scene Basic Flow",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "basics"
}

scn:pc("Frodo")
scn:pc("Sam")

scn:start_session("Session")

-- Create a scene scoped to this session.
scn:create_scene{name = "The Shire", description = "A peaceful beginning", characters = {"Frodo", "Sam"}}

-- End the scene cleanly.
scn:end_scene{name = "The Shire"}

scn:end_session()

return scn
