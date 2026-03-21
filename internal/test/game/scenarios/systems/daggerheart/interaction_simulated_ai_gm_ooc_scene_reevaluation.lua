local scn = Scenario.new("interaction_simulated_ai_gm_ooc_scene_reevaluation")

-- Exercise the post-OOC resolution path where the GM changes course and
-- replaces the interrupted scene with a newly framed beat elsewhere.
scn:campaign{
  name = "Interaction Simulated AI GM OOC Scene Reevaluation",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "interaction"
}

scn:participant({name = "Guide", role = "GM"})
scn:participant({name = "Rhea"}):character({name = "Aria"})
scn:participant({name = "Bryn"}):character({name = "Corin"})

scn:start_session("Vault Run")
scn:create_scene{
  name = "Sealed Vault",
  description = "An old vault door hums with warding magic and a narrow seam of light.",
  characters = {"Aria", "Corin"}
}
scn:create_scene{
  name = "Collapsed Antechamber",
  description = "Broken columns and a drift of dust offer a defensible fallback from the warded door.",
  characters = {"Aria", "Corin"},
  activate = false
}
scn:interaction_set_session_gm_authority({participant = "Guide"})
scn:interaction_open_scene_player_phase{
  as = "Guide",
  scene = "Sealed Vault",
  interaction = {
    title = "Warded Vault",
    beats = {
      {type = "prompt", text = "The ward crackles hotter with every step toward the seam. What do you do?"},
    },
  },
  characters = {"Aria", "Corin"}
}

-- A player interrupts with OOC, and the table decides the scene should change.
scn:interaction_open_session_ooc({as = "Rhea", reason = "Check whether the vault is even a viable target right now."})
scn:interaction_post_session_ooc({as = "Rhea", body = "If the ward surges on any approach, Aria wants to fall back instead of forcing the issue."})
scn:interaction_post_session_ooc({as = "Guide", body = "That makes sense. The ward is escalating too fast to tackle head-on in this beat."})
scn:interaction_post_session_ooc({as = "Bryn", body = "Then Corin should pull everyone back into the antechamber and regroup."})
scn:interaction_mark_ooc_ready_to_resume({as = "Rhea"})
scn:interaction_mark_ooc_ready_to_resume({as = "Bryn"})
scn:interaction_expect{
  active_scene = "Sealed Vault",
  phase_status = "PLAYERS",
  ooc_open = true,
  ooc_requested_by = "Rhea",
  ooc_interrupted_scene = "Sealed Vault",
  ooc_interrupted_phase_status = "PLAYERS",
  ooc_ready = {"Rhea", "Bryn"}
}

-- Simulate the AI GM reevaluating the scene and moving the action elsewhere.
scn:interaction_resolve_session_ooc{
  as = "Guide",
  scene = "Collapsed Antechamber",
  interaction = {
    title = "Retreat To The Antechamber",
    beats = {
      {type = "fiction", text = "The ward's shriek drives you back before the seam can be tested. Dust rains from the ceiling as you retreat into the collapsed antechamber."},
      {type = "prompt", text = "Safe from the immediate surge, what do you salvage and how do you regroup in the antechamber?"},
    },
  },
  characters = {"Aria", "Corin"}
}
scn:interaction_expect{
  active_scene = "Collapsed Antechamber",
  phase_status = "PLAYERS",
  prompt = "Safe from the immediate surge, what do you salvage and how do you regroup in the antechamber?",
  acting_characters = {"Aria", "Corin"},
  acting_participants = {"Rhea", "Bryn"},
  ooc_open = false,
  ooc_resolution_pending = false,
  gm_authority = "Guide"
}

scn:end_session()

return scn
