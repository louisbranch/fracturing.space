local scn = Scenario.new("interaction_control_ooc")

-- OOC open should expose transcript/ready actions, keep scene writes blocked,
-- and make the pause visible in control state.
scn:campaign{
  name = "Interaction Control OOC",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "interaction"
}

scn:participant({name = "Guide", role = "GM"})
scn:participant({name = "Rhea"}):character({name = "Aria"})

scn:start_session("Vault")
scn:create_scene{
  name = "Sealed Vault",
  description = "A ward crackles along the seam of the vault door.",
  characters = {"Aria"}
}
scn:interaction_set_session_gm_authority({participant = "Guide"})
scn:interaction_open_scene_player_phase{
  as = "Guide",
  scene = "Sealed Vault",
  interaction = {
    title = "Ward At The Seam",
    beats = {
      {type = "prompt", text = "Aria, how do you test the ward at the seam?"},
    },
  },
  characters = {"Aria"}
}
scn:interaction_open_session_ooc({as = "Rhea", reason = "Clarify how the ward reacts to tools."})
scn:interaction_expect{
  as = "Rhea",
  control_mode = "OOC",
  ooc_open = true,
  ooc_requested_by = "Rhea",
  allowed_transitions = {"POST_SESSION_OOC", "MARK_OOC_READY_TO_RESUME"},
  recommended_transition = "MARK_OOC_READY_TO_RESUME"
}
scn:interaction_post_session_ooc({as = "Rhea", body = "Does the ward react to contact with the seam or only to skin?"})
scn:interaction_mark_ooc_ready_to_resume({as = "Rhea"})
scn:interaction_expect{
  as = "Rhea",
  control_mode = "OOC",
  allowed_transitions = {"POST_SESSION_OOC", "CLEAR_OOC_READY_TO_RESUME"},
  recommended_transition = "POST_SESSION_OOC",
  ooc_ready = {"Rhea"},
  ooc_posts = {
    {participant = "Rhea", body = "Does the ward react to contact with the seam or only to skin?"}
  }
}
scn:interaction_submit_scene_player_action{
  as = "Rhea",
  scene = "Sealed Vault",
  summary = "Aria tries to keep playing while OOC is open.",
  characters = {"Aria"},
  expect_error = {code = "FAILED_PRECONDITION", contains = "out-of-character discussion"}
}

scn:end_session()

return scn
