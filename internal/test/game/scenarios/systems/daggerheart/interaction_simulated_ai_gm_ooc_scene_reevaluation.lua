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
  characters = {"Aria", "Corin"}
}
scn:interaction_set_gm_authority({participant = "Guide"})
scn:interaction_set_active_scene({scene = "Sealed Vault"})
scn:interaction_start_player_phase{
  scene = "Sealed Vault",
  frame_text = "The ward crackles hotter with every step toward the seam. What do you do?",
  characters = {"Aria", "Corin"}
}

-- A player interrupts with OOC, and the table decides the scene should change.
scn:interaction_pause_ooc({as = "Rhea", reason = "Check whether the vault is even a viable target right now."})
scn:interaction_post_ooc({as = "Rhea", body = "If the ward surges on any approach, Aria wants to fall back instead of forcing the issue."})
scn:interaction_post_ooc({as = "Guide", body = "That makes sense. The ward is escalating too fast to tackle head-on in this beat."})
scn:interaction_post_ooc({as = "Bryn", body = "Then Corin should pull everyone back into the antechamber and regroup."})
scn:interaction_ready_ooc({as = "Rhea"})
scn:interaction_ready_ooc({as = "Bryn"})
scn:interaction_expect{
  active_scene = "Sealed Vault",
  phase_status = "PLAYERS",
  ooc_open = true,
  ooc_requested_by = "Rhea",
  ooc_interrupted_scene = "Sealed Vault",
  ooc_interrupted_phase_status = "PLAYERS",
  ooc_ready = {"Rhea", "Bryn"}
}

scn:interaction_resume_ooc()
scn:interaction_expect{
  active_scene = "Sealed Vault",
  phase_status = "PLAYERS",
  ooc_open = false,
  ooc_resolution_pending = true,
  gm_authority = "Guide"
}

-- Simulate the AI GM reevaluating the scene and moving the action elsewhere.
scn:interaction_resolve_interrupted_phase{
  as = "Guide",
  scene = "Collapsed Antechamber",
  gm_output_text = "The ward's shriek drives you back before the seam can be tested. Dust rains from the ceiling as you retreat into the collapsed antechamber.",
  frame_text = "Safe from the immediate surge, what do you salvage and how do you regroup in the antechamber?",
  characters = {"Aria", "Corin"}
}
scn:interaction_expect{
  active_scene = "Collapsed Antechamber",
  phase_status = "PLAYERS",
  frame_text = "Safe from the immediate surge, what do you salvage and how do you regroup in the antechamber?",
  acting_characters = {"Aria", "Corin"},
  acting_participants = {"Rhea", "Bryn"},
  ooc_open = false,
  ooc_resolution_pending = false,
  gm_authority = "Guide"
}

scn:end_session()

return scn
