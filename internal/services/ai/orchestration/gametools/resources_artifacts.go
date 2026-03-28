package gametools

import (
	"context"
	"fmt"
	"strings"
)

func (s *DirectSession) readCampaignArtifact(ctx context.Context, uri string) (string, error) {
	campaignID, artifactPath, err := parseArtifactURI(uri)
	if err != nil {
		return "", err
	}
	if s.clients.Artifact == nil {
		return "", fmt.Errorf("artifact manager is not configured")
	}

	record, err := s.clients.Artifact.GetArtifact(ctx, campaignID, artifactPath)
	if err != nil {
		return "", fmt.Errorf("campaign artifact get failed: %w", err)
	}
	return marshalIndent(artifactFromRecord(record, true))
}

func matchCampaignArtifactURI(uri string) bool {
	return strings.HasPrefix(uri, "campaign://") && strings.Contains(uri, "/artifacts/")
}
