package game

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestDirectAppendEventUsageIsRestrictedToMaintenancePaths(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	grpcRoot := filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc")
	allowed := map[string]struct{}{
		"internal/services/game/api/grpc/game/domain_adapter.go":        {},
		"internal/services/game/api/grpc/game/event_application.go":     {},
		"internal/services/game/api/grpc/game/fork_application.go":      {},
		"internal/services/game/api/grpc/game/fork_application_fork.go": {},
	}

	var violations []string
	walkErr := filepath.WalkDir(grpcRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)
		if _, ok := allowed[relPath]; ok {
			return nil
		}
		lines, err := appendEventCallLines(path)
		if err != nil {
			return err
		}
		for _, line := range lines {
			violations = append(violations, fmt.Sprintf("%s:%d", relPath, line))
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("scan grpc files: %v", walkErr)
	}

	sort.Strings(violations)
	if len(violations) == 0 {
		return
	}
	t.Fatalf("direct AppendEvent usage outside maintenance/import paths:\n%s", strings.Join(violations, "\n"))
}

func TestDirectDomainExecuteUsageIsForbidden(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	gameRoot := filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game")

	var violations []string
	walkErr := filepath.WalkDir(gameRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		lines, err := domainExecuteCallLines(path)
		if err != nil {
			return err
		}
		for _, line := range lines {
			violations = append(violations, fmt.Sprintf("%s:%d", filepath.ToSlash(relPath), line))
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("scan game grpc files: %v", walkErr)
	}

	sort.Strings(violations)
	if len(violations) == 0 {
		return
	}
	t.Fatalf("direct Domain.Execute usage found:\n%s", strings.Join(violations, "\n"))
}

func TestSessionGateCommandExecutorUsageIsRestrictedToGateApplications(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	gameRoot := filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game")
	allowed := map[string]struct{}{
		"internal/services/game/api/grpc/game/session_gate_application.go":          {},
		"internal/services/game/api/grpc/game/communication_application_control.go": {},
	}

	var violations []string
	walkErr := filepath.WalkDir(gameRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)
		if _, ok := allowed[relPath]; ok {
			return nil
		}
		lines, err := selectorCallLines(path, []string{"a", "gateCommands", "Execute"})
		if err != nil {
			return err
		}
		for _, line := range lines {
			violations = append(violations, fmt.Sprintf("%s:%d", relPath, line))
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("scan game grpc files: %v", walkErr)
	}

	sort.Strings(violations)
	if len(violations) == 0 {
		return
	}
	t.Fatalf("session gate command executor usage outside gate applications:\n%s", strings.Join(violations, "\n"))
}

func TestSessionApplicationWriteFlowsUseSessionCommandExecutor(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	applicationFiles := []string{
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "session_lifecycle_application.go"),
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "session_spotlight_application.go"),
	}

	var violations []string
	for _, path := range applicationFiles {
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			t.Fatalf("relative path %s: %v", path, err)
		}
		relPath = filepath.ToSlash(relPath)

		callTargets := []struct {
			label string
			lines func(string) ([]int, error)
		}{
			{
				label: "executeAndApplyDomainCommand",
				lines: func(path string) ([]int, error) { return namedCallLines(path, "executeAndApplyDomainCommand") },
			},
			{
				label: "commandbuild.Core",
				lines: func(path string) ([]int, error) { return selectorCallLines(path, []string{"commandbuild", "Core"}) },
			},
		}
		for _, target := range callTargets {
			lines, err := target.lines(path)
			if err != nil {
				t.Fatalf("scan %s for %s: %v", path, target.label, err)
			}
			for _, line := range lines {
				violations = append(violations, fmt.Sprintf("%s:%d uses %s", relPath, line, target.label))
			}
		}
	}

	sort.Strings(violations)
	if len(violations) == 0 {
		return
	}
	t.Fatalf("session application write flows bypass the session command executor:\n%s", strings.Join(violations, "\n"))
}

