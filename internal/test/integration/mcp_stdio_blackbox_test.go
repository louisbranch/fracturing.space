//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/tools/seed"
)

var stdioBlackboxFixtureFiles = []string{
	"blackbox_00_context_flow.json",
	"blackbox_campaign_flow.json",
}

// TestMCPStdioBlackbox validates a small stdio fixture set so transport-process
// boundaries stay covered without replaying the full blackbox matrix.
func TestMCPStdioBlackbox(t *testing.T) {
	fixture := newSuiteFixture(t)
	binaryPath := mcpBinaryForTests(t)

	fixtures := loadBlackboxFixtureFiles(t, stdioBlackboxFixtureFiles)
	for _, blackboxFixture := range fixtures {
		func(blackboxFixture seed.BlackboxFixture) {
			userID := fixture.newUserID(t, uniqueTestUsername(t, "blackbox-creator", blackboxFixture.Name))
			ctx, cancel := context.WithCancel(context.Background())
			client, err := seed.StartMCPClientBinary(ctx, binaryPath, fixture.grpcAddr)
			if err != nil {
				cancel()
				t.Fatalf("start MCP stdio server: %v", err)
			}
			defer cancel()
			defer client.Close()

			captures := make(map[string]string)
			for _, step := range blackboxFixture.Steps {
				executeStdioBlackboxStep(t, ctx, client, step, captures, userID)
			}
		}(blackboxFixture)
	}
}

func executeStdioBlackboxStep(t *testing.T, ctx context.Context, client *seed.StdioClient, step seed.BlackboxStep, captures map[string]string, userID string) {
	t.Helper()

	request, err := seed.RenderPlaceholders(step.Request, captures)
	if err != nil {
		t.Fatalf("%s render placeholders: %v", step.Name, err)
	}
	requestMap, ok := request.(map[string]any)
	if !ok {
		t.Fatalf("%s request is not an object", step.Name)
	}
	if userID != "" {
		injectCampaignCreatorUserID(requestMap, userID)
	}
	invoke := func(reqMap map[string]any) (map[string]any, []byte, error) {
		requestID, hasID := reqMap["id"]
		if err := client.WriteMessage(reqMap); err != nil {
			return nil, nil, err
		}
		if !hasID {
			return map[string]any{}, nil, nil
		}
		responseAny, responseBytes, err := client.ReadResponseForID(ctx, requestID, 30*time.Second)
		if err != nil {
			return nil, responseBytes, err
		}
		responseMap, _ := responseAny.(map[string]any)
		if responseMap == nil {
			return nil, responseBytes, fmt.Errorf("response is not an object")
		}
		return responseMap, responseBytes, nil
	}

	maybeEnsureSessionStartReadinessForBlackbox(t, step.Name, requestMap, captures, invoke)

	responseAny, responseBytes, err := invoke(requestMap)
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
