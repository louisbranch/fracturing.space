package worldbuilder

import (
	"math/rand"
	"strings"
	"testing"
)

func TestWorldBuilderCampaignNameFormat(t *testing.T) {
	builder := New(rand.New(rand.NewSource(1)))
	name := builder.CampaignName()
	if !strings.HasPrefix(name, "The ") {
		t.Fatalf("expected campaign name to start with 'The ', got %q", name)
	}
}

func TestWorldBuilderCharacterNameFormat(t *testing.T) {
	builder := New(rand.New(rand.NewSource(2)))
	name := builder.CharacterName()
	parts := strings.Split(name, " ")
	if len(parts) != 2 {
		t.Fatalf("expected character name to have 2 parts, got %q", name)
	}
}

func TestWorldBuilderParticipantNameFromList(t *testing.T) {
	builder := New(rand.New(rand.NewSource(3)))
	name := builder.ParticipantName()
	if !containsString(participantNames, name) {
		t.Fatalf("expected participant name %q to be in list", name)
	}
}

func TestWorldBuilderSessionNameFormat(t *testing.T) {
	builder := New(rand.New(rand.NewSource(4)))
	name := builder.SessionName(7)
	if !strings.HasPrefix(name, "Session 7: ") {
		t.Fatalf("expected session name to include sequence, got %q", name)
	}
}

func TestWorldBuilderThemePromptFromList(t *testing.T) {
	builder := New(rand.New(rand.NewSource(5)))
	value := builder.ThemePrompt()
	if !containsString(themePrompts, value) {
		t.Fatalf("expected theme prompt %q to be in list", value)
	}
}

func TestWorldBuilderNoteContentFromList(t *testing.T) {
	builder := New(rand.New(rand.NewSource(6)))
	value := builder.NoteContent()
	if !containsString(noteTemplates, value) {
		t.Fatalf("expected note content %q to be in list", value)
	}
}

func TestWorldBuilderNPCDescriptionFromList(t *testing.T) {
	builder := New(rand.New(rand.NewSource(7)))
	value := builder.NPCDescription()
	if !containsString(npcDescriptions, value) {
		t.Fatalf("expected NPC description %q to be in list", value)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
