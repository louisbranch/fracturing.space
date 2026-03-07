-- Scene-scoped gate and spotlight lifecycle.
local scn = Scenario.new("scene_gate_spotlight")

scn:campaign{
  name = "Scene Gate Spotlight Campaign",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "basics"
}

scn:pc("Frodo")

scn:start_session("Session")

scn:create_scene{name = "Battle", characters = {"Frodo"}}

-- Set spotlight on a character within the scene.
scn:scene_set_spotlight{scene = "Battle", type = "character", target = "Frodo"}

-- Open a decision gate, resolve it, then clear spotlight.
scn:scene_gate_open{scene = "Battle", gate_type = "decision", gate_id = "gate-1"}
scn:scene_gate_resolve{scene = "Battle", gate_id = "gate-1", decision = "allow"}
scn:scene_clear_spotlight{scene = "Battle"}

scn:end_scene{name = "Battle"}
scn:end_session()

return scn
