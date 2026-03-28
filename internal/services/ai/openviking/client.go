package openviking

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Config declares how the OpenViking HTTP client connects to the remote
// service.
type Config struct {
	BaseURL     string
	APIKey      string
	HTTPClient  *http.Client
	Timeout     time.Duration
	ResourceTTL time.Duration
}

// Client talks to one OpenViking HTTP endpoint.
type Client struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// AddResourceInput declares one resource mirroring request.
type AddResourceInput struct {
	Path    string
	To      string
	Parent  string
	Reason  string
	Wait    bool
	Timeout time.Duration
}

// AddResourceResult describes one completed resource ingest request.
type AddResourceResult struct {
	Status     string   `json:"status"`
	Errors     []string `json:"errors"`
	SourcePath string   `json:"source_path"`
	RootURI    string   `json:"root_uri"`
	TempURI    string   `json:"temp_uri"`
}

// SearchInput declares one OpenViking search call.
type SearchInput struct {
	Query     string
	SessionID string
	TargetURI string
	Limit     int
}

// SearchResult carries the cross-type contexts returned by OpenViking.
type SearchResult struct {
	Memories  []MatchedContext `json:"memories"`
	Resources []MatchedContext `json:"resources"`
	Skills    []MatchedContext `json:"skills"`
}

// MatchedContext mirrors the OpenViking search response shape used by this
// pilot.
type MatchedContext struct {
	URI         string  `json:"uri"`
	ContextType string  `json:"context_type"`
	IsLeaf      bool    `json:"is_leaf"`
	Level       int     `json:"level"`
	Abstract    string  `json:"abstract"`
	Score       float64 `json:"score"`
	MatchReason string  `json:"match_reason"`
}

// AddMessageInput declares one session message write.
type AddMessageInput struct {
	Role    string        `json:"role"`
	Content string        `json:"content,omitempty"`
	Parts   []MessagePart `json:"parts,omitempty"`
}

// UsedInput declares one session usage record.
type UsedInput struct {
	Contexts []string       `json:"contexts,omitempty"`
	Skill    *SkillUseInput `json:"skill,omitempty"`
}

// SkillUseInput mirrors the OpenViking skill usage shape.
type SkillUseInput struct {
	URI     string         `json:"uri"`
	Input   map[string]any `json:"input,omitempty"`
	Output  string         `json:"output,omitempty"`
	Success bool           `json:"success"`
}

// MessagePart mirrors the HTTP "parts mode" message shape used by session
// sync.
type MessagePart struct {
	Type        string         `json:"type"`
	Text        string         `json:"text,omitempty"`
	URI         string         `json:"uri,omitempty"`
	ContextType string         `json:"context_type,omitempty"`
	Abstract    string         `json:"abstract,omitempty"`
	ToolID      string         `json:"tool_id,omitempty"`
	ToolName    string         `json:"tool_name,omitempty"`
	ToolInput   map[string]any `json:"tool_input,omitempty"`
	ToolOutput  string         `json:"tool_output,omitempty"`
	ToolStatus  string         `json:"tool_status,omitempty"`
}

type responseEnvelope[T any] struct {
	Status string `json:"status"`
	Result T      `json:"result"`
	Error  struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// CommitResult describes one accepted session commit request.
type CommitResult struct {
	SessionID         string `json:"session_id"`
	Status            string `json:"status"`
	TaskID            string `json:"task_id"`
	MemoriesExtracted int    `json:"memories_extracted"`
}

// TaskStatus describes one OpenViking background task.
type TaskStatus struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

// SessionUser describes the resolved OpenViking user/agent identity for one
// session.
type SessionUser struct {
	AccountID string `json:"account_id"`
	UserID    string `json:"user_id"`
	AgentID   string `json:"agent_id"`
}

// SessionInfo describes one named OpenViking session.
type SessionInfo struct {
	SessionID    string      `json:"session_id"`
	User         SessionUser `json:"user"`
	MessageCount int         `json:"message_count"`
}

// FilesystemEntry describes one OpenViking filesystem entry.
type FilesystemEntry struct {
	Name    string `json:"name"`
	URI     string `json:"uri"`
	RelPath string `json:"rel_path"`
	IsDir   bool   `json:"isDir"`
}

// New builds an OpenViking HTTP client from explicit configuration.
func New(cfg Config) (*Client, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("openviking base URL is required")
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = 15 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
	}
	return &Client{
		baseURL: baseURL,
		apiKey:  strings.TrimSpace(cfg.APIKey),
		client:  httpClient,
	}, nil
}

