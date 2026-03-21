local scn = Scenario.new("interaction_control_multi_actor")

-- A shared acting set should keep both acting participants in PLAYERS until
-- the final yield moves the phase into GM review.
scn:campaign{
  name = "Interaction Control Multi Actor",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "interaction"
}

scn:participant({name = "Guide", role = "GM"})
scn:participant({name = "Rhea"}):character({name = "Aria"})
scn:participant({name = "Bryn"}):character({name = "Corin"})

scn:start_session("Crossing")
scn:create_scene{
  name = "The Bridge",
  description = "Rain lashes the ropes and the planks sway over the gorge.",
  characters = {"Aria", "Corin"}
}
scn:interaction_set_session_gm_authority({participant = "Guide"})
scn:interaction_open_scene_player_phase{
  as = "Guide",
  scene = "The Bridge",
  interaction = {
    title = "Bridge Prompt",
    beats = {
      {type = "prompt", text = "The bridge jerks under your boots. Aria and Corin, what do you do?"},
    },
  },
  characters = {"Aria", "Corin"}
}
scn:interaction_expect{
  as = "Guide",
  phase_status = "PLAYERS",
  control_mode = "PLAYERS",
  acting_characters = {"Aria", "Corin"},
  acting_participants = {"Rhea", "Bryn"}
}

scn:interaction_submit_scene_player_action{
  as = "Rhea",
  scene = "The Bridge",
  summary = "Aria drops low and grabs the near rope to steady the span.",
  characters = {"Aria"}
}
scn:interaction_yield_scene_player_phase({as = "Rhea", scene = "The Bridge"})
scn:interaction_expect{
  as = "Bryn",
  phase_status = "PLAYERS",
  control_mode = "PLAYERS",
  acting_characters = {"Aria", "Corin"},
  acting_participants = {"Rhea", "Bryn"},
  recommended_transition = "SUBMIT_SCENE_PLAYER_ACTION",
  slots = {
    {
      participant = "Bryn"
    },
    {
      participant = "Rhea",
      summary = "Aria drops low and grabs the near rope to steady the span.",
      characters = {"Aria"},
      yielded = true
    }
  }
}

scn:interaction_submit_scene_player_action{
  as = "Bryn",
  scene = "The Bridge",
  summary = "Corin shields the lantern and inches toward the midpoint.",
  characters = {"Corin"}
}
scn:interaction_yield_scene_player_phase({as = "Bryn", scene = "The Bridge"})
scn:interaction_expect{
  as = "Guide",
  phase_status = "GM_REVIEW",
  control_mode = "GM_REVIEW",
  recommended_transition = "RESOLVE_SCENE_PLAYER_REVIEW",
  slots = {
    {
      participant = "Bryn",
      summary = "Corin shields the lantern and inches toward the midpoint.",
      characters = {"Corin"},
      yielded = true,
      review_status = "UNDER_REVIEW"
    },
    {
      participant = "Rhea",
      summary = "Aria drops low and grabs the near rope to steady the span.",
      characters = {"Aria"},
      yielded = true,
      review_status = "UNDER_REVIEW"
    }
  }
}

scn:end_session()

return scn