func TestParticipantAndCharacterTransportHelpersDoNotLiveInRootPackage(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	legacyFiles := []string{
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "participant_mappers.go"),
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "character_mappers.go"),
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "character_service_helpers.go"),
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "daggerheart_conditions.go"),
	}

	var violations []string
	for _, path := range legacyFiles {
		if _, err := os.Stat(path); err == nil {
			relPath, relErr := filepath.Rel(repoRoot, path)
			if relErr != nil {
				t.Fatalf("relative path %s: %v", path, relErr)
			}
			violations = append(violations, filepath.ToSlash(relPath))
			continue
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", path, err)
		}
	}

	sort.Strings(violations)
	if len(violations) == 0 {
		return
	}
	t.Fatalf("legacy transport helpers still live in the root game package:\n%s", strings.Join(violations, "\n"))
}

func TestCommunicationServiceHandlersDoNotAccessStoresDirectly(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	serviceFiles := []string{
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "communication_service.go"),
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "communication_service_control.go"),
	}

	var violations []string
	for _, path := range serviceFiles {
		lines, err := selectorUsageLines(path, []string{"s", "stores"})
		if err != nil {
			t.Fatalf("scan %s: %v", path, err)
		}
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			t.Fatalf("relative path %s: %v", path, err)
		}
		for _, line := range lines {
			violations = append(violations, fmt.Sprintf("%s:%d", filepath.ToSlash(relPath), line))
		}
	}

	sort.Strings(violations)
	if len(violations) == 0 {
		return
	}
	t.Fatalf("communication service handlers access stores directly:\n%s", strings.Join(violations, "\n"))
}

func TestCommunicationApplicationUsesSessionReadBoundary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	paths := []string{
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "communication_application_context.go"),
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "communication_application_control.go"),
	}

	targets := [][]string{
		{"a", "stores", "Session"},
		{"a", "stores", "SessionGate"},
		{"a", "stores", "SessionSpotlight"},
	}

	var violations []string
	for _, path := range paths {
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			t.Fatalf("relative path %s: %v", path, err)
		}
		relPath = filepath.ToSlash(relPath)
		for _, target := range targets {
			lines, err := selectorUsageLines(path, target)
			if err != nil {
				t.Fatalf("scan %s for %s: %v", path, strings.Join(target, "."), err)
			}
			for _, line := range lines {
				violations = append(violations, fmt.Sprintf("%s:%d uses %s", relPath, line, strings.Join(target, ".")))
			}
		}
	}

	sort.Strings(violations)
	if len(violations) == 0 {
		return
	}
	t.Fatalf("communication context bypasses session-owned read boundary:\n%s", strings.Join(violations, "\n"))
}

func TestSessionAndCommunicationApplicationsUseFocusedPolicyDependencies(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	paths := []string{
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "session_application.go"),
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "communication_application.go"),
	}

	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.Contains(string(content), "auth         Stores") {
			relPath, err := filepath.Rel(repoRoot, path)
			if err != nil {
				t.Fatalf("relative path %s: %v", path, err)
			}
			t.Fatalf("%s still carries full Stores for auth; use focused policyDependencies instead", filepath.ToSlash(relPath))
		}
	}
}

func TestCampaignApplicationUsesFocusedPolicyDependencies(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	path := filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "campaign_application.go")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if strings.Contains(string(content), "auth        Stores") {
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			t.Fatalf("relative path %s: %v", path, err)
		}
		t.Fatalf("%s still carries full Stores for auth; use focused policyDependencies instead", filepath.ToSlash(relPath))
	}
}

func TestCampaignServiceReadHandlersDoNotAccessStoresDirectly(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	serviceFiles := []string{
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "campaign_service_read.go"),
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "campaign_service_list.go"),
	}

	var violations []string
	for _, path := range serviceFiles {
		lines, err := selectorUsageLines(path, []string{"s", "stores"})
		if err != nil {
			t.Fatalf("scan %s: %v", path, err)
		}
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			t.Fatalf("relative path %s: %v", path, err)
		}
		for _, line := range lines {
			violations = append(violations, fmt.Sprintf("%s:%d", filepath.ToSlash(relPath), line))
		}
	}

	sort.Strings(violations)
	if len(violations) == 0 {
		return
	}
	t.Fatalf("campaign service read handlers access stores directly:\n%s", strings.Join(violations, "\n"))
}

func TestParticipantApplicationUsesFocusedPolicyDependencies(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	path := filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "participant_application.go")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if strings.Contains(string(content), "auth        Stores") {
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			t.Fatalf("relative path %s: %v", path, err)
		}
		t.Fatalf("%s still carries full Stores for auth; use focused policyDependencies instead", filepath.ToSlash(relPath))
	}
}

