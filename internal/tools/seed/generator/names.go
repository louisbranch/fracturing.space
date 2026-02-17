package generator

import (
	"fmt"
	"strings"
)

const seedEmailDomain = "example.com"

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

func (g *Generator) seedPrimaryEmail(displayName string) string {
	base := strings.TrimSpace(strings.ToLower(displayName))
	if base == "" {
		base = "seed-user"
	}
	// Keep names stable and email-safe for seeding where we do not need fancy
	// username semantics anymore.
	var b strings.Builder
	for _, ch := range base {
		switch {
		case ch >= 'a' && ch <= 'z', ch >= '0' && ch <= '9':
			b.WriteRune(ch)
		case ch == ' ' || ch == '_' || ch == '.':
			b.WriteRune('-')
		}
	}
	local := strings.Trim(b.String(), "-")
	if local == "" {
		local = "seed-user"
	}
	if len(local) > 32 {
		local = local[:32]
	}
	if local == "" {
		local = "seed-user"
	}
	return local + "@" + seedEmailDomain
}
