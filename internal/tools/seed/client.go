package seed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	processClientHealthPath   = "/mcp/health"
	processClientMessagesPath = "/mcp"
	processClientReadyTimeout = 15 * time.Second
	processClientHTTPTimeout  = 30 * time.Second
)

type storedResponse struct {
	value any
	data  []byte
}

// ProcessClient communicates with an MCP child process over the internal HTTP bridge.
type ProcessClient struct {
	baseURL   string
	http      *http.Client
	cmd       *exec.Cmd
	sessionID string

	mu        sync.Mutex
	responses map[string]storedResponse
}

// StartMCPClient launches the MCP process in HTTP mode and returns a raw JSON-RPC client.
func StartMCPClient(ctx context.Context, repoRoot, grpcAddr string) (*ProcessClient, error) {
	httpAddr, err := pickUnusedAddress()
	if err != nil {
		return nil, fmt.Errorf("pick mcp http addr: %w", err)
	}
	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/mcp", "-addr="+grpcAddr, "-http-addr="+httpAddr)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "FRACTURING_SPACE_MCP_PROFILE=harness")
	return startMCPClientCommand(ctx, cmd, httpAddr)
}

// StartMCPClientBinary launches a prebuilt MCP binary in HTTP mode.
func StartMCPClientBinary(ctx context.Context, binaryPath, grpcAddr string) (*ProcessClient, error) {
	httpAddr, err := pickUnusedAddress()
	if err != nil {
		return nil, fmt.Errorf("pick mcp http addr: %w", err)
	}
	cmd := exec.CommandContext(ctx, binaryPath, "-addr="+grpcAddr, "-http-addr="+httpAddr)
	cmd.Env = append(os.Environ(), "FRACTURING_SPACE_MCP_PROFILE=harness")
	return startMCPClientCommand(ctx, cmd, httpAddr)
}

func startMCPClientCommand(ctx context.Context, cmd *exec.Cmd, httpAddr string) (*ProcessClient, error) {
	if cmd == nil {
		return nil, fmt.Errorf("command is required")
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("cookie jar: %w", err)
	}
	httpClient := &http.Client{
		Jar:     jar,
		Timeout: processClientHTTPTimeout,
	}

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	client := &ProcessClient{
		baseURL:   "http://" + httpAddr,
		http:      httpClient,
		cmd:       cmd,
		responses: make(map[string]storedResponse),
	}
	if err := client.waitForHealth(ctx); err != nil {
		client.Close()
		return nil, err
	}
	return client, nil
}

// Close terminates the MCP child process.
func (c *ProcessClient) Close() {
	if c == nil || c.cmd == nil || c.cmd.Process == nil {
		return
	}

	processGroupID := -c.cmd.Process.Pid
	_ = syscall.Kill(processGroupID, syscall.SIGINT)

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- c.cmd.Wait()
	}()

	select {
	case <-waitDone:
	case <-time.After(5 * time.Second):
		_ = syscall.Kill(processGroupID, syscall.SIGKILL)
		<-waitDone
	}
}

// WriteMessage sends one JSON-RPC message to the MCP HTTP bridge.
func (c *ProcessClient) WriteMessage(message any) error {
	if c == nil || c.http == nil {
		return fmt.Errorf("mcp client is not configured")
	}
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+processClientMessagesPath, strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	sessionID := c.currentSessionID()
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("post message: %w", err)
	}
	defer resp.Body.Close()

	if sessionID = strings.TrimSpace(resp.Header.Get("Mcp-Session-Id")); sessionID != "" {
		c.setSessionID(sessionID)
	}

	requestID, hasID := messageID(message)
	if !hasID {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			_, _ = io.Copy(io.Discard, resp.Body)
			return nil
		}
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("post notification: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("post message: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	value, err := DecodeJSONValue(body)
	if err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	c.mu.Lock()
	if c.responses == nil {
		c.responses = make(map[string]storedResponse)
	}
	c.responses[requestID] = storedResponse{value: value, data: body}
	c.mu.Unlock()
	return nil
}

// ReadResponseForID returns the stored response for the matching request id.
func (c *ProcessClient) ReadResponseForID(ctx context.Context, requestID any, timeout time.Duration) (any, []byte, error) {
	if ctx == nil {
		return nil, nil, errors.New("context is nil")
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	key := fmt.Sprint(requestID)
	for {
		c.mu.Lock()
		response, ok := c.responses[key]
		if ok {
			delete(c.responses, key)
		}
		c.mu.Unlock()
		if ok {
			return response.value, response.data, nil
		}

		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func (c *ProcessClient) waitForHealth(ctx context.Context) error {
	if c == nil || c.http == nil {
		return fmt.Errorf("mcp client is not configured")
	}
	deadline := time.Now().Add(processClientReadyTimeout)
	for time.Now().Before(deadline) {
		reqCtx := ctx
		if reqCtx == nil {
			reqCtx = context.Background()
		}
		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, c.baseURL+processClientHealthPath, nil)
		if err != nil {
			return fmt.Errorf("build health request: %w", err)
		}
		resp, err := c.http.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		if ctx != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("mcp http health check did not become ready")
}

func (c *ProcessClient) currentSessionID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sessionID
}

func (c *ProcessClient) setSessionID(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionID = strings.TrimSpace(sessionID)
}

func messageID(message any) (string, bool) {
	msgMap, ok := message.(map[string]any)
	if !ok || msgMap == nil {
		return "", false
	}
	value, ok := msgMap["id"]
	if !ok {
		return "", false
	}
	return fmt.Sprint(value), true
}

func pickUnusedAddress() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer listener.Close()
	return listener.Addr().String(), nil
}
