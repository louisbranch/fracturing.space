package gametools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/ai/campaignartifact"
)

type artifactManagerStub struct {
	artifact campaignartifact.Artifact
	err      error
}

func (stub artifactManagerStub) ListArtifacts(context.Context, string) ([]campaignartifact.Artifact, error) {
	return nil, nil
}

func (stub artifactManagerStub) GetArtifact(context.Context, string, string) (campaignartifact.Artifact, error) {
	if stub.err != nil {
		return campaignartifact.Artifact{}, stub.err
	}
	return stub.artifact, nil
}

func (stub artifactManagerStub) UpsertArtifact(context.Context, string, string, string) (campaignartifact.Artifact, error) {
	return campaignartifact.Artifact{}, nil
}

func TestReadResourceContextCurrent(t *testing.T) {
	session := NewDirectSession(Clients{}, SessionContext{
		CampaignID:    "camp-1",
		SessionID:     "sess-1",
		ParticipantID: "part-1",
	})

	value, err := session.ReadResource(context.Background(), "context://current")
	if err != nil {
		t.Fatalf("ReadResource(context://current) error = %v", err)
	}

	var payload struct {
		Context struct {
			CampaignID    string `json:"campaign_id"`
			SessionID     string `json:"session_id"`
			ParticipantID string `json:"participant_id"`
		} `json:"context"`
	}
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		t.Fatalf("unmarshal context payload: %v", err)
	}
	if payload.Context.CampaignID != "camp-1" || payload.Context.SessionID != "sess-1" || payload.Context.ParticipantID != "part-1" {
		t.Fatalf("context payload = %#v", payload.Context)
	}
}

func TestReadResourceArtifactDispatch(t *testing.T) {
	session := NewDirectSession(Clients{
		Artifact: artifactManagerStub{
			artifact: campaignartifact.Artifact{
				CampaignID: "camp-1",
				Path:       "notes/gm.md",
				Content:    "keep the clock moving",
			},
		},
	}, SessionContext{})

	value, err := session.ReadResource(context.Background(), "campaign://camp-1/artifacts/notes/gm.md")
	if err != nil {
		t.Fatalf("ReadResource(artifact) error = %v", err)
	}
	if !strings.Contains(value, "\"path\": \"notes/gm.md\"") {
		t.Fatalf("artifact payload = %s, want path field", value)
	}
	if !strings.Contains(value, "\"content\": \"keep the clock moving\"") {
		t.Fatalf("artifact payload = %s, want content field", value)
	}
}

func TestReadResourceRejectsMalformedParticipantURI(t *testing.T) {
	session := NewDirectSession(Clients{}, SessionContext{})

	_, err := session.ReadResource(context.Background(), "campaign:///participants")
	if err == nil {
		t.Fatal("ReadResource malformed participants URI error = nil, want error")
	}
	if !strings.Contains(err.Error(), "campaign ID is required") {
		t.Fatalf("ReadResource malformed participants URI error = %v", err)
	}
}

func TestReadResourceRejectsUnknownURI(t *testing.T) {
	session := NewDirectSession(Clients{}, SessionContext{})

	_, err := session.ReadResource(context.Background(), "mystery://unsupported")
	if err == nil {
		t.Fatal("ReadResource unknown URI error = nil, want error")
	}
	if !strings.Contains(err.Error(), "unknown resource URI") {
		t.Fatalf("ReadResource unknown URI error = %v", err)
	}
}

func TestParseSceneListURI(t *testing.T) {
	campaignID, sessionID, err := parseSceneListURI("campaign://camp-1/sessions/sess-1/scenes")
	if err != nil {
		t.Fatalf("parseSceneListURI error = %v", err)
	}
	if campaignID != "camp-1" || sessionID != "sess-1" {
		t.Fatalf("parseSceneListURI = (%q, %q), want (camp-1, sess-1)", campaignID, sessionID)
	}
}

func TestParseArtifactURIRejectsMissingPath(t *testing.T) {
	_, _, err := parseArtifactURI("campaign://camp-1/artifacts/")
	if err == nil {
		t.Fatal("parseArtifactURI missing path error = nil, want error")
	}
	if !strings.Contains(err.Error(), "campaign and artifact path are required") {
		t.Fatalf("parseArtifactURI missing path error = %v", err)
	}
}
