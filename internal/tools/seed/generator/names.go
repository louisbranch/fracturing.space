package generator

import (
	"fmt"
	"strings"
)

// nameRegistry keeps generated display names unique to avoid auth username collisions.
type nameRegistry struct {
	counts map[string]int
}

func newNameRegistry() *nameRegistry {
	return &nameRegistry{counts: make(map[string]int)}
}

func (r *nameRegistry) uniqueDisplayName(base string) string {
	trimmed := strings.TrimSpace(base)
	if trimmed == "" {
		return base
	}
	if r.counts == nil {
		r.counts = make(map[string]int)
	}
	count := r.counts[trimmed]
	r.counts[trimmed] = count + 1
	if count == 0 {
		return trimmed
	}
	return fmt.Sprintf("%s-%d", trimmed, count+1)
}

// uniqueDisplayName ensures display names remain stable but distinct across a seed run.
func (g *Generator) uniqueDisplayName(base string) string {
	if g.nameRegistry == nil {
		g.nameRegistry = newNameRegistry()
	}
	return g.nameRegistry.uniqueDisplayName(base)
}
