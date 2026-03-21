package orchestration

import "strings"

// ToolPolicy defines which session-exposed tools are visible to one runner.
type ToolPolicy interface {
	Allows(name string) bool
}

type allowAllToolPolicy struct{}

// AllowAllToolPolicy returns a tool policy that does not apply any additional
// filtering beyond the session's own advertised catalog.
func AllowAllToolPolicy() ToolPolicy {
	return allowAllToolPolicy{}
}

func (allowAllToolPolicy) Allows(string) bool { return true }

type staticToolPolicy struct {
	allowed map[string]struct{}
}

// NewStaticToolPolicy returns a name-based allowlist policy for one runner.
func NewStaticToolPolicy(names []string) ToolPolicy {
	allowed := make(map[string]struct{}, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		allowed[name] = struct{}{}
	}
	return staticToolPolicy{allowed: allowed}
}

func (p staticToolPolicy) Allows(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	_, ok := p.allowed[name]
	return ok
}

func filterTools(tools []Tool, policy ToolPolicy) []Tool {
	filtered := make([]Tool, 0, len(tools))
	for _, tool := range tools {
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			continue
		}
		if policy != nil && !policy.Allows(name) {
			continue
		}
		tool.Name = name
		filtered = append(filtered, tool)
	}
	return filtered
}