func TestParticipantServiceReadHandlersDoNotAccessStoresDirectly(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	path := filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "participant_service_read.go")

	lines, err := selectorUsageLines(path, []string{"s", "stores"})
	if err != nil {
		t.Fatalf("scan %s: %v", path, err)
	}
	if len(lines) == 0 {
		return
	}
	relPath, err := filepath.Rel(repoRoot, path)
	if err != nil {
		t.Fatalf("relative path %s: %v", path, err)
	}

	violations := make([]string, 0, len(lines))
	for _, line := range lines {
		violations = append(violations, fmt.Sprintf("%s:%d", filepath.ToSlash(relPath), line))
	}
	t.Fatalf("participant service read handlers access stores directly:\n%s", strings.Join(violations, "\n"))
}

func TestCharacterApplicationUsesFocusedPolicyDependencies(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	path := filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "character_application.go")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if strings.Contains(string(content), "auth        Stores") {
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			t.Fatalf("relative path %s: %v", path, err)
		}
		t.Fatalf("%s still carries full Stores for auth; use focused policyDependencies instead", filepath.ToSlash(relPath))
	}
}

func TestCharacterServiceReadHandlersDoNotAccessStoresDirectly(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	path := filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "character_service_read.go")

	lines, err := selectorUsageLines(path, []string{"s", "stores"})
	if err != nil {
		t.Fatalf("scan %s: %v", path, err)
	}
	if len(lines) == 0 {
		return
	}
	relPath, err := filepath.Rel(repoRoot, path)
	if err != nil {
		t.Fatalf("relative path %s: %v", path, err)
	}

	violations := make([]string, 0, len(lines))
	for _, line := range lines {
		violations = append(violations, fmt.Sprintf("%s:%d", filepath.ToSlash(relPath), line))
	}
	t.Fatalf("character service read handlers access stores directly:\n%s", strings.Join(violations, "\n"))
}

func TestForkApplicationUsesFocusedPolicyDependencies(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	path := filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "fork_application.go")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if strings.Contains(string(content), "auth        Stores") {
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			t.Fatalf("relative path %s: %v", path, err)
		}
		t.Fatalf("%s still carries full Stores for auth; use focused policyDependencies instead", filepath.ToSlash(relPath))
	}
}

func TestForkServiceReadHandlersDoNotAccessStoresDirectly(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	path := filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "fork_service_read.go")

	lines, err := selectorUsageLines(path, []string{"s", "stores"})
	if err != nil {
		t.Fatalf("scan %s: %v", path, err)
	}
	if len(lines) == 0 {
		return
	}
	relPath, err := filepath.Rel(repoRoot, path)
	if err != nil {
		t.Fatalf("relative path %s: %v", path, err)
	}

	violations := make([]string, 0, len(lines))
	for _, line := range lines {
		violations = append(violations, fmt.Sprintf("%s:%d", filepath.ToSlash(relPath), line))
	}
	t.Fatalf("fork service read handlers access stores directly:\n%s", strings.Join(violations, "\n"))
}

func TestForkApplicationForkUsesReplaySeam(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	path := filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "fork_application_fork.go")

	targets := [][]string{
		{"a", "stores", "Event", "ListEvents"},
		{"a", "stores", "Event", "AppendEvent"},
	}

	var violations []string
	for _, target := range targets {
		lines, err := selectorUsageLines(path, target)
		if err != nil {
			t.Fatalf("scan %s for %s: %v", path, strings.Join(target, "."), err)
		}
		for _, line := range lines {
			relPath, err := filepath.Rel(repoRoot, path)
			if err != nil {
				t.Fatalf("relative path %s: %v", path, err)
			}
			violations = append(violations, fmt.Sprintf("%s:%d uses %s", filepath.ToSlash(relPath), line, strings.Join(target, ".")))
		}
	}

	sort.Strings(violations)
	if len(violations) == 0 {
		return
	}
	t.Fatalf("fork replay bypasses the fork event replay seam:\n%s", strings.Join(violations, "\n"))
}

