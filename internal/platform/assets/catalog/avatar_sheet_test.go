package catalog

import "testing"

func TestAvatarSheetBySetID_V1IncludesPortraitSlices(t *testing.T) {
	sheet, ok := AvatarSheetBySetID(AvatarSetV1)
	if !ok {
		t.Fatalf("expected avatar sheet for %q", AvatarSetV1)
	}
	if sheet.WidthPX != 1024 {
		t.Fatalf("sheet width = %d, want %d", sheet.WidthPX, 1024)
	}
	if sheet.HeightPX != 1536 {
		t.Fatalf("sheet height = %d, want %d", sheet.HeightPX, 1536)
	}
	if len(sheet.Portraits) != 4 {
		t.Fatalf("portrait count = %d, want %d", len(sheet.Portraits), 4)
	}
	second, ok := sheet.Portraits[2]
	if !ok {
		t.Fatalf("expected portrait slot %d", 2)
	}
	if second.X != 512 || second.Y != 0 || second.WidthPX != 512 || second.HeightPX != 768 {
		t.Fatalf("slot 2 rect = %+v, want x=512 y=0 w=512 h=768", second)
	}
}

func TestResolveAvatarPortraitSlot_RoleRules(t *testing.T) {
	userSlot, err := ResolveAvatarPortraitSlot("user", "user-1")
	if err != nil {
		t.Fatalf("resolve user slot: %v", err)
	}
	if userSlot != 1 {
		t.Fatalf("user slot = %d, want %d", userSlot, 1)
	}

	participantSlot, err := ResolveAvatarPortraitSlot("participant", "participant-1")
	if err != nil {
		t.Fatalf("resolve participant slot: %v", err)
	}
	if participantSlot != 1 {
		t.Fatalf("participant slot = %d, want %d", participantSlot, 1)
	}

	charSlotA, err := ResolveAvatarPortraitSlot("character", "character-1")
	if err != nil {
		t.Fatalf("resolve character slot: %v", err)
	}
	charSlotB, err := ResolveAvatarPortraitSlot("character", "character-1")
	if err != nil {
		t.Fatalf("resolve character slot again: %v", err)
	}
	if charSlotA != charSlotB {
		t.Fatalf("character slot must be deterministic, got %d then %d", charSlotA, charSlotB)
	}
	if charSlotA < 2 || charSlotA > 4 {
		t.Fatalf("character slot = %d, want one of [2,3,4]", charSlotA)
	}
}
