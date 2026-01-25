package domain

import (
	"strings"
	"testing"
)

func TestParseCampaignIDFromResourceURI(t *testing.T) {
	tests := []struct {
		name        string
		uri         string
		resourceType string
		wantID      string
		wantErr     bool
		errContains string
	}{
		// Valid cases
		{
			name:         "valid participants URI",
			uri:          "campaign://camp-123/participants",
			resourceType: "participants",
			wantID:       "camp-123",
			wantErr:      false,
		},
		{
			name:         "valid actors URI",
			uri:          "campaign://camp-456/actors",
			resourceType: "actors",
			wantID:       "camp-456",
			wantErr:      false,
		},
		{
			name:         "valid sessions URI",
			uri:          "campaign://camp-789/sessions",
			resourceType: "sessions",
			wantID:       "camp-789",
			wantErr:      false,
		},
		{
			name:         "valid URI with long campaign ID",
			uri:          "campaign://campaign-with-very-long-id-12345/participants",
			resourceType: "participants",
			wantID:       "campaign-with-very-long-id-12345",
			wantErr:      false,
		},
		{
			name:         "valid URI with whitespace trimmed",
			uri:          "campaign://  camp-123  /participants",
			resourceType: "participants",
			wantID:       "camp-123",
			wantErr:      false,
		},

		// Invalid prefix cases
		{
			name:         "missing prefix",
			uri:          "camp-123/participants",
			resourceType: "participants",
			wantErr:      true,
			errContains:  "URI must start with",
		},
		{
			name:         "wrong prefix",
			uri:          "http://camp-123/participants",
			resourceType: "participants",
			wantErr:      true,
			errContains:  "URI must start with",
		},

		// Invalid suffix cases
		{
			name:         "missing suffix",
			uri:          "campaign://camp-123",
			resourceType: "participants",
			wantErr:      true,
			errContains:  "URI must end with",
		},
		{
			name:         "wrong suffix",
			uri:          "campaign://camp-123/actors",
			resourceType: "participants",
			wantErr:      true,
			errContains:  "URI must end with",
		},
		{
			name:         "wrong resource type",
			uri:          "campaign://camp-123/sessions",
			resourceType: "participants",
			wantErr:      true,
			errContains:  "URI must end with",
		},

		// Empty campaign ID cases
		{
			name:         "empty campaign ID",
			uri:          "campaign:///participants",
			resourceType: "participants",
			wantErr:      true,
			errContains:  "campaign ID is required",
		},
		{
			name:         "only whitespace campaign ID",
			uri:          "campaign://   /participants",
			resourceType: "participants",
			wantErr:      true,
			errContains:  "campaign ID is required",
		},

		// Placeholder rejection
		{
			name:         "placeholder participants",
			uri:          "campaign://_/participants",
			resourceType: "participants",
			wantErr:      true,
			errContains:  "campaign ID placeholder '_' is not a valid campaign ID",
		},
		{
			name:         "placeholder actors",
			uri:          "campaign://_/actors",
			resourceType: "actors",
			wantErr:      true,
			errContains:  "campaign ID placeholder '_' is not a valid campaign ID",
		},
		{
			name:         "placeholder sessions",
			uri:          "campaign://_/sessions",
			resourceType: "sessions",
			wantErr:      true,
			errContains:  "campaign ID placeholder '_' is not a valid campaign ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, err := parseCampaignIDFromResourceURI(tt.uri, tt.resourceType)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseCampaignIDFromResourceURI() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("parseCampaignIDFromResourceURI() error = %v, want error containing %q", err, tt.errContains)
				}
				if gotID != "" {
					t.Errorf("parseCampaignIDFromResourceURI() gotID = %q, want empty string on error", gotID)
				}
			} else {
				if err != nil {
					t.Errorf("parseCampaignIDFromResourceURI() unexpected error = %v", err)
					return
				}
				if gotID != tt.wantID {
					t.Errorf("parseCampaignIDFromResourceURI() gotID = %q, want %q", gotID, tt.wantID)
				}
			}
		})
	}
}