func TestSnapshotApplicationUsesFocusedPolicyDependencies(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	path := filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "snapshot_application.go")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if strings.Contains(string(content), "auth    Stores") {
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			t.Fatalf("relative path %s: %v", path, err)
		}
		t.Fatalf("%s still carries full Stores for auth; use focused policyDependencies instead", filepath.ToSlash(relPath))
	}
}

func TestSnapshotServiceHandlersDoNotAccessStoresDirectly(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	path := filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "snapshot_service.go")

	lines, err := selectorUsageLines(path, []string{"s", "stores"})
	if err != nil {
		t.Fatalf("scan %s: %v", path, err)
	}
	if len(lines) == 0 {
		return
	}
	relPath, err := filepath.Rel(repoRoot, path)
	if err != nil {
		t.Fatalf("relative path %s: %v", path, err)
	}

	violations := make([]string, 0, len(lines))
	for _, line := range lines {
		violations = append(violations, fmt.Sprintf("%s:%d", filepath.ToSlash(relPath), line))
	}
	t.Fatalf("snapshot service handlers access stores directly:\n%s", strings.Join(violations, "\n"))
}

func TestEventApplicationUsesFocusedPolicyDependencies(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	path := filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "event_application.go")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if strings.Contains(string(content), "auth  Stores") || strings.Contains(string(content), "auth Stores") {
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			t.Fatalf("relative path %s: %v", path, err)
		}
		t.Fatalf("%s still carries full Stores for auth; use focused policyDependencies instead", filepath.ToSlash(relPath))
	}
}

func TestEventServiceReadHandlersDoNotAccessStoresDirectly(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	serviceFiles := []string{
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "event_list_service.go"),
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "event_subscribe_service.go"),
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "timeline_service.go"),
	}

	var violations []string
	for _, path := range serviceFiles {
		lines, err := selectorUsageLines(path, []string{"s", "stores"})
		if err != nil {
			t.Fatalf("scan %s: %v", path, err)
		}
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			t.Fatalf("relative path %s: %v", path, err)
		}
		for _, line := range lines {
			violations = append(violations, fmt.Sprintf("%s:%d", filepath.ToSlash(relPath), line))
		}
	}

	sort.Strings(violations)
	if len(violations) == 0 {
		return
	}
	t.Fatalf("event/timeline service handlers access stores directly:\n%s", strings.Join(violations, "\n"))
}

func TestAuthorizationServiceUsesApplicationBoundary(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	path := filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "authorization_service.go")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	source := string(content)
	if strings.Contains(source, "stores    Stores") {
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			t.Fatalf("relative path %s: %v", path, err)
		}
		t.Fatalf("%s still carries full Stores; use authorizationApplication instead", filepath.ToSlash(relPath))
	}
	if !strings.Contains(source, "app authorizationApplication") {
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			t.Fatalf("relative path %s: %v", path, err)
		}
		t.Fatalf("%s no longer exposes the authorization application boundary", filepath.ToSlash(relPath))
	}
}

func TestAuthorizationServiceHandlersDoNotEvaluateDirectly(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	serviceFiles := []string{
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "authorization_can_service.go"),
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "authorization_batch_service.go"),
	}

	var violations []string
	for _, path := range serviceFiles {
		lines, err := selectorUsageLines(path, []string{"s", "evaluator"})
		if err != nil {
			t.Fatalf("scan %s: %v", path, err)
		}
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			t.Fatalf("relative path %s: %v", path, err)
		}
		for _, line := range lines {
			violations = append(violations, fmt.Sprintf("%s:%d", filepath.ToSlash(relPath), line))
		}
	}

	sort.Strings(violations)
	if len(violations) == 0 {
		return
	}
	t.Fatalf("authorization service handlers bypass the application boundary:\n%s", strings.Join(violations, "\n"))
}

func TestSessionServiceReadHandlersDoNotAccessStoresDirectly(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	serviceFiles := []string{
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "session_service_lifecycle.go"),
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "session_service_spotlight.go"),
		filepath.Join(repoRoot, "internal", "services", "game", "api", "grpc", "game", "session_service_user.go"),
	}

	var violations []string
	for _, path := range serviceFiles {
		lines, err := selectorUsageLines(path, []string{"s", "stores"})
		if err != nil {
			t.Fatalf("scan %s: %v", path, err)
		}
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			t.Fatalf("relative path %s: %v", path, err)
		}
		for _, line := range lines {
			violations = append(violations, fmt.Sprintf("%s:%d", filepath.ToSlash(relPath), line))
		}
	}

	sort.Strings(violations)
	if len(violations) == 0 {
		return
	}
	t.Fatalf("session service read handlers access stores directly:\n%s", strings.Join(violations, "\n"))
}

