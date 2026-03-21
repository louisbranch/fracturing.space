local scn = Scenario.new("interaction_control_immediate_yield")

-- A single acting participant may yield without posting, which should move the
-- beat directly into GM review with an empty summary slot.
scn:campaign{
  name = "Interaction Control Immediate Yield",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "interaction"
}

scn:participant({name = "Guide", role = "GM"})
scn:participant({name = "Rhea"}):character({name = "Aria"})

scn:start_session("Rooftops")
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
      {type = "prompt", text = "The alley below is quiet for the moment. Do you act now or keep watching?"},
    },
  },
  characters = {"Aria"}
}
scn:interaction_expect{
  as = "Rhea",
  phase_status = "PLAYERS",
  control_mode = "PLAYERS",
  acting_characters = {"Aria"},
  acting_participants = {"Rhea"},
  recommended_transition = "SUBMIT_SCENE_PLAYER_ACTION"
}
scn:interaction_yield_scene_player_phase({as = "Rhea", scene = "Slate Roof"})
scn:interaction_expect{
  as = "Guide",
  phase_status = "GM_REVIEW",
  control_mode = "GM_REVIEW",
  recommended_transition = "RESOLVE_SCENE_PLAYER_REVIEW",
  slots = {
    {participant = "Rhea", yielded = true, review_status = "UNDER_REVIEW"}
  }
}
scn:interaction_resolve_scene_player_review{
  as = "Guide",
  scene = "Slate Roof",
  return_to_gm = true,
  interaction = {
    title = "Watching The Alley",
    beats = {
      {type = "resolution", text = "Aria yields the moment and the scene returns to GM control."},
    },
  },
}
scn:interaction_expect{
  as = "Guide",
  phase_status = "GM",
  control_mode = "GM",
  recommended_transition = "OPEN_SCENE_PLAYER_PHASE",
  slots = {}
}

scn:end_session()

return scn
