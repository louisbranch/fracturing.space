package gametools

import (
	"context"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
)

type toolExecutor func(*DirectSession, context.Context, []byte) (orchestration.ToolResult, error)

type productionToolDefinition struct {
	Tool    orchestration.Tool
	Execute toolExecutor
}

type productionToolRegistry struct {
	definitions []productionToolDefinition
	byName      map[string]productionToolDefinition
}

// defaultRegistry is the standard production registry used by NewDirectDialer.
// Kept as a package-level cache since the definitions are static; injection
// happens at the DirectDialer/DirectSession level so tests can substitute.
var defaultRegistry = newProductionToolRegistry()

func newProductionToolRegistry() productionToolRegistry {
	definitions := newProductionToolDefinitions()

	byName := make(map[string]productionToolDefinition, len(definitions))
	for i, definition := range definitions {
		name := definition.Tool.Name
		if name == "" {
			panic("gametools: production tool name is required")
		}
		if definition.Execute == nil {
			panic(fmt.Sprintf("gametools: production tool %q is missing an executor", name))
		}
		definitions[i] = definition
		if _, exists := byName[name]; exists {
			panic(fmt.Sprintf("gametools: duplicate production tool %q", name))
		}
		byName[name] = definition
	}

	return productionToolRegistry{
		definitions: definitions,
		byName:      byName,
	}
}

func (r productionToolRegistry) tools() []orchestration.Tool {
	tools := make([]orchestration.Tool, 0, len(r.definitions))
	for _, definition := range r.definitions {
		tools = append(tools, definition.Tool)
	}
	return tools
}

func (r productionToolRegistry) lookup(name string) (productionToolDefinition, bool) {
	definition, ok := r.byName[name]
	return definition, ok
}

// ProductionToolNames returns the concrete production tool profile owned by
// the direct game-tools bridge.
func ProductionToolNames() []string {
	names := make([]string, 0, len(defaultRegistry.definitions))
	for _, definition := range defaultRegistry.definitions {
		names = append(names, definition.Tool.Name)
	}
	return names
}
