package memorydoc

import "testing"

func TestSectionRead(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		heading   string
		wantBody  string
		wantFound bool
	}{
		{name: "empty document", content: "", heading: "NPCs", wantBody: "", wantFound: false},
		{name: "empty heading", content: "## NPCs\n\nSome NPC info\n", heading: "", wantBody: "", wantFound: false},
		{name: "exact match", content: "## NPCs\n\nBartender at the Rusty Nail.\n", heading: "NPCs", wantBody: "Bartender at the Rusty Nail.", wantFound: true},
		{name: "case insensitive match", content: "## Plot Hooks\n\nThe harbor debt is due.\n", heading: "plot hooks", wantBody: "The harbor debt is due.", wantFound: true},
		{name: "heading not found", content: "## NPCs\n\nBartender.\n", heading: "World State", wantBody: "", wantFound: false},
		{name: "multi-section returns correct one", content: "## NPCs\n\nBartender.\n\n## Plot Hooks\n\nDebt at dawn.\n\n## World State\n\nHarbor town.\n", heading: "Plot Hooks", wantBody: "Debt at dawn.", wantFound: true},
		{name: "preamble before first heading", content: "Campaign memory for Harbor Run.\n\n## NPCs\n\nBartender.\n", heading: "NPCs", wantBody: "Bartender.", wantFound: true},
		{name: "section with no body", content: "## NPCs\n## Plot Hooks\n\nDebt.\n", heading: "NPCs", wantBody: "", wantFound: true},
		{name: "multiline body", content: "## NPCs\n\n- Bartender at the Rusty Nail\n- Harbor master\n- The Black Lantern\n", heading: "NPCs", wantBody: "- Bartender at the Rusty Nail\n- Harbor master\n- The Black Lantern", wantFound: true},
		{name: "whitespace heading trimmed", content: "## NPCs\n\nBartender.\n", heading: "  NPCs  ", wantBody: "Bartender.", wantFound: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, found := SectionRead(tt.content, tt.heading)
			if found != tt.wantFound {
				t.Errorf("found = %v, want %v", found, tt.wantFound)
			}
			if body != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestSectionUpdate(t *testing.T) {
	tests := []struct {
		name    string
		content string
		heading string
		body    string
		want    string
	}{
		{name: "empty document appends new section", content: "", heading: "NPCs", body: "Bartender at the Rusty Nail.", want: "## NPCs\n\nBartender at the Rusty Nail.\n"},
		{name: "empty heading is a no-op", content: "existing content\n", heading: "", body: "should not appear", want: "existing content\n"},
		{name: "replace existing section", content: "## NPCs\n\nOld bartender.\n\n## Plot Hooks\n\nDebt at dawn.\n", heading: "NPCs", body: "New bartender at the Rusty Nail.", want: "## NPCs\n\nNew bartender at the Rusty Nail.\n\n## Plot Hooks\n\nDebt at dawn.\n"},
		{name: "case insensitive replace", content: "## Plot Hooks\n\nOld hook.\n", heading: "plot hooks", body: "New hook: the debt.", want: "## Plot Hooks\n\nNew hook: the debt.\n"},
		{name: "append new section to existing document", content: "## NPCs\n\nBartender.\n", heading: "World State", body: "Harbor town is tense.", want: "## NPCs\n\nBartender.\n\n## World State\n\nHarbor town is tense.\n"},
		{name: "preserve preamble", content: "Campaign: Harbor Run\n\n## NPCs\n\nBartender.\n", heading: "NPCs", body: "Updated bartender.", want: "Campaign: Harbor Run\n\n## NPCs\n\nUpdated bartender.\n"},
		{name: "replace with empty body", content: "## NPCs\n\nBartender.\n\n## Plot Hooks\n\nDebt.\n", heading: "NPCs", body: "", want: "## NPCs\n\n## Plot Hooks\n\nDebt.\n"},
		{name: "replace section in middle preserves order", content: "## NPCs\n\nA.\n\n## Plot Hooks\n\nB.\n\n## World State\n\nC.\n", heading: "Plot Hooks", body: "Updated B.", want: "## NPCs\n\nA.\n\n## Plot Hooks\n\nUpdated B.\n\n## World State\n\nC.\n"},
		{name: "multiline body replacement", content: "## NPCs\n\nOld.\n", heading: "NPCs", body: "- Bartender\n- Harbor master\n- Black Lantern", want: "## NPCs\n\n- Bartender\n- Harbor master\n- Black Lantern\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SectionUpdate(tt.content, tt.heading, tt.body)
			if got != tt.want {
				t.Errorf("got:\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}

func TestParseH2Sections(t *testing.T) {
	content := "Preamble text\n\n## Alpha\n\nAlpha body.\n\n## Beta\n\nBeta body.\n"
	sections := parseH2Sections(content)
	if len(sections) != 3 {
		t.Fatalf("len(sections) = %d, want 3", len(sections))
	}
	if sections[0].heading != "" {
		t.Errorf("sections[0].heading = %q, want preamble", sections[0].heading)
	}
	if sections[1].heading != "Alpha" {
		t.Errorf("sections[1].heading = %q, want Alpha", sections[1].heading)
	}
	if sections[2].heading != "Beta" {
		t.Errorf("sections[2].heading = %q, want Beta", sections[2].heading)
	}
}
