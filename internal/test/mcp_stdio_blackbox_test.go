//go:build integration

package integration

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/seed"
)

// TestMCPStdioBlackbox validates the stdio MCP surface using the shared fixture.
func TestMCPStdioBlackbox(t *testing.T) {
	grpcAddr, stopGRPC := startGRPCServer(t)
	defer stopGRPC()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := seed.StartMCPClient(ctx, repoRoot(t), grpcAddr)
	if err != nil {
		t.Fatalf("start MCP stdio server: %v", err)
	}
	defer client.Close()

	fixtures := loadBlackboxFixtures(t, filepath.Join(repoRoot(t), blackboxFixtureGlob))
	for _, fixture := range fixtures {
		captures := make(map[string]string)
		for _, step := range fixture.Steps {
			executeStdioBlackboxStep(t, client, step, captures)
		}
	}
}

func executeStdioBlackboxStep(t *testing.T, client *seed.StdioClient, step seed.BlackboxStep, captures map[string]string) {
	t.Helper()

	request, err := seed.RenderPlaceholders(step.Request, captures)
	if err != nil {
		t.Fatalf("%s render placeholders: %v", step.Name, err)
	}
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
		actual, err := seed.LookupJSONPath(responseAny, path)
		if err != nil {
			errorDetails := seed.FormatJSONRPCError(responseAny)
			if errorDetails != "" {
				t.Fatalf("%s lookup %s: %v (error=%s)", step.Name, path, err, errorDetails)
			}
			t.Fatalf("%s lookup %s: %v (response=%s)", step.Name, path, err, string(responseBytes))
		}
		resolvedExpected, err := seed.RenderPlaceholders(expected, captures)
		if err != nil {
			t.Fatalf("%s render expected: %v", step.Name, err)
		}
		if !seed.ValuesEqual(actual, resolvedExpected) {
			t.Fatalf("%s expected %s = %v, got %v (response=%s)", step.Name, path, resolvedExpected, actual, string(responseBytes))
		}
	}

	for key, paths := range step.Captures {
		value, err := seed.CaptureFromPaths(responseAny, paths)
		if err != nil {
			hints := seed.CaptureHints(responseAny)
			if len(hints) > 0 {
				t.Fatalf("%s capture %s: %v (hints=%s, response=%s)", step.Name, key, err, seed.FormatCaptureHints(hints), string(responseBytes))
			}
			t.Fatalf("%s capture %s: %v (response=%s)", step.Name, key, err, string(responseBytes))
		}
		if value == "" {
			t.Fatalf("%s capture %s: empty value", step.Name, key)
		}
		captures[key] = value
	}
}
