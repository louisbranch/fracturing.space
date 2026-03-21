local scn = Scenario.new("interaction_control_interrupt")

-- The GM may explicitly interrupt an open player phase and return the active
-- scene to GM control without switching scenes.
scn:campaign{
  name = "Interaction Control Interrupt",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "interaction"
}

scn:participant({name = "Guide", role = "GM"})
scn:participant({name = "Rhea"}):character({name = "Aria"})

scn:start_session("Rooftop")
scn:create_scene{
  name = "Slate Roof",
  description = "Aria watches the alley from a wet rooftop.",
  characters = {"Aria"}
}
scn:interaction_set_session_gm_authority({participant = "Guide"})
scn:interaction_open_scene_player_phase{
  as = "Guide",
  scene = "Slate Roof",
  interaction = {
    title = "Quiet Alley",
    beats = {
      {type = "prompt", text = "The alley below is quiet for the moment. What do you do?"},
    },
  },
  characters = {"Aria"}
}
scn:interaction_expect{
  as = "Guide",
  phase_status = "PLAYERS",
  control_mode = "PLAYERS",
  allowed_transitions = {
    "INTERRUPT_SCENE_PLAYER_PHASE",
    "OPEN_SESSION_OOC"
  },
  recommended_transition = "UNSPECIFIED"
}
scn:interaction_interrupt_scene_player_phase{
  as = "Guide",
  scene = "Slate Roof",
  reason = "gm_reframes_scene"
}
scn:interaction_expect{
  as = "Guide",
  phase_status = "GM",
  control_mode = "GM",
  recommended_transition = "OPEN_SCENE_PLAYER_PHASE",
  slots = {}
}
scn:interaction_submit_scene_player_action{
  as = "Rhea",
  scene = "Slate Roof",
  summary = "Aria tries to keep acting after the GM interrupts the phase.",
  characters = {"Aria"},
  expect_error = {code = "FAILED_PRECONDITION", contains = "scene player phase is not open"}
}

scn:end_session()

return scn
