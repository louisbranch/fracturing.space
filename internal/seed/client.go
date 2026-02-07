package seed

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// StdioClient communicates with an MCP process over stdio.
type StdioClient struct {
	reader *bufio.Reader
	writer io.Writer
	mu     sync.Mutex
	cmd    *exec.Cmd
}

// StartMCPClient launches the MCP stdio process and returns a client.
func StartMCPClient(ctx context.Context, repoRoot, grpcAddr string) (*StdioClient, error) {
	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/mcp", "-addr="+grpcAddr)
	cmd.Dir = repoRoot
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	client := &StdioClient{
		reader: bufio.NewReader(stdout),
		writer: stdin,
		cmd:    cmd,
	}
	return client, nil
}

// Close terminates the MCP process.
func (c *StdioClient) Close() {
	if c.cmd == nil || c.cmd.Process == nil {
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

// WriteMessage sends a JSON message to the MCP process.
func (c *StdioClient) WriteMessage(message any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	data = append(data, '\n')
	if _, err := c.writer.Write(data); err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	return nil
}

// ReadResponseForID waits for a response matching the request ID.
func (c *StdioClient) ReadResponseForID(requestID any, timeout time.Duration) (any, []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	requestIDString := fmt.Sprint(requestID)
	for {
		responseAny, responseBytes, err := c.readMessage(ctx)
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

func (c *StdioClient) readMessage(ctx context.Context) (any, []byte, error) {
	type result struct {
		value any
		data  []byte
		err   error
	}
	resultChan := make(chan result, 1)

	go func() {
		data, err := readStdioLine(c.reader)
		if err != nil {
			resultChan <- result{err: err}
			return
		}
		value, err := DecodeJSONValue(data)
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
