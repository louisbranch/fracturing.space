local scn = Scenario.new("interaction_control_gm_review")

-- GM review should make the review transition explicit and cover all three
-- review outcomes: request revisions, open the next player phase, and return
-- the scene to GM control.
scn:campaign{
  name = "Interaction Control GM Review",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "interaction"
}

scn:participant({name = "Guide", role = "GM"})
scn:participant({name = "Rhea"}):character({name = "Aria"})

scn:start_session("Archive")
scn:create_scene{
  name = "Flooded Archive",
  description = "Water rises around the ledger vault.",
  characters = {"Aria"}
}
scn:interaction_set_session_gm_authority({participant = "Guide"})
scn:interaction_open_scene_player_phase{
  as = "Guide",
  scene = "Flooded Archive",
  interaction = {
    title = "Rising Water",
    beats = {
      {type = "prompt", text = "Aria, what do you do before the ledger is swept away?"},
    },
  },
  characters = {"Aria"}
}
scn:interaction_submit_scene_player_action{
  as = "Rhea",
  scene = "Flooded Archive",
  summary = "Aria tests the current with a hooked pole before moving deeper.",
  characters = {"Aria"},
  yield = true
}
scn:interaction_expect{
  as = "Guide",
  phase_status = "GM_REVIEW",
  control_mode = "GM_REVIEW",
  allowed_transitions = {"OPEN_SESSION_OOC", "RESOLVE_SCENE_PLAYER_REVIEW"},
  recommended_transition = "RESOLVE_SCENE_PLAYER_REVIEW"
}

scn:interaction_resolve_scene_player_review{
  as = "Guide",
  scene = "Flooded Archive",
  interaction = {
    title = "Clarify The Route",
    beats = {
      {type = "guidance", text = "Commit to the route Aria takes through the floodwater."},
    },
  },
  revisions = {
    {participant = "Rhea", reason = "Commit to the route Aria takes through the floodwater.", characters = {"Aria"}}
  }
}
scn:interaction_expect{
  as = "Rhea",
  phase_status = "PLAYERS",
  control_mode = "PLAYERS",
  recommended_transition = "SUBMIT_SCENE_PLAYER_ACTION",
  slots = {
    {
      participant = "Rhea",
      summary = "Aria tests the current with a hooked pole before moving deeper.",
      characters = {"Aria"},
      review_status = "CHANGES_REQUESTED",
      review_reason = "Commit to the route Aria takes through the floodwater.",
      review_characters = {"Aria"}
    }
  }
}

scn:interaction_submit_scene_player_action{
  as = "Rhea",
  scene = "Flooded Archive",
  summary = "Aria wades left along the shelf line, hooks the ledger free, and backs toward the doorway.",
  characters = {"Aria"},
  yield = true
}
scn:interaction_expect{
  as = "Guide",
  phase_status = "GM_REVIEW",
  control_mode = "GM_REVIEW",
  recommended_transition = "RESOLVE_SCENE_PLAYER_REVIEW"
}
scn:interaction_resolve_scene_player_review{
  as = "Guide",
  scene = "Flooded Archive",
  interaction = {
    title = "Falling Shelf",
    beats = {
      {type = "consequence", text = "The shelf gives way and the doorway narrows behind Aria."},
      {type = "prompt", text = "Aria, do you dive through the gap or hold the shelf back?"},
    },
  },
  characters = {"Aria"}
}
scn:interaction_expect{
  as = "Rhea",
  phase_status = "PLAYERS",
  control_mode = "PLAYERS",
  recommended_transition = "SUBMIT_SCENE_PLAYER_ACTION",
  prompt = "Aria, do you dive through the gap or hold the shelf back?"
}

scn:interaction_submit_scene_player_action{
  as = "Rhea",
  scene = "Flooded Archive",
  summary = "Aria shoulders the shelf long enough to shove the ledger through the gap.",
  characters = {"Aria"},
  yield = true
}
scn:interaction_resolve_scene_player_review{
  as = "Guide",
  scene = "Flooded Archive",
  return_to_gm = true,
  interaction = {
    title = "Ledger Secured",
    beats = {
      {type = "resolution", text = "The ledger is secured and the beat returns to the GM."},
    },
  },
}
scn:interaction_expect{
  as = "Guide",
  phase_status = "GM",
  control_mode = "GM",
  recommended_transition = "OPEN_SCENE_PLAYER_PHASE",
  slots = {},
  gm_authority = "Guide"
}
scn:interaction_resolve_scene_player_review{
  as = "Guide",
  scene = "Flooded Archive",
  return_to_gm = true,
  interaction = {
    title = "Illegal Extra Review",
    beats = {
      {type = "guidance", text = "The GM tries to resolve review after the scene already returned to GM control."},
    },
  },
  expect_error = {code = "FAILED_PRECONDITION"}
}

scn:end_session()

return scn
