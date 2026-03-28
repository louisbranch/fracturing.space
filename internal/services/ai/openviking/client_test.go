package openviking

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientSearchIncludesTargetURI(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search/search" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"result": map[string]any{
				"resources": []map[string]any{{
					"uri":          "viking://resources/campaign/story.md",
					"context_type": "resource",
					"is_leaf":      true,
					"abstract":     "Storm warning.",
					"score":        0.91,
					"match_reason": "story",
				}},
			},
		})
	}))
	defer srv.Close()

	client, err := New(Config{BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result, err := client.Search(context.Background(), SearchInput{
		Query:     "storm",
		SessionID: "sess-1",
		TargetURI: "viking://resources/campaign/",
		Limit:     3,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if body["target_uri"] != "viking://resources/campaign/" {
		t.Fatalf("target_uri = %#v", body["target_uri"])
	}
	if len(result.Resources) != 1 || !result.Resources[0].IsLeaf {
		t.Fatalf("result = %#v", result)
	}
}

func TestClientFindUsesTargetedEndpoint(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search/find" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"result": map[string]any{
				"resources": []map[string]any{{
					"uri":          "viking://resources/story/story.md",
					"context_type": "resource",
					"level":        2,
					"abstract":     "Dockmaster Harl knows the debt.",
					"score":        0.91,
				}},
			},
		})
	}))
	defer srv.Close()

	client, err := New(Config{BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	result, err := client.Find(context.Background(), SearchInput{
		Query:     "harbor debt",
		TargetURI: "viking://resources/story/",
		Limit:     2,
	})
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if body["target_uri"] != "viking://resources/story/" {
		t.Fatalf("target_uri = %#v", body["target_uri"])
	}
	if len(result.Resources) != 1 || result.Resources[0].Level != 2 {
		t.Fatalf("result = %#v", result)
	}
}

func TestClientOverviewAndReadUseContentEndpoints(t *testing.T) {
	requests := []string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path+"?"+r.URL.RawQuery)
		var result any
		switch r.URL.Path {
		case "/api/v1/content/overview":
			result = "## docs\nOverview"
		case "/api/v1/content/read":
			result = "# Full content"
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"result": result,
		})
	}))
	defer srv.Close()

	client, err := New(Config{BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	overview, err := client.Overview(context.Background(), "viking://resources/docs/")
	if err != nil {
		t.Fatalf("Overview() error = %v", err)
	}
	read, err := client.Read(context.Background(), "viking://resources/docs/api.md")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if overview == "" || read == "" {
		t.Fatalf("overview=%q read=%q", overview, read)
	}
	if len(requests) != 2 {
		t.Fatalf("requests = %#v", requests)
	}
}

func TestClientUsedPostsContexts(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/sessions/sess-1/used" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"result": map[string]any{},
		})
	}))
	defer srv.Close()

	client, err := New(Config{BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := client.Used(context.Background(), "sess-1", UsedInput{
		Contexts: []string{"viking://resources/story.md"},
	}); err != nil {
		t.Fatalf("Used() error = %v", err)
	}
	contexts, _ := body["contexts"].([]any)
	if len(contexts) != 1 {
		t.Fatalf("contexts = %#v", body["contexts"])
	}
}

func TestClientAddResourceUsesToAndReportsNestedErrors(t *testing.T) {
	requestBodies := []map[string]any{}
	successSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/resources":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			requestBodies = append(requestBodies, body)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "ok",
				"result": map[string]any{
					"status":      "success",
					"source_path": "/app/data/story.md",
					"root_uri":    "viking://resources/fracturing-space/story.md",
				},
			})
		case "/api/v1/resources-error":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "ok",
				"result": map[string]any{
					"status": "error",
					"errors": []string{"parse failed"},
				},
			})
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer successSrv.Close()

	client, err := New(Config{BaseURL: successSrv.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	result, err := client.AddResource(context.Background(), AddResourceInput{
		Path: "/app/data/story.md",
		To:   "viking://resources/fracturing-space/story.md",
		Wait: true,
	})
	if err != nil {
		t.Fatalf("AddResource() error = %v", err)
	}
	if len(requestBodies) != 1 || requestBodies[0]["to"] != "viking://resources/fracturing-space/story.md" {
		t.Fatalf("request bodies = %#v", requestBodies)
	}
	if result.RootURI != "viking://resources/fracturing-space/story.md" {
		t.Fatalf("result = %#v", result)
	}

	errorSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/resources" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"result": map[string]any{
				"status": "error",
				"errors": []string{"parse failed"},
			},
		})
	}))
	defer errorSrv.Close()

	errorClient, err := New(Config{BaseURL: errorSrv.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_, err = errorClient.AddResource(context.Background(), AddResourceInput{
		Path: "/app/data/story.md",
		To:   "viking://resources/fracturing-space/story.md",
	})
	if err == nil || !strings.Contains(err.Error(), "parse failed") {
		t.Fatalf("AddResource() error = %v, want nested parse failure", err)
	}
}

func TestClientCommitAndGetTask(t *testing.T) {
	requests := []string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path+"?"+r.URL.RawQuery)
		switch r.URL.Path {
		case "/api/v1/sessions/sess-1/commit":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "ok",
				"result": map[string]any{
					"session_id":         "sess-1",
					"status":             "committed",
					"memories_extracted": 1,
				},
			})
		case "/api/v1/tasks/task-1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "ok",
				"result": map[string]any{
					"task_id": "task-1",
					"status":  "completed",
				},
			})
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()

	client, err := New(Config{BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	commit, err := client.Commit(context.Background(), "sess-1", true)
	if err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if commit.Status != "committed" || commit.MemoriesExtracted != 1 {
		t.Fatalf("commit = %#v", commit)
	}
	if len(requests) == 0 || requests[0] != "/api/v1/sessions/sess-1/commit?wait=true" {
		t.Fatalf("requests = %#v", requests)
	}
	task, err := client.GetTask(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if task.Status != "completed" {
		t.Fatalf("task = %#v", task)
	}
}

func TestClientGetSessionUsesNamedSessionEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/sessions/sess-1" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"result": map[string]any{
				"session_id":    "sess-1",
				"message_count": 2,
				"user": map[string]any{
					"account_id": "default",
					"user_id":    "default",
					"agent_id":   "default",
				},
			},
		})
	}))
	defer srv.Close()

	client, err := New(Config{BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	session, err := client.GetSession(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if session.User.UserID != "default" || session.MessageCount != 2 {
		t.Fatalf("session = %#v", session)
	}
}
