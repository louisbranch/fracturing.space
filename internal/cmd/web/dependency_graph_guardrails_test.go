package web

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/testast"
)

func TestDependencyGraphUsesServiceOwnedDescriptorsAndOnlyStatusProto(t *testing.T) {
	t.Parallel()

	testast.AssertImportPresent(t, "dependency_graph.go", "github.com/louisbranch/fracturing.space/internal/services/web")
	testast.AssertImportPresent(t, "dependency_graph.go", "github.com/louisbranch/fracturing.space/api/gen/go/status/v1")
	testast.AssertImportsDoNotContainPrefixExcept(t, "dependency_graph.go", "github.com/louisbranch/fracturing.space/api/gen/go/", "github.com/louisbranch/fracturing.space/api/gen/go/status/v1")
	testast.AssertFuncCallsSelector(t, "dependency_graph.go", "dependencyRequirements", "web", "StartupDependencyDescriptors")
}
