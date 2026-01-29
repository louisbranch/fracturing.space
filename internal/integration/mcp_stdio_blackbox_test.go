//go:build integration

package integration

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"
)

// TestMCPStdioBlackbox validates the stdio MCP surface using the shared fixture.
func TestMCPStdioBlackbox(t *testing.T) {
	grpcAddr, stopGRPC := startGRPCServer(t)
	defer stopGRPC()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd, client, err := startMCPStdioServer(ctx, t, grpcAddr)
	if err != nil {
		t.Fatalf("start MCP stdio server: %v", err)
	}
	defer stopMCPProcess(t, cancel, cmd)

	fixture := loadBlackboxFixture(t, filepath.Join(repoRoot(t), blackboxFixturePath))
	captures := make(map[string]string)
	for _, step := range fixture.Steps {
		executeStdioBlackboxStep(t, client, step, captures)
	}
}

type stdioClient struct {
	reader *bufio.Reader
	writer io.Writer
	mu     sync.Mutex
}

func startMCPStdioServer(ctx context.Context, t *testing.T, grpcAddr string) (*exec.Cmd, *stdioClient, error) {
	t.Helper()

	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/mcp", "-addr="+grpcAddr)
	cmd.Dir = repoRoot(t)
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	client := &stdioClient{
		reader: bufio.NewReader(stdout),
		writer: stdin,
	}
	return cmd, client, nil
}

func executeStdioBlackboxStep(t *testing.T, client *stdioClient, step blackboxStep, captures map[string]string) {
	t.Helper()

	request := renderPlaceholders(step.Request, captures)
	requestMap, ok := request.(map[string]any)
	if !ok {
		t.Fatalf("%s request is not an object", step.Name)
	}
	requestID, hasID := requestMap["id"]

	if err := client.WriteMessage(request); err != nil {
		t.Fatalf("write request %s: %v", step.Name, err)
	}
	if !hasID {
		return
	}

	responseAny, responseBytes, err := client.ReadResponseForID(requestID, 5*time.Second)
	if err != nil {
		t.Fatalf("read response %s: %v", step.Name, err)
	}
	if responseAny == nil {
		t.Fatalf("%s response is nil", step.Name)
	}

	for path, expected := range step.ExpectPaths {
		actual, err := lookupJSONPath(responseAny, path)
		if err != nil {
			errorDetails := formatJSONRPCError(responseAny)
			if errorDetails != "" {
				t.Fatalf("%s lookup %s: %v (error=%s)", step.Name, path, err, errorDetails)
			}
			t.Fatalf("%s lookup %s: %v (response=%s)", step.Name, path, err, string(responseBytes))
		}
		resolvedExpected := renderPlaceholders(expected, captures)
		if !valuesEqual(actual, resolvedExpected) {
			t.Fatalf("%s expected %s = %v, got %v (response=%s)", step.Name, path, resolvedExpected, actual, string(responseBytes))
		}
	}

	for key, paths := range step.Captures {
		value, err := captureFromPaths(responseAny, paths)
		if err != nil {
			t.Fatalf("%s capture %s: %v (response=%s)", step.Name, key, err, string(responseBytes))
		}
		if value == "" {
			t.Fatalf("%s capture %s: empty value", step.Name, key)
		}
		captures[key] = value
	}
}

func (client *stdioClient) WriteMessage(message any) error {
	client.mu.Lock()
	defer client.mu.Unlock()

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	data = append(data, '\n')
	if _, err := client.writer.Write(data); err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	return nil
}

func (client *stdioClient) ReadResponseForID(requestID any, timeout time.Duration) (any, []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	requestIDString := fmt.Sprint(requestID)
	for {
		responseAny, responseBytes, err := client.readMessage(ctx)
		if err != nil {
			return nil, nil, err
		}
		responseMap, ok := responseAny.(map[string]any)
		if !ok {
			continue
		}
		responseID, hasID := responseMap["id"]
		if !hasID {
			continue
		}
		if fmt.Sprint(responseID) == requestIDString {
			return responseAny, responseBytes, nil
		}
	}
}

func (client *stdioClient) readMessage(ctx context.Context) (any, []byte, error) {
	type result struct {
		value any
		data  []byte
		err   error
	}
	resultChan := make(chan result, 1)

	go func() {
		data, err := readStdioLine(client.reader)
		if err != nil {
			resultChan <- result{err: err}
			return
		}
		value, err := decodeJSONValueInternalBytes(data)
		if err != nil {
			resultChan <- result{err: err, data: data}
			return
		}
		resultChan <- result{value: value, data: data}
	}()

	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case res := <-resultChan:
		return res.value, res.data, res.err
	}
}

func readStdioLine(reader *bufio.Reader) ([]byte, error) {
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return nil, fmt.Errorf("read line: %w", err)
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		return line, nil
	}
}

func decodeJSONValueInternalBytes(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("decode JSON message: %w (payload=%s)", err, string(data))
	}
	return value, nil
}
