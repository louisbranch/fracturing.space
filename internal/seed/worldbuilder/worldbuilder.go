// Package worldbuilder provides TTRPG-themed name and content generation
// for seeding the database with diverse, fantasy-appropriate data.
package worldbuilder

import (
	"fmt"
	"math/rand"
)

// WorldBuilder generates TTRPG-themed names and content.
type WorldBuilder struct {
	rng *rand.Rand
}

// New creates a WorldBuilder with the given random source.
func New(rng *rand.Rand) *WorldBuilder {
	return &WorldBuilder{rng: rng}
}

// CampaignName generates a fantasy campaign name like "The Jade Oasis".
func (w *WorldBuilder) CampaignName() string {
	adj := campaignAdjectives[w.rng.Intn(len(campaignAdjectives))]
	noun := campaignNouns[w.rng.Intn(len(campaignNouns))]
	return fmt.Sprintf("The %s %s", adj, noun)
}

// CharacterName generates a culturally diverse fantasy character name.
func (w *WorldBuilder) CharacterName() string {
	first := characterFirstNames[w.rng.Intn(len(characterFirstNames))]
	last := characterSurnames[w.rng.Intn(len(characterSurnames))]
	return fmt.Sprintf("%s %s", first, last)
}

// ParticipantName generates a modern, globally diverse player name.
func (w *WorldBuilder) ParticipantName() string {
	return participantNames[w.rng.Intn(len(participantNames))]
}

// SessionName generates a session name with the given sequence number.
func (w *WorldBuilder) SessionName(seq int) string {
	title := sessionTitles[w.rng.Intn(len(sessionTitles))]
	return fmt.Sprintf("Session %d: %s", seq, title)
}

// ThemePrompt generates a campaign theme/setting description.
func (w *WorldBuilder) ThemePrompt() string {
	return themePrompts[w.rng.Intn(len(themePrompts))]
}

// NoteContent generates random GM/player note content.
func (w *WorldBuilder) NoteContent() string {
	return noteTemplates[w.rng.Intn(len(noteTemplates))]
}

// NPCDescription generates a brief NPC description.
func (w *WorldBuilder) NPCDescription() string {
	return npcDescriptions[w.rng.Intn(len(npcDescriptions))]
}
