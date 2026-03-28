package gametools

import (
	"testing"
)

func TestParseArtifactURIEdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		uri      string
		wantCamp string
		wantPath string
		wantErr  bool
	}{
		{"valid", "campaign://camp-1/artifacts/notes/intro.md", "camp-1", "notes/intro.md", false},
		{"missing prefix", "file://camp-1/artifacts/x", "", "", true},
		{"no artifacts segment", "campaign://camp-1/other", "", "", true},
		{"empty campaign", "campaign:///artifacts/x", "", "", true},
		{"empty path", "campaign://camp-1/artifacts/", "", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			camp, path, err := parseArtifactURI(tc.uri)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseArtifactURI(%q) err = %v, wantErr %v", tc.uri, err, tc.wantErr)
			}
			if camp != tc.wantCamp {
				t.Errorf("campaign = %q, want %q", camp, tc.wantCamp)
			}
			if path != tc.wantPath {
				t.Errorf("path = %q, want %q", path, tc.wantPath)
			}
		})
	}
}

func TestParseCampaignIDFromSuffixURI(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		uri     string
		suffix  string
		want    string
		wantErr bool
	}{
		{"valid participants", "campaign://camp-1/participants", "participants", "camp-1", false},
		{"valid sessions", "campaign://camp-2/sessions", "sessions", "camp-2", false},
		{"missing prefix", "game://camp-1/participants", "participants", "", true},
		{"wrong suffix", "campaign://camp-1/characters", "participants", "", true},
		{"empty campaign", "campaign:///participants", "participants", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseCampaignIDFromSuffixURI(tc.uri, tc.suffix)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseCampaignIDFromSuffixURI(%q, %q) err = %v, wantErr %v", tc.uri, tc.suffix, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("got = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseSceneListURIEdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		uri      string
		wantCamp string
		wantSess string
		wantErr  bool
	}{
		{"valid", "campaign://camp-1/sessions/sess-1/scenes", "camp-1", "sess-1", false},
		{"missing prefix", "file://camp-1/sessions/sess-1/scenes", "", "", true},
		{"wrong structure", "campaign://camp-1/scenes", "", "", true},
		{"empty campaign", "campaign:///sessions/sess-1/scenes", "", "", true},
		{"empty session", "campaign://camp-1/sessions//scenes", "", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			camp, sess, err := parseSceneListURI(tc.uri)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseSceneListURI(%q) err = %v, wantErr %v", tc.uri, err, tc.wantErr)
			}
			if camp != tc.wantCamp {
				t.Errorf("campaign = %q, want %q", camp, tc.wantCamp)
			}
			if sess != tc.wantSess {
				t.Errorf("session = %q, want %q", sess, tc.wantSess)
			}
		})
	}
}