func TestSessionGateStoreDoesNotUseLegacyProjectionJSONBlobs(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	path := filepath.Join(repoRoot, "internal", "services", "game", "storage", "sqlite", "store_projection_session.go")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	source := string(content)
	legacyFields := []string{"MetadataJson", "ProgressJson", "ResolutionJson"}
	var violations []string
	for _, field := range legacyFields {
		if strings.Contains(source, field) {
			relPath, err := filepath.Rel(repoRoot, path)
			if err != nil {
				t.Fatalf("relative path %s: %v", path, err)
			}
			violations = append(violations, fmt.Sprintf("%s uses %s", filepath.ToSlash(relPath), field))
		}
	}

	sort.Strings(violations)
	if len(violations) == 0 {
		return
	}
	t.Fatalf("session gate SQLite path still depends on legacy JSON blob fields:\n%s", strings.Join(violations, "\n"))
}

func repoRootFromThisFile(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..", ".."))
}

func appendEventCallLines(path string) ([]int, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	lines := make([]int, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if sel.Sel == nil || sel.Sel.Name != "AppendEvent" {
			return true
		}
		parentSelector, ok := sel.X.(*ast.SelectorExpr)
		if !ok || parentSelector.Sel == nil || parentSelector.Sel.Name != "Event" {
			return true
		}
		line := fset.Position(sel.Sel.Pos()).Line
		lines = append(lines, line)
		return true
	})
	return lines, nil
}

func namedCallLines(path string, target string) ([]int, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	lines := make([]int, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch typed := call.Fun.(type) {
		case *ast.Ident:
			if typed.Name == target {
				lines = append(lines, fset.Position(call.Lparen).Line)
			}
		case *ast.IndexExpr:
			if ident, ok := typed.X.(*ast.Ident); ok && ident.Name == target {
				lines = append(lines, fset.Position(call.Lparen).Line)
			}
		case *ast.IndexListExpr:
			if ident, ok := typed.X.(*ast.Ident); ok && ident.Name == target {
				lines = append(lines, fset.Position(call.Lparen).Line)
			}
		}
		return true
	})
	return lines, nil
}

func domainExecuteCallLines(path string) ([]int, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	lines := make([]int, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		callPath := selectorPath(call.Fun)
		if !strings.HasSuffix(callPath, ".Domain.Execute") {
			return true
		}
		lines = append(lines, fset.Position(call.Lparen).Line)
		return true
	})
	return lines, nil
}

func selectorUsageLines(path string, target []string) ([]int, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	lines := make([]int, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		sel, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		path := strings.Split(selectorPath(sel), ".")
		if len(path) != len(target) {
			return true
		}
		for i := range path {
			if path[i] != target[i] {
				return true
			}
		}
		lines = append(lines, fset.Position(sel.Sel.Pos()).Line)
		return true
	})
	return lines, nil
}

func selectorCallLines(path string, target []string) ([]int, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	lines := make([]int, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		var callPath string
		switch typed := call.Fun.(type) {
		case *ast.SelectorExpr:
			callPath = selectorPath(typed)
		case *ast.IndexExpr:
			callPath = selectorPath(typed.X)
		case *ast.IndexListExpr:
			callPath = selectorPath(typed.X)
		default:
			return true
		}
		pathParts := strings.Split(callPath, ".")
		if len(pathParts) != len(target) {
			return true
		}
		for i := range pathParts {
			if pathParts[i] != target[i] {
				return true
			}
		}
		lines = append(lines, fset.Position(call.Lparen).Line)
		return true
	})
	return lines, nil
}

func selectorPath(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.SelectorExpr:
		prefix := selectorPath(typed.X)
		if prefix == "" {
			return typed.Sel.Name
		}
		return prefix + "." + typed.Sel.Name
	case *ast.Ident:
		return typed.Name
	case *ast.ParenExpr:
		return selectorPath(typed.X)
	case *ast.StarExpr:
		return selectorPath(typed.X)
	default:
		return ""
	}
}