// GetSession reads one named OpenViking session.
func (c *Client) GetSession(ctx context.Context, sessionID string) (SessionInfo, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return SessionInfo{}, fmt.Errorf("session ID is required")
	}
	var result SessionInfo
	if err := c.do(ctx, http.MethodGet, "/api/v1/sessions/"+url.PathEscape(sessionID), nil, &result); err != nil {
		return SessionInfo{}, err
	}
	return result, nil
}

// AddMessage appends one message to a session.
func (c *Client) AddMessage(ctx context.Context, sessionID string, input AddMessageInput) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session ID is required")
	}
	if strings.TrimSpace(input.Role) == "" {
		return fmt.Errorf("message role is required")
	}
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/sessions/%s/messages", url.PathEscape(sessionID)), input, nil)
}

// Commit archives one session and optionally waits for memory extraction.
func (c *Client) Commit(ctx context.Context, sessionID string, wait bool) (CommitResult, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return CommitResult{}, fmt.Errorf("session ID is required")
	}
	var result CommitResult
	path := fmt.Sprintf("/api/v1/sessions/%s/commit?wait=%t", url.PathEscape(sessionID), wait)
	if err := c.do(ctx, http.MethodPost, path, map[string]any{}, &result); err != nil {
		return CommitResult{}, err
	}
	return result, nil
}

// Used records the contexts and/or skills actually used during a session turn.
func (c *Client) Used(ctx context.Context, sessionID string, input UsedInput) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session ID is required")
	}
	body := map[string]any{}
	contexts := make([]string, 0, len(input.Contexts))
	for _, item := range input.Contexts {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		contexts = append(contexts, item)
	}
	if len(contexts) > 0 {
		body["contexts"] = contexts
	}
	if input.Skill != nil && strings.TrimSpace(input.Skill.URI) != "" {
		body["skill"] = input.Skill
	}
	if len(body) == 0 {
		return nil
	}
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/sessions/%s/used", url.PathEscape(sessionID)), body, nil)
}

// AddResource mirrors one local file into one OpenViking resource URI.
func (c *Client) AddResource(ctx context.Context, input AddResourceInput) (AddResourceResult, error) {
	if strings.TrimSpace(input.Path) == "" {
		return AddResourceResult{}, fmt.Errorf("resource path is required")
	}
	to := strings.TrimSpace(input.To)
	parent := strings.TrimSpace(input.Parent)
	if to != "" && parent != "" {
		return AddResourceResult{}, fmt.Errorf("resource to and parent are mutually exclusive")
	}
	body := map[string]any{
		"path": strings.TrimSpace(input.Path),
	}
	if to != "" {
		body["to"] = to
	}
	if parent != "" {
		body["parent"] = parent
	}
	if reason := strings.TrimSpace(input.Reason); reason != "" {
		body["reason"] = reason
	}
	if input.Wait {
		body["wait"] = true
	}
	if input.Timeout > 0 {
		body["timeout"] = input.Timeout.Seconds()
	}
	var result AddResourceResult
	if err := c.do(ctx, http.MethodPost, "/api/v1/resources", body, &result); err != nil {
		return AddResourceResult{}, err
	}
	if strings.EqualFold(strings.TrimSpace(result.Status), "error") {
		return result, fmt.Errorf("openviking resource ingest failed: %s", strings.Join(result.Errors, "; "))
	}
	return result, nil
}

