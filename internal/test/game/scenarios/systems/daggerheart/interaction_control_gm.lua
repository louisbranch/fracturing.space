local scn = Scenario.new("interaction_control_gm")

-- GM control should expose scene activation, GM narration, and player-phase
-- opening as the legal next moves after a default-active scene create.
scn:campaign{
  name = "Interaction Control GM",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "interaction"
}

scn:participant({name = "Guide", role = "GM"})
scn:participant({name = "Rhea"}):character({name = "Aria"})

scn:start_session("Harbor Dawn")
scn:create_scene{
  name = "Ledger Wharf",
  description = "Fog drifts over the wharf while the debt collector waits.",
  characters = {"Aria"}
}
scn:interaction_set_session_gm_authority({participant = "Guide"})
scn:interaction_expect{
  as = "Guide",
  active_scene = "Ledger Wharf",
  phase_status = "GM",
  control_mode = "GM",
  allowed_transitions = {
    "ACTIVATE_SCENE",
    "OPEN_SCENE_PLAYER_PHASE",
    "OPEN_SESSION_OOC",
    "RECORD_SCENE_GM_INTERACTION"
  },
  recommended_transition = "OPEN_SCENE_PLAYER_PHASE",
  gm_authority = "Guide"
}

scn:interaction_record_scene_gm_interaction{
  as = "Guide",
  scene = "Ledger Wharf",
  interaction = {
    title = "Debt At Dawn",
    beats = {
      {type = "fiction", text = "A courier lifts a black-sealed notice and calls Aria's debt due at dawn."},
      {type = "guidance", text = "The wharf is tense but no player action is open yet."},
    },
  },
}
scn:interaction_expect{
  as = "Guide",
  active_scene = "Ledger Wharf",
  phase_status = "GM",
  control_mode = "GM",
  recommended_transition = "OPEN_SCENE_PLAYER_PHASE"
}

scn:interaction_submit_scene_player_action{
  as = "Guide",
  scene = "Ledger Wharf",
  summary = "The GM tries to write a player action while no player phase is open.",
  characters = {"Aria"},
  expect_error = {code = "FAILED_PRECONDITION", contains = "scene player phase is not open"}
}

scn:end_session()

return scn
