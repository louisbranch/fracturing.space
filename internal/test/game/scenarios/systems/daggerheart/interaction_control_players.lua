local scn = Scenario.new("interaction_control_players")

-- Player control should expose submit/yield/withdraw-yield to acting players,
-- keep non-acting players from posting, and keep the GM from slipping in a
-- GM-only write.
scn:campaign{
  name = "Interaction Control Players",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "interaction"
}

scn:participant({name = "Guide", role = "GM"})
scn:participant({name = "Rhea"}):character({name = "Aria"})
scn:participant({name = "Bryn"}):character({name = "Corin"})
scn:participant({name = "Dax"}):character({name = "Mira"})

scn:start_session("Watchtower")
scn:create_scene{
  name = "Watch Platform",
  description = "Aria watches the road while Corin waits below and Mira keeps to the stairwell.",
  characters = {"Aria", "Corin", "Mira"}
}
scn:interaction_set_session_gm_authority({participant = "Guide"})
scn:interaction_open_scene_player_phase{
  as = "Guide",
  scene = "Watch Platform",
  interaction = {
    title = "Changing Watch",
    beats = {
      {type = "fiction", text = "A horn sounds from the road below the tower."},
      {type = "prompt", text = "Aria and Corin, what do you do from the watch platform?"},
    },
  },
  characters = {"Aria", "Corin"}
}
scn:interaction_expect{
  as = "Rhea",
  active_scene = "Watch Platform",
  phase_status = "PLAYERS",
  control_mode = "PLAYERS",
  acting_participants = {"Rhea", "Bryn"},
  allowed_transitions = {
    "OPEN_SESSION_OOC",
    "SUBMIT_SCENE_PLAYER_ACTION",
    "YIELD_SCENE_PLAYER_PHASE"
  },
  recommended_transition = "SUBMIT_SCENE_PLAYER_ACTION"
}
scn:interaction_expect{
  as = "Dax",
  control_mode = "PLAYERS",
  allowed_transitions = {"OPEN_SESSION_OOC"},
  recommended_transition = "UNSPECIFIED"
}
scn:interaction_submit_scene_player_action{
  as = "Dax",
  scene = "Watch Platform",
  summary = "Mira answers even though she is not acting in this phase.",
  characters = {"Mira"},
  expect_error = {code = "PERMISSION_DENIED", contains = "participant is not acting"}
}
scn:interaction_record_scene_gm_interaction{
  as = "Guide",
  scene = "Watch Platform",
  interaction = {
    title = "Illegal GM Insert",
    beats = {
      {type = "fiction", text = "The GM tries to narrate over the open player phase."},
    },
  },
  expect_error = {code = "FAILED_PRECONDITION", contains = "scene player phase is open"}
}

scn:interaction_submit_scene_player_action{
  as = "Rhea",
  scene = "Watch Platform",
  summary = "Aria drops low behind the parapet and studies the riders through the rain.",
  characters = {"Aria"}
}
scn:interaction_expect{
  as = "Rhea",
  phase_status = "PLAYERS",
  control_mode = "PLAYERS",
  acting_participants = {"Rhea", "Bryn"},
  allowed_transitions = {
    "OPEN_SESSION_OOC",
    "SUBMIT_SCENE_PLAYER_ACTION",
    "YIELD_SCENE_PLAYER_PHASE"
  },
  recommended_transition = "YIELD_SCENE_PLAYER_PHASE",
  slots = {
    {participant = "Rhea", summary = "Aria drops low behind the parapet and studies the riders through the rain.", characters = {"Aria"}},
    {participant = "Bryn"}
  }
}
scn:interaction_yield_scene_player_phase({as = "Rhea", scene = "Watch Platform"})
scn:interaction_expect{
  as = "Rhea",
  phase_status = "PLAYERS",
  control_mode = "PLAYERS",
  acting_participants = {"Rhea", "Bryn"},
  allowed_transitions = {
    "OPEN_SESSION_OOC",
    "SUBMIT_SCENE_PLAYER_ACTION",
    "WITHDRAW_SCENE_PLAYER_YIELD"
  },
  recommended_transition = "WITHDRAW_SCENE_PLAYER_YIELD",
  slots = {
    {
      participant = "Rhea",
      summary = "Aria drops low behind the parapet and studies the riders through the rain.",
      characters = {"Aria"},
      yielded = true
    },
    {participant = "Bryn"}
  }
}
scn:interaction_withdraw_scene_player_yield({as = "Rhea", scene = "Watch Platform"})
scn:interaction_expect{
  as = "Rhea",
  phase_status = "PLAYERS",
  control_mode = "PLAYERS",
  acting_participants = {"Rhea", "Bryn"},
  allowed_transitions = {
    "OPEN_SESSION_OOC",
    "SUBMIT_SCENE_PLAYER_ACTION",
    "YIELD_SCENE_PLAYER_PHASE"
  },
  recommended_transition = "YIELD_SCENE_PLAYER_PHASE",
  slots = {
    {participant = "Rhea", summary = "Aria drops low behind the parapet and studies the riders through the rain.", characters = {"Aria"}},
    {participant = "Bryn"}
  }
}
scn:interaction_submit_scene_player_action{
  as = "Bryn",
  scene = "Watch Platform",
  summary = "Corin leans over the rail and tracks how many riders are on the road below.",
  characters = {"Corin"}
}
scn:interaction_expect{
  as = "Bryn",
  phase_status = "PLAYERS",
  control_mode = "PLAYERS",
  allowed_transitions = {
    "OPEN_SESSION_OOC",
    "SUBMIT_SCENE_PLAYER_ACTION",
    "YIELD_SCENE_PLAYER_PHASE"
  },
  recommended_transition = "YIELD_SCENE_PLAYER_PHASE",
  slots = {
    {participant = "Bryn", summary = "Corin leans over the rail and tracks how many riders are on the road below.", characters = {"Corin"}},
    {participant = "Rhea", summary = "Aria drops low behind the parapet and studies the riders through the rain.", characters = {"Aria"}}
  }
}
scn:interaction_yield_scene_player_phase({as = "Rhea", scene = "Watch Platform"})
scn:interaction_expect{
  as = "Bryn",
  phase_status = "PLAYERS",
  control_mode = "PLAYERS",
  allowed_transitions = {
    "OPEN_SESSION_OOC",
    "SUBMIT_SCENE_PLAYER_ACTION",
    "YIELD_SCENE_PLAYER_PHASE"
  },
  recommended_transition = "YIELD_SCENE_PLAYER_PHASE",
  slots = {
    {participant = "Bryn", summary = "Corin leans over the rail and tracks how many riders are on the road below.", characters = {"Corin"}},
    {
      participant = "Rhea",
      summary = "Aria drops low behind the parapet and studies the riders through the rain.",
      characters = {"Aria"},
      yielded = true
    }
  }
}
scn:interaction_yield_scene_player_phase({as = "Bryn", scene = "Watch Platform"})
scn:interaction_expect{
  as = "Guide",
  phase_status = "GM_REVIEW",
  control_mode = "GM_REVIEW",
  recommended_transition = "RESOLVE_SCENE_PLAYER_REVIEW",
  slots = {
    {
      participant = "Bryn",
      summary = "Corin leans over the rail and tracks how many riders are on the road below.",
      characters = {"Corin"},
      yielded = true,
      review_status = "UNDER_REVIEW"
    },
    {
      participant = "Rhea",
      summary = "Aria drops low behind the parapet and studies the riders through the rain.",
      characters = {"Aria"},
      yielded = true,
      review_status = "UNDER_REVIEW"
    }
  }
}

scn:end_session()

return scn