// Find executes one targeted retrieval request without session context.
func (c *Client) Find(ctx context.Context, input SearchInput) (SearchResult, error) {
	if strings.TrimSpace(input.Query) == "" {
		return SearchResult{}, fmt.Errorf("search query is required")
	}
	body := map[string]any{
		"query": strings.TrimSpace(input.Query),
	}
	if input.Limit > 0 {
		body["limit"] = input.Limit
	}
	if targetURI := strings.TrimSpace(input.TargetURI); targetURI != "" {
		body["target_uri"] = targetURI
	}
	var result SearchResult
	if err := c.do(ctx, http.MethodPost, "/api/v1/search/find", body, &result); err != nil {
		return SearchResult{}, err
	}
	return result, nil
}

// Search executes one context-aware search request.
func (c *Client) Search(ctx context.Context, input SearchInput) (SearchResult, error) {
	if strings.TrimSpace(input.Query) == "" {
		return SearchResult{}, fmt.Errorf("search query is required")
	}
	body := map[string]any{
		"query": strings.TrimSpace(input.Query),
	}
	if input.Limit > 0 {
		body["limit"] = input.Limit
	}
	if sessionID := strings.TrimSpace(input.SessionID); sessionID != "" {
		body["session_id"] = sessionID
	}
	if targetURI := strings.TrimSpace(input.TargetURI); targetURI != "" {
		body["target_uri"] = targetURI
	}
	var result SearchResult
	if err := c.do(ctx, http.MethodPost, "/api/v1/search/search", body, &result); err != nil {
		return SearchResult{}, err
	}
	return result, nil
}

// Overview reads one directory-level L1 overview from OpenViking.
func (c *Client) Overview(ctx context.Context, uri string) (string, error) {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return "", fmt.Errorf("uri is required")
	}
	var result string
	if err := c.do(ctx, http.MethodGet, "/api/v1/content/overview?uri="+url.QueryEscape(uri), nil, &result); err != nil {
		return "", err
	}
	return result, nil
}

// Read reads one L2 content node from OpenViking.
func (c *Client) Read(ctx context.Context, uri string) (string, error) {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return "", fmt.Errorf("uri is required")
	}
	var result string
	if err := c.do(ctx, http.MethodGet, "/api/v1/content/read?uri="+url.QueryEscape(uri), nil, &result); err != nil {
		return "", err
	}
	return result, nil
}

// Tree lists the recursive file tree rooted at one OpenViking URI.
func (c *Client) Tree(ctx context.Context, uri string) ([]FilesystemEntry, error) {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return nil, fmt.Errorf("uri is required")
	}
	var result []FilesystemEntry
	if err := c.do(ctx, http.MethodGet, "/api/v1/fs/tree?uri="+url.QueryEscape(uri), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetTask reads one background task status from OpenViking.
func (c *Client) GetTask(ctx context.Context, taskID string) (TaskStatus, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return TaskStatus{}, fmt.Errorf("task ID is required")
	}
	var result TaskStatus
	if err := c.do(ctx, http.MethodGet, "/api/v1/tasks/"+url.PathEscape(taskID), nil, &result); err != nil {
		return TaskStatus{}, err
	}
	return result, nil
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var reader *bytes.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal openviking request: %w", err)
		}
		reader = bytes.NewReader(payload)
	} else {
		reader = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("build openviking request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("openviking request failed: %w", err)
	}
	defer res.Body.Close()

	var envelope responseEnvelope[json.RawMessage]
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("decode openviking response: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 || !strings.EqualFold(envelope.Status, "ok") {
		message := strings.TrimSpace(envelope.Error.Message)
		if message == "" {
			message = strings.TrimSpace(string(envelope.Result))
		}
		if message == "" {
			message = res.Status
		}
		return fmt.Errorf("openviking %s %s: %s", method, path, message)
	}
	if out == nil || len(envelope.Result) == 0 || string(envelope.Result) == "null" {
		return nil
	}
	if err := json.Unmarshal(envelope.Result, out); err != nil {
		return fmt.Errorf("decode openviking result: %w", err)
	}
	return nil
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(text, "not found")
}
