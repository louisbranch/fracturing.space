local scn = Scenario.new("interaction_invalid_flow_contracts")

-- Invalid interaction flows should fail with stable gRPC status codes instead
-- of silently mutating unrelated scene state.
scn:campaign{
  name = "Interaction Invalid Flow Contracts",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "interaction"
}

scn:participant({name = "Guide", role = "GM"})
scn:participant({name = "Rhea"}):character({name = "Aria"})
scn:participant({name = "Bryn"}):character({name = "Corin"})

-- Create two scenes so the invalid active-scene transition can be asserted directly.
scn:start_session("Contracts")
scn:create_scene{
  name = "Signal Tower",
  description = "Aria watches the road from the tower platform.",
  characters = {"Aria"}
}
scn:create_scene{
  name = "Courtyard",
  description = "Corin waits below for the next sign from the tower.",
  characters = {"Corin"}
}
scn:interaction_set_gm_authority({participant = "Guide"})
scn:interaction_set_active_scene({scene = "Signal Tower"})

-- 1. A player phase cannot start on a scene that is not active.
scn:interaction_start_player_phase{
  scene = "Courtyard",
  interaction = {
    title = "Courtyard Prompt",
    beats = {
      {type = "prompt", text = "Corin, what do you do below the tower?"},
    },
  },
  characters = {"Corin"},
  expect_error = {code = "FAILED_PRECONDITION", contains = "scene is not the active scene"}
}

-- 2. A participant outside the acting set cannot post into the current player phase.
scn:interaction_start_player_phase{
  scene = "Signal Tower",
  interaction = {
    title = "Signal Tower Prompt",
    beats = {
      {type = "prompt", text = "Aria, what do you do from the tower?"},
    },
  },
  characters = {"Aria"}
}
scn:interaction_post{
  as = "Bryn",
  scene = "Signal Tower",
  summary = "Corin answers from below even though it is not his beat.",
  characters = {"Corin"},
  expect_error = {code = "PERMISSION_DENIED", contains = "participant is not acting in the current scene phase"}
}

-- 3. Yielding without an open player phase must fail.
scn:interaction_end_player_phase({scene = "Signal Tower"})
scn:interaction_yield{
  as = "Rhea",
  scene = "Signal Tower",
  expect_error = {code = "FAILED_PRECONDITION", contains = "scene player phase is not open"}
}

-- 4. Resuming when the session is not paused for OOC must fail.
scn:interaction_resume_ooc({
  expect_error = {code = "FAILED_PRECONDITION", contains = "session is not paused for out-of-character discussion"}
})

scn:end_session()

return scn
