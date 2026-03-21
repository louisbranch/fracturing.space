local scn = Scenario.new("interaction_control_ooc_resolution")

-- ResolveSessionOOC should cleanly support resume, return-to-GM, and
-- open-player-phase outcomes from the paused OOC state.
scn:campaign{
  name = "Interaction Control OOC Resolution",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "interaction"
}

scn:participant({name = "Guide", role = "GM"})
scn:participant({name = "Rhea"}):character({name = "Aria"})

scn:start_session("Reevaluation")
scn:create_scene{
  name = "Sealed Vault",
  description = "The vault ward surges whenever Aria nears the seam.",
  characters = {"Aria"}
}
scn:interaction_set_session_gm_authority({participant = "Guide"})

-- Outcome 1: resume the interrupted player phase.
scn:interaction_open_scene_player_phase{
  as = "Guide",
  scene = "Sealed Vault",
  interaction = {
    title = "Ward Study",
    beats = {
      {type = "prompt", text = "Aria, what do you test first about the ward?"},
    },
  },
  characters = {"Aria"}
}
scn:interaction_open_session_ooc({as = "Rhea", reason = "Clarify the ward trigger."})
scn:interaction_resolve_session_ooc({as = "Guide", resume_interrupted_phase = true})
scn:interaction_expect{
  as = "Rhea",
  phase_status = "PLAYERS",
  control_mode = "PLAYERS",
  recommended_transition = "SUBMIT_SCENE_PLAYER_ACTION",
  ooc_open = false
}

-- Outcome 2: return the scene to GM control.
scn:interaction_open_session_ooc({as = "Rhea", reason = "The group wants to reassess before acting."})
scn:interaction_resolve_session_ooc({as = "Guide", return_to_gm = true, scene = "Sealed Vault"})
scn:interaction_expect{
  as = "Guide",
  phase_status = "GM",
  control_mode = "GM",
  recommended_transition = "OPEN_SCENE_PLAYER_PHASE",
  ooc_open = false
}

-- Outcome 3: replace the interrupted beat with a newly opened player phase.
scn:interaction_open_scene_player_phase{
  as = "Guide",
  scene = "Sealed Vault",
  interaction = {
    title = "Return To The Seam",
    beats = {
      {type = "prompt", text = "Aria, how do you approach the seam after regrouping?"},
    },
  },
  characters = {"Aria"}
}
scn:interaction_open_session_ooc({as = "Rhea", reason = "Shift the beat after the ruling."})
scn:interaction_resolve_session_ooc{
  as = "Guide",
  scene = "Sealed Vault",
  interaction = {
    title = "New Approach",
    beats = {
      {type = "fiction", text = "The group abandons the seam and circles toward the roof vent instead."},
      {type = "prompt", text = "Aria, how do you reach the roof vent before the ward surges again?"},
    },
  },
  characters = {"Aria"}
}
scn:interaction_expect{
  as = "Rhea",
  phase_status = "PLAYERS",
  control_mode = "PLAYERS",
  prompt = "Aria, how do you reach the roof vent before the ward surges again?",
  recommended_transition = "SUBMIT_SCENE_PLAYER_ACTION",
  ooc_open = false
}
scn:interaction_resolve_session_ooc{
  as = "Guide",
  return_to_gm = true,
  scene = "Sealed Vault",
  expect_error = {code = "FAILED_PRECONDITION"}
}

scn:end_session()

return scn
