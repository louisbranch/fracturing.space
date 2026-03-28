package gametools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/campaignartifact"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext/referencecorpus"
)

type artifactManagerTestStub struct {
	listArtifacts  []campaignartifact.Artifact
	listErr        error
	getArtifact    campaignartifact.Artifact
	getErr         error
	upsertArtifact campaignartifact.Artifact
	upsertErr      error

	lastCampaignID string
	lastPath       string
	lastContent    string
}

func (stub *artifactManagerTestStub) ListArtifacts(_ context.Context, campaignID string) ([]campaignartifact.Artifact, error) {
	stub.lastCampaignID = campaignID
	if stub.listErr != nil {
		return nil, stub.listErr
	}
	return append([]campaignartifact.Artifact(nil), stub.listArtifacts...), nil
}

func (stub *artifactManagerTestStub) GetArtifact(_ context.Context, campaignID string, path string) (campaignartifact.Artifact, error) {
	stub.lastCampaignID = campaignID
	stub.lastPath = path
	if stub.getErr != nil {
		return campaignartifact.Artifact{}, stub.getErr
	}
	return stub.getArtifact, nil
}

func (stub *artifactManagerTestStub) UpsertArtifact(_ context.Context, campaignID string, path string, content string) (campaignartifact.Artifact, error) {
	stub.lastCampaignID = campaignID
	stub.lastPath = path
	stub.lastContent = content
	if stub.upsertErr != nil {
		return campaignartifact.Artifact{}, stub.upsertErr
	}
	return stub.upsertArtifact, nil
}

type referenceCorpusTestStub struct {
	searchResults []referencecorpus.SearchResult
	searchErr     error
	readDocument  referencecorpus.Document
	readErr       error

	lastSystem     string
	lastQuery      string
	lastMaxResults int
	lastDocumentID string
}

func (stub *referenceCorpusTestStub) Search(_ context.Context, system, query string, maxResults int) ([]referencecorpus.SearchResult, error) {
	stub.lastSystem = system
	stub.lastQuery = query
	stub.lastMaxResults = maxResults
	if stub.searchErr != nil {
		return nil, stub.searchErr
	}
	return append([]referencecorpus.SearchResult(nil), stub.searchResults...), nil
}

func (stub *referenceCorpusTestStub) Read(_ context.Context, system, documentID string) (referencecorpus.Document, error) {
	stub.lastSystem = system
	stub.lastDocumentID = documentID
	if stub.readErr != nil {
		return referencecorpus.Document{}, stub.readErr
	}
	return stub.readDocument, nil
}

func decodeToolOutput[T any](t *testing.T, result string) T {
	t.Helper()

	var value T
	if err := json.Unmarshal([]byte(result), &value); err != nil {
		t.Fatalf("unmarshal tool output: %v", err)
	}
	return value
}

func TestToolResultJSONMarshalsPayload(t *testing.T) {
	result, err := toolResultJSON(struct {
		Name string `json:"name"`
	}{Name: "oracle"})
	if err != nil {
		t.Fatalf("toolResultJSON() error = %v", err)
	}

	payload := decodeToolOutput[struct {
		Name string `json:"name"`
	}](t, result.Output)
	if payload.Name != "oracle" {
		t.Fatalf("name = %q, want oracle", payload.Name)
	}
}

