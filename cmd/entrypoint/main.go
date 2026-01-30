// Package main runs the gRPC server and MCP bridge in one container.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

// defaultGrpcAddr is the internal gRPC address used by the MCP bridge.
const defaultGrpcAddr = "127.0.0.1:8080"

// defaultMcpHTTPAddr is the MCP HTTP bind address for container use.
const defaultMcpHTTPAddr = "0.0.0.0:8081"

// shutdownTimeout is the grace period before forcing child exit.
const shutdownTimeout = 10 * time.Second

// childProcess describes a managed child command.
type childProcess struct {
	name string
	cmd  *exec.Cmd
}

// processExit reports a child process exit result.
type processExit struct {
	name string
	err  error
}

// main starts the gRPC server and MCP HTTP bridge, then supervises them.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverCmd := exec.Command("/app/server", "-port=8080")
	server, err := startChild("grpc-server", serverCmd)
	if err != nil {
		log.Fatalf("failed to start gRPC server: %v", err)
	}

	grpcAddr := getenvDefault("DUALITY_GRPC_ADDR", defaultGrpcAddr)
	mcpHTTPAddr := getenvDefault("DUALITY_MCP_HTTP_ADDR", defaultMcpHTTPAddr)
	mcpCmd := exec.Command(
		"/app/mcp",
		"-transport=http",
		"-http-addr="+mcpHTTPAddr,
		"-addr="+grpcAddr,
	)
	mcp, err := startChild("mcp", mcpCmd)
	if err != nil {
		terminateChildren([]*childProcess{server})
		log.Fatalf("failed to start MCP server: %v", err)
	}

	children := []*childProcess{server, mcp}
	exitCh := make(chan processExit, len(children))
	go waitChild(server, exitCh)
	go waitChild(mcp, exitCh)

	select {
	case <-ctx.Done():
		log.Printf("shutdown signal received")
		terminateChildren(children)
		waitForChildren(exitCh, len(children), shutdownTimeout, children)
		return
	case exit := <-exitCh:
		log.Printf("%s exited: %v", exit.name, exit.err)
		terminateChildren(children)
		waitForChildren(exitCh, len(children)-1, shutdownTimeout, children)
		os.Exit(exitCode(exit.err))
	}
}

// startChild starts a child process with inherited stdio streams.
func startChild(name string, cmd *exec.Cmd) (*childProcess, error) {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("start %s: %w", name, err)
	}

	return &childProcess{name: name, cmd: cmd}, nil
}

// waitChild waits for a child process and reports its exit.
func waitChild(child *childProcess, exitCh chan<- processExit) {
	err := child.cmd.Wait()
	exitCh <- processExit{name: child.name, err: err}
}

// terminateChildren sends SIGTERM to all child processes.
func terminateChildren(children []*childProcess) {
	for _, child := range children {
		if child == nil || child.cmd == nil || child.cmd.Process == nil {
			continue
		}
		_ = child.cmd.Process.Signal(syscall.SIGTERM)
	}
}

// waitForChildren waits for the remaining exits or forces shutdown.
func waitForChildren(exitCh <-chan processExit, remaining int, timeout time.Duration, children []*childProcess) {
	if remaining <= 0 {
		return
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for remaining > 0 {
		select {
		case <-exitCh:
			remaining--
		case <-timer.C:
			forceKill(children)
			return
		}
	}
}

// forceKill sends SIGKILL to any child still running.
func forceKill(children []*childProcess) {
	for _, child := range children {
		if child == nil || child.cmd == nil || child.cmd.Process == nil {
			continue
		}
		if child.cmd.ProcessState != nil {
			continue
		}
		_ = child.cmd.Process.Kill()
	}
}

// exitCode derives a process exit code from a wait error.
func exitCode(err error) int {
	if err == nil {
		return 0
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}

	return 1
}

// getenvDefault returns the env value or a fallback when unset.
func getenvDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
