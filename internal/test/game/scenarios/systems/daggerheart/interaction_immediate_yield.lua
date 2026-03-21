local scn = Scenario.new("interaction_immediate_yield")

-- A single acting participant yields without posting, so control returns
-- immediately to the GM.
scn:campaign{
  name = "Interaction Immediate Yield",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "interaction"
}

scn:participant({name = "Guide", role = "GM"})
scn:participant({name = "Rhea"}):character({name = "Aria"})

-- Open a one-character rooftop beat.
scn:start_session("Rooftops")
scn:create_scene{
  name = "Slate Roof",
  description = "Aria watches the alley from a wet rooftop.",
  characters = {"Aria"}
}
scn:interaction_set_gm_authority({participant = "Guide"})
scn:interaction_set_active_scene({scene = "Slate Roof"})
scn:interaction_start_player_phase{
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
  phase_status = "PLAYERS",
  acting_characters = {"Aria"},
  acting_participants = {"Rhea"}
}
scn:interaction_yield({as = "Rhea", scene = "Slate Roof"})
scn:interaction_expect{
  phase_status = "GM_REVIEW",
  slots = {
    {participant = "Rhea", yielded = true, review_status = "UNDER_REVIEW"}
  },
  gm_authority = "Guide"
}
scn:interaction_resolve_review({
  as = "Guide",
  return_to_gm = true,
  interaction = {
    title = "Watching The Alley",
    beats = {
      {type = "resolution", text = "Aria yields the moment and the scene returns to GM control."},
    },
  },
})
scn:interaction_expect{
  phase_status = "GM",
  slots = {},
  gm_authority = "Guide"
}

scn:end_session()

return scn