func TestDirectSessionCloseIsNoOp(t *testing.T) {
	session := NewDirectSession(Clients{}, SessionContext{})
	if err := session.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestArtifactListUsesResolvedCampaignIDAndShapesResponse(t *testing.T) {
	manager := &artifactManagerTestStub{
		listArtifacts: []campaignartifact.Artifact{{
			CampaignID: "camp-1",
			Path:       "notes/gm.md",
			ReadOnly:   true,
			CreatedAt:  time.Date(2026, time.March, 1, 14, 0, 0, 0, time.FixedZone("EST", -5*60*60)),
			UpdatedAt:  time.Date(2026, time.March, 2, 15, 30, 0, 0, time.FixedZone("CET", 1*60*60)),
		}},
	}
	session := NewDirectSession(Clients{Artifact: manager}, SessionContext{CampaignID: "camp-1"})

	result, err := session.artifactList(context.Background(), []byte(`{}`))
	if err != nil {
		t.Fatalf("artifactList() error = %v", err)
	}

	payload := decodeToolOutput[artifactListResult](t, result.Output)
	if payload.CampaignID != "camp-1" {
		t.Fatalf("campaign_id = %q, want camp-1", payload.CampaignID)
	}
	if len(payload.Artifacts) != 1 {
		t.Fatalf("artifacts length = %d, want 1", len(payload.Artifacts))
	}
	if payload.Artifacts[0].Content != "" {
		t.Fatalf("content = %q, want empty list payload", payload.Artifacts[0].Content)
	}
	if payload.Artifacts[0].CreatedAt != "2026-03-01T19:00:00Z" {
		t.Fatalf("created_at = %q", payload.Artifacts[0].CreatedAt)
	}
	if payload.Artifacts[0].UpdatedAt != "2026-03-02T14:30:00Z" {
		t.Fatalf("updated_at = %q", payload.Artifacts[0].UpdatedAt)
	}
	if manager.lastCampaignID != "camp-1" {
		t.Fatalf("campaign_id passed to manager = %q, want camp-1", manager.lastCampaignID)
	}
}

func TestArtifactHandlersValidateDependenciesAndMapErrors(t *testing.T) {
	t.Run("artifactGet requires campaign id", func(t *testing.T) {
		session := NewDirectSession(Clients{}, SessionContext{})

		_, err := session.artifactGet(context.Background(), []byte(`{"path":"notes/gm.md"}`))
		if err == nil || err.Error() != "campaign_id is required" {
			t.Fatalf("artifactGet() error = %v, want campaign_id is required", err)
		}
	})

	t.Run("artifactGet requires manager", func(t *testing.T) {
		session := NewDirectSession(Clients{}, SessionContext{CampaignID: "camp-1"})

		_, err := session.artifactGet(context.Background(), []byte(`{"path":"notes/gm.md"}`))
		if err == nil || err.Error() != "artifact manager is not configured" {
			t.Fatalf("artifactGet() error = %v, want manager error", err)
		}
	})

	t.Run("artifactGet wraps manager error", func(t *testing.T) {
		session := NewDirectSession(Clients{
			Artifact: &artifactManagerTestStub{getErr: errors.New("db offline")},
		}, SessionContext{CampaignID: "camp-1"})

		_, err := session.artifactGet(context.Background(), []byte(`{"path":"notes/gm.md"}`))
		if err == nil || !strings.Contains(err.Error(), "campaign artifact get failed: db offline") {
			t.Fatalf("artifactGet() error = %v", err)
		}
	})

	t.Run("artifactGet includes content", func(t *testing.T) {
		session := NewDirectSession(Clients{
			Artifact: &artifactManagerTestStub{getArtifact: campaignartifact.Artifact{
				CampaignID: "camp-1",
				Path:       "notes/gm.md",
				Content:    "hold the gate",
			}},
		}, SessionContext{CampaignID: "camp-1"})

		result, err := session.artifactGet(context.Background(), []byte(`{"path":"notes/gm.md"}`))
		if err != nil {
			t.Fatalf("artifactGet() error = %v", err)
		}

		payload := decodeToolOutput[artifactResult](t, result.Output)
		if payload.Content != "hold the gate" {
			t.Fatalf("content = %q, want hold the gate", payload.Content)
		}
	})

	t.Run("artifactUpsert wraps manager error", func(t *testing.T) {
		session := NewDirectSession(Clients{
			Artifact: &artifactManagerTestStub{upsertErr: errors.New("write denied")},
		}, SessionContext{CampaignID: "camp-1"})

		_, err := session.artifactUpsert(context.Background(), []byte(`{"path":"notes/gm.md","content":"updated"}`))
		if err == nil || !strings.Contains(err.Error(), "campaign artifact upsert failed: write denied") {
			t.Fatalf("artifactUpsert() error = %v", err)
		}
	})

	t.Run("artifactUpsert returns saved artifact", func(t *testing.T) {
		manager := &artifactManagerTestStub{upsertArtifact: campaignartifact.Artifact{
			CampaignID: "camp-1",
			Path:       "notes/gm.md",
			Content:    "updated",
		}}
		session := NewDirectSession(Clients{Artifact: manager}, SessionContext{CampaignID: "camp-1"})

		result, err := session.artifactUpsert(context.Background(), []byte(`{"path":"notes/gm.md","content":"updated"}`))
		if err != nil {
			t.Fatalf("artifactUpsert() error = %v", err)
		}

		payload := decodeToolOutput[artifactResult](t, result.Output)
		if payload.Content != "updated" {
			t.Fatalf("content = %q, want updated", payload.Content)
		}
		if manager.lastPath != "notes/gm.md" || manager.lastContent != "updated" {
			t.Fatalf("upsert call = (%q, %q), want (notes/gm.md, updated)", manager.lastPath, manager.lastContent)
		}
	})
}

func TestMemorySectionReadCoversValidationAndLookup(t *testing.T) {
	t.Run("rejects malformed json", func(t *testing.T) {
		session := NewDirectSession(Clients{}, SessionContext{})

		_, err := session.memorySectionRead(context.Background(), []byte(`{`))
		if err == nil || !strings.Contains(err.Error(), "unmarshal args") {
			t.Fatalf("memorySectionRead() error = %v", err)
		}
	})

	t.Run("requires campaign id", func(t *testing.T) {
		session := NewDirectSession(Clients{}, SessionContext{})

		_, err := session.memorySectionRead(context.Background(), []byte(`{"heading":"Loose Ends"}`))
		if err == nil || err.Error() != "campaign_id is required" {
			t.Fatalf("memorySectionRead() error = %v, want campaign id error", err)
		}
	})

	t.Run("requires manager", func(t *testing.T) {
		session := NewDirectSession(Clients{}, SessionContext{CampaignID: "camp-1"})

		_, err := session.memorySectionRead(context.Background(), []byte(`{"heading":"Loose Ends"}`))
		if err == nil || err.Error() != "artifact manager is not configured" {
			t.Fatalf("memorySectionRead() error = %v, want manager error", err)
		}
	})

	t.Run("wraps read error", func(t *testing.T) {
		session := NewDirectSession(Clients{
			Artifact: &artifactManagerTestStub{getErr: errors.New("storage unavailable")},
		}, SessionContext{CampaignID: "camp-1"})

		_, err := session.memorySectionRead(context.Background(), []byte(`{"heading":"Loose Ends"}`))
		if err == nil || !strings.Contains(err.Error(), "get memory artifact: storage unavailable") {
			t.Fatalf("memorySectionRead() error = %v", err)
		}
	})

	t.Run("returns found section", func(t *testing.T) {
		session := NewDirectSession(Clients{
			Artifact: &artifactManagerTestStub{getArtifact: campaignartifact.Artifact{
				Content: "## Loose Ends\n\nFind the broker.\n\n## Threats\n\nClock advances.\n",
			}},
		}, SessionContext{CampaignID: "camp-1"})

		result, err := session.memorySectionRead(context.Background(), []byte(`{"heading":"Loose Ends"}`))
		if err != nil {
			t.Fatalf("memorySectionRead() error = %v", err)
		}

		payload := decodeToolOutput[memorySectionResult](t, result.Output)
		if !payload.Found || payload.Content != "\nFind the broker.\n" {
			t.Fatalf("payload = %#v", payload)
		}
	})
}

func TestMemorySectionUpdatePersistsMergedDocument(t *testing.T) {
	t.Run("wraps write error", func(t *testing.T) {
		session := NewDirectSession(Clients{
			Artifact: &artifactManagerTestStub{
				getArtifact: campaignartifact.Artifact{Content: "## Loose Ends\n\nFind the broker.\n"},
				upsertErr:   errors.New("write failed"),
			},
		}, SessionContext{CampaignID: "camp-1"})

		_, err := session.memorySectionUpdate(context.Background(), []byte(`{"heading":"Loose Ends","content":"Protect the broker."}`))
		if err == nil || !strings.Contains(err.Error(), "upsert memory artifact: write failed") {
			t.Fatalf("memorySectionUpdate() error = %v", err)
		}
	})

	t.Run("updates requested section", func(t *testing.T) {
		manager := &artifactManagerTestStub{
			getArtifact: campaignartifact.Artifact{Content: "## Loose Ends\n\nFind the broker.\n"},
			upsertArtifact: campaignartifact.Artifact{
				CampaignID: "camp-1",
				Path:       "memory.md",
				Content:    "## Loose Ends\n\nProtect the broker.\n",
			},
		}
		session := NewDirectSession(Clients{Artifact: manager}, SessionContext{CampaignID: "camp-1"})

		result, err := session.memorySectionUpdate(context.Background(), []byte(`{"heading":"Loose Ends","content":"Protect the broker."}`))
		if err != nil {
			t.Fatalf("memorySectionUpdate() error = %v", err)
		}

		payload := decodeToolOutput[memorySectionResult](t, result.Output)
		if !payload.Found || payload.Content != "Protect the broker." {
			t.Fatalf("payload = %#v", payload)
		}
		if !strings.Contains(manager.lastContent, "## Loose Ends\n\nProtect the broker.\n") {
			t.Fatalf("upserted content = %q", manager.lastContent)
		}
	})
}

func TestReferenceHandlersCoverConfigurationErrorAndSuccessPaths(t *testing.T) {
	t.Run("search rejects malformed json", func(t *testing.T) {
		session := NewDirectSession(Clients{}, SessionContext{})

		_, err := session.referenceSearch(context.Background(), []byte(`{`))
		if err == nil || !strings.Contains(err.Error(), "unmarshal args") {
			t.Fatalf("referenceSearch() error = %v", err)
		}
	})

	t.Run("search requires configured corpus", func(t *testing.T) {
		session := NewDirectSession(Clients{}, SessionContext{})

		_, err := session.referenceSearch(context.Background(), []byte(`{"query":"fear"}`))
		if err == nil || err.Error() != "reference corpus is not configured" {
			t.Fatalf("referenceSearch() error = %v", err)
		}
	})

	t.Run("search wraps corpus error and copies results", func(t *testing.T) {
		errCorpus := &referenceCorpusTestStub{searchErr: errors.New("index missing")}
		session := NewDirectSession(Clients{Reference: errCorpus}, SessionContext{})

		_, err := session.referenceSearch(context.Background(), []byte(`{"system":"daggerheart","query":"fear"}`))
		if err == nil || !strings.Contains(err.Error(), "system reference search failed: index missing") {
			t.Fatalf("referenceSearch() error = %v", err)
		}
	})

	t.Run("search shapes result payload", func(t *testing.T) {
		corpus := &referenceCorpusTestStub{searchResults: []referencecorpus.SearchResult{{
			System:     "daggerheart",
			DocumentID: "moves/fear",
			Title:      "Fear",
			Kind:       "rule",
			Path:       "moves/fear.md",
			Aliases:    []string{"GM Fear"},
			Snippet:    "Spend Fear when...",
		}}}
		session := NewDirectSession(Clients{Reference: corpus}, SessionContext{})

		result, err := session.referenceSearch(context.Background(), []byte(`{"system":"daggerheart","query":"fear","max_results":3}`))
		if err != nil {
			t.Fatalf("referenceSearch() error = %v", err)
		}

		payload := decodeToolOutput[referenceSearchResult](t, result.Output)
		if len(payload.Results) != 1 {
			t.Fatalf("results length = %d, want 1", len(payload.Results))
		}
		if payload.Results[0].Title != "Fear" || corpus.lastMaxResults != 3 {
			t.Fatalf("payload/result call = %#v / %d", payload.Results[0], corpus.lastMaxResults)
		}
	})

	t.Run("read requires configured corpus", func(t *testing.T) {
		session := NewDirectSession(Clients{}, SessionContext{})

		_, err := session.referenceRead(context.Background(), []byte(`{"document_id":"moves/fear"}`))
		if err == nil || err.Error() != "reference corpus is not configured" {
			t.Fatalf("referenceRead() error = %v", err)
		}
	})

	t.Run("read wraps corpus error", func(t *testing.T) {
		session := NewDirectSession(Clients{
			Reference: &referenceCorpusTestStub{readErr: errors.New("document missing")},
		}, SessionContext{})

		_, err := session.referenceRead(context.Background(), []byte(`{"system":"daggerheart","document_id":"moves/fear"}`))
		if err == nil || !strings.Contains(err.Error(), "system reference read failed: document missing") {
			t.Fatalf("referenceRead() error = %v", err)
		}
	})

	t.Run("read shapes document payload", func(t *testing.T) {
		corpus := &referenceCorpusTestStub{readDocument: referencecorpus.Document{
			System:     "daggerheart",
			DocumentID: "moves/fear",
			Title:      "Fear",
			Kind:       "rule",
			Path:       "moves/fear.md",
			Aliases:    []string{"GM Fear"},
			Content:    "Spend Fear when the fiction turns.",
		}}
		session := NewDirectSession(Clients{Reference: corpus}, SessionContext{})

		result, err := session.referenceRead(context.Background(), []byte(`{"system":"daggerheart","document_id":"moves/fear"}`))
		if err != nil {
			t.Fatalf("referenceRead() error = %v", err)
		}

		payload := decodeToolOutput[referenceDocumentResult](t, result.Output)
		if payload.Content != "Spend Fear when the fiction turns." {
			t.Fatalf("content = %q", payload.Content)
		}
		if corpus.lastDocumentID != "moves/fear" {
			t.Fatalf("document_id passed to corpus = %q", corpus.lastDocumentID)
		}
	})
}

func TestSceneHandlersValidateRequiredFieldsBeforeTransportCalls(t *testing.T) {
	tests := []struct {
		name string
		call func(*DirectSession) error
		want string
	}{
		{
			name: "sceneCreate requires campaign",
			call: func(session *DirectSession) error {
				_, err := session.sceneCreate(context.Background(), []byte(`{"name":"Docks"}`))
				return err
			},
			want: "campaign_id is required",
		},
		{
			name: "sceneCreate requires session",
			call: func(session *DirectSession) error {
				_, err := session.sceneCreate(context.Background(), []byte(`{"campaign_id":"camp-1","name":"Docks"}`))
				return err
			},
			want: "session_id is required",
		},
		{
			name: "sceneUpdate requires scene id",
			call: func(session *DirectSession) error {
				_, err := session.sceneUpdate(context.Background(), []byte(`{"campaign_id":"camp-1"}`))
				return err
			},
			want: "scene_id is required",
		},
		{
			name: "sceneEnd requires scene id",
			call: func(session *DirectSession) error {
				_, err := session.sceneEnd(context.Background(), []byte(`{"campaign_id":"camp-1"}`))
				return err
			},
			want: "scene_id is required",
		},
		{
			name: "sceneAddCharacter requires character id",
			call: func(session *DirectSession) error {
				_, err := session.sceneAddCharacter(context.Background(), []byte(`{"campaign_id":"camp-1","scene_id":"scene-1"}`))
				return err
			},
			want: "character_id is required",
		},
		{
			name: "sceneRemoveCharacter requires character id",
			call: func(session *DirectSession) error {
				_, err := session.sceneRemoveCharacter(context.Background(), []byte(`{"campaign_id":"camp-1","scene_id":"scene-1"}`))
				return err
			},
			want: "character_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := NewDirectSession(Clients{}, SessionContext{})

			err := tt.call(session)
			if err == nil || err.Error() != tt.want {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}
