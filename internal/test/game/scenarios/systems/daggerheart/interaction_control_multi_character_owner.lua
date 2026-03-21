local scn = Scenario.new("interaction_control_multi_character_owner")

-- One participant may own multiple acting characters and still submit one slot
-- that covers the whole acting set.
scn:campaign{
  name = "Interaction Control Multi Character Owner",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "interaction"
}

scn:participant({name = "Guide", role = "GM"})
local rhea = scn:participant({name = "Rhea"})
rhea:character({name = "Aria"})
rhea:character({name = "Sable"})

scn:start_session("Courtyard")
scn:create_scene{
  name = "Moonlit Courtyard",
  description = "Aria and Sable confer beside the fountain while the keep sleeps.",
  characters = {"Aria", "Sable"}
}
scn:interaction_set_session_gm_authority({participant = "Guide"})
scn:interaction_open_scene_player_phase{
  as = "Guide",
  scene = "Moonlit Courtyard",
  interaction = {
    title = "Moonlit Courtyard",
    beats = {
      {type = "prompt", text = "The fountain masks your voices for now. How do the two of you move next?"},
    },
  },
  characters = {"Aria", "Sable"}
}
scn:interaction_expect{
  as = "Rhea",
  phase_status = "PLAYERS",
  control_mode = "PLAYERS",
  acting_characters = {"Aria", "Sable"},
  acting_participants = {"Rhea"},
  recommended_transition = "SUBMIT_SCENE_PLAYER_ACTION"
}
scn:interaction_submit_scene_player_action{
  as = "Rhea",
  scene = "Moonlit Courtyard",
  summary = "Aria keeps watch at the archway while Sable slips to the fountain to inspect the satchel.",
  characters = {"Aria", "Sable"}
}
scn:interaction_yield_scene_player_phase({as = "Rhea", scene = "Moonlit Courtyard"})
scn:interaction_expect{
  as = "Guide",
  phase_status = "GM_REVIEW",
  control_mode = "GM_REVIEW",
  recommended_transition = "RESOLVE_SCENE_PLAYER_REVIEW",
  slots = {
    {
      participant = "Rhea",
      summary = "Aria keeps watch at the archway while Sable slips to the fountain to inspect the satchel.",
      characters = {"Aria", "Sable"},
      yielded = true,
      review_status = "UNDER_REVIEW"
    }
  }
}
scn:interaction_resolve_scene_player_review{
  as = "Guide",
  scene = "Moonlit Courtyard",
  return_to_gm = true,
  interaction = {
    title = "Courtyard Beat Resolved",
    beats = {
      {type = "resolution", text = "The courtyard exchange resolves and the scene returns to the GM."},
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
