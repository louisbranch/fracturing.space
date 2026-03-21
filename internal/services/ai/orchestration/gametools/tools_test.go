package gametools

import (
	"context"
	"reflect"
	"testing"
)

func TestProductionToolRegistryMatchesSessionCatalog(t *testing.T) {
	sess := NewDirectSession(Clients{}, sessionContext{})

	tools, err := sess.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	names := make([]string, 0, len(tools))
	seen := make(map[string]struct{}, len(tools))
	for _, tool := range tools {
		if tool.Name == "" {
			t.Fatalf("tool has empty name: %#v", tool)
		}
		if _, exists := seen[tool.Name]; exists {
			t.Fatalf("duplicate tool name %q", tool.Name)
		}
		seen[tool.Name] = struct{}{}
		names = append(names, tool.Name)
		if _, ok := registry.lookup(tool.Name); !ok {
			t.Fatalf("tool %q is listed but not dispatchable", tool.Name)
		}
	}

	if !reflect.DeepEqual(names, ProductionToolNames()) {
		t.Fatalf("catalog names = %#v, want %#v", names, ProductionToolNames())
	}
}
