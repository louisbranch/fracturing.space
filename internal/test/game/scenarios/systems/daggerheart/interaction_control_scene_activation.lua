local scn = Scenario.new("interaction_control_scene_activation")

-- Scene creation should activate by default, activate=false should leave the
-- current active scene unchanged, and switching scenes should interrupt the
-- open phase on the previous scene.
scn:campaign{
  name = "Interaction Control Scene Activation",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "interaction"
}

scn:participant({name = "Guide", role = "GM"})
scn:participant({name = "Rhea"}):character({name = "Aria"})
scn:participant({name = "Bryn"}):character({name = "Corin"})

scn:start_session("Split Party")
scn:create_scene{
  name = "North Gate",
  description = "Aria watches the guard rotation above the gate.",
  characters = {"Aria"}
}
scn:create_scene{
  name = "South Tunnel",
  description = "Corin waits in the drainage tunnel beneath the keep.",
  characters = {"Corin"},
  activate = false
}
scn:interaction_set_session_gm_authority({participant = "Guide"})
scn:interaction_expect{
  as = "Guide",
  active_scene = "North Gate",
  phase_status = "GM",
  control_mode = "GM",
  recommended_transition = "OPEN_SCENE_PLAYER_PHASE"
}

scn:interaction_open_scene_player_phase{
  as = "Guide",
  scene = "North Gate",
  interaction = {
    title = "Changing Watch",
    beats = {
      {type = "prompt", text = "Aria, what do you do before the guard change completes?"},
    },
  },
  characters = {"Aria"}
}
scn:interaction_submit_scene_player_action{
  as = "Rhea",
  scene = "North Gate",
  summary = "Aria counts the blind angles and marks the quickest route down the wall.",
  characters = {"Aria"}
}
scn:interaction_activate_scene({as = "Guide", scene = "South Tunnel"})
scn:interaction_expect{
  as = "Guide",
  active_scene = "South Tunnel",
  phase_status = "GM",
  control_mode = "GM",
  recommended_transition = "OPEN_SCENE_PLAYER_PHASE"
}

scn:interaction_submit_scene_player_action{
  as = "Rhea",
  scene = "North Gate",
  summary = "Aria tries to keep acting on the now-inactive scene.",
  characters = {"Aria"},
  expect_error = {code = "FAILED_PRECONDITION", contains = "scene is not the active scene"}
}

scn:end_session()

return scn
