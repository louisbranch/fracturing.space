package web

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/testast"
)

func TestDependenciesCallLeafBindFunctions(t *testing.T) {
	t.Parallel()

	// dependencies.go now calls leaf-level sub-module bind functions directly
	// instead of going through modules.BindXxx forwarding functions. The proto
	// gen imports are allowed for the two dashboard-sync client constructions
	// that have no sub-module bind function.
	for _, tc := range []struct {
		funcName string
		want     []selectorCall
	}{
		{
			funcName: "BindAuthDependency",
			want: []selectorCall{
				{recv: "principal", name: "BindAuthDependency"},
				{recv: "publicauth", name: "BindAuthDependency"},
				{recv: "profile", name: "BindAuthDependency"},
				{recv: "settings", name: "BindAuthDependency"},
				{recv: "campaigns", name: "BindAuthDependency"},
				{recv: "invite", name: "BindAuthDependency"},
			},
		},
		{
			funcName: "BindSocialDependency",
			want: []selectorCall{
				{recv: "principal", name: "BindSocialDependency"},
				{recv: "profile", name: "BindSocialDependency"},
				{recv: "settings", name: "BindSocialDependency"},
				{recv: "campaigns", name: "BindSocialDependency"},
			},
		},
		{
			funcName: "BindGameDependency",
			want: []selectorCall{
				{recv: "campaigns", name: "BindGameDependency"},
			},
		},
		{
			funcName: "BindInviteDependency",
			want: []selectorCall{
				{recv: "campaigns", name: "BindInviteDependency"},
				{recv: "invite", name: "BindInviteDependency"},
			},
		},
		{
			funcName: "BindAIDependency",
			want: []selectorCall{
				{recv: "settings", name: "BindAIDependency"},
				{recv: "campaigns", name: "BindAIDependency"},
			},
		},
		{
			funcName: "BindDiscoveryDependency",
			want: []selectorCall{
				{recv: "discovery", name: "BindDependency"},
				{recv: "campaigns", name: "BindDiscoveryDependency"},
			},
		},
		{
			funcName: "BindUserHubDependency",
			want: []selectorCall{
				{recv: "dashboard", name: "BindUserHubDependency"},
			},
		},
		{
			funcName: "BindNotificationsDependency",
			want: []selectorCall{
				{recv: "principal", name: "BindNotificationsDependency"},
				{recv: "notifications", name: "BindDependency"},
			},
		},
		{
			funcName: "BindStatusDependency",
			want: []selectorCall{
				{recv: "dashboard", name: "BindStatusDependency"},
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
