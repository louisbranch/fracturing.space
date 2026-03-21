package web

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/testast"
)

func TestDependenciesDelegateBindingWithoutGeneratedClientImports(t *testing.T) {
	t.Parallel()

	testast.AssertImportsDoNotContainPrefix(t, "dependencies.go", "github.com/louisbranch/fracturing.space/api/gen/go/")

	for _, tc := range []struct {
		funcName string
		want     []selectorCall
	}{
		{
			funcName: "BindAuthDependency",
			want: []selectorCall{
				{recv: "principal", name: "BindAuthDependency"},
				{recv: "modules", name: "BindAuthDependency"},
			},
		},
		{
			funcName: "BindSocialDependency",
			want: []selectorCall{
				{recv: "principal", name: "BindSocialDependency"},
				{recv: "modules", name: "BindSocialDependency"},
			},
		},
		{
			funcName: "BindGameDependency",
			want: []selectorCall{
				{recv: "modules", name: "BindGameDependency"},
			},
		},
		{
			funcName: "BindAIDependency",
			want: []selectorCall{
				{recv: "modules", name: "BindAIDependency"},
			},
		},
		{
			funcName: "BindDiscoveryDependency",
			want: []selectorCall{
				{recv: "modules", name: "BindDiscoveryDependency"},
			},
		},
		{
			funcName: "BindUserHubDependency",
			want: []selectorCall{
				{recv: "modules", name: "BindUserHubDependency"},
			},
		},
		{
			funcName: "BindNotificationsDependency",
			want: []selectorCall{
				{recv: "principal", name: "BindNotificationsDependency"},
				{recv: "modules", name: "BindNotificationsDependency"},
			},
		},
		{
			funcName: "BindStatusDependency",
			want: []selectorCall{
				{recv: "modules", name: "BindStatusDependency"},
			},
		},
	} {
		for _, call := range tc.want {
			testast.AssertFuncCallsSelector(t, "dependencies.go", tc.funcName, call.recv, call.name)
		}
	}
}

func TestStartupDependencyDescriptorsStayServiceOwnedAndBinderBased(t *testing.T) {
	t.Parallel()

	testast.AssertImportsDoNotContainPrefix(t, "startup_dependencies.go", "github.com/louisbranch/fracturing.space/api/gen/go/")
	for _, typeName := range []string{
		"StartupDependencyDescriptor",
		"StartupDependencyPolicy",
	} {
		testast.AssertTypeExists(t, "startup_dependencies.go", typeName)
	}
	for _, funcName := range []string{
		"StartupDependencyDescriptors",
		"LookupStartupDependencyDescriptor",
	} {
		testast.AssertFuncExists(t, "startup_dependencies.go", funcName)
	}
}

type selectorCall struct {
	recv string
	name string
}
