package orchestration

import "context"

// ContextSource contributes BriefSections to the prompt assembly.
// Game systems implement this interface to inject system-specific context
// (e.g. dice rules, domain cards, active mechanics) alongside the core
// campaign context.
type ContextSource interface {
	Sections(ctx context.Context, sess Session, input Input) ([]BriefSection, error)
}

// ContextSourceFunc adapts a plain function to the ContextSource interface.
type ContextSourceFunc func(ctx context.Context, sess Session, input Input) ([]BriefSection, error)

// Sections implements ContextSource.
func (f ContextSourceFunc) Sections(ctx context.Context, sess Session, input Input) ([]BriefSection, error) {
	return f(ctx, sess, input)
}

// ContextSourceRegistry holds an ordered list of context sources that the
// prompt builder composes into the session brief.
type ContextSourceRegistry struct {
	sources []ContextSource
}

// NewContextSourceRegistry creates an empty registry.
func NewContextSourceRegistry() *ContextSourceRegistry {
	return &ContextSourceRegistry{}
}

// Register adds a context source to the registry.
func (r *ContextSourceRegistry) Register(src ContextSource) {
	if src != nil {
		r.sources = append(r.sources, src)
	}
}

// CollectSections invokes all registered sources and returns the combined
// sections. If any source returns an error, collection stops and the error
// is returned.
func (r *ContextSourceRegistry) CollectSections(ctx context.Context, sess Session, input Input) ([]BriefSection, error) {
	if r == nil {
		return nil, nil
	}
	var all []BriefSection
	for _, src := range r.sources {
		sections, err := src.Sections(ctx, sess, input)
		if err != nil {
			return nil, err
		}
		all = append(all, sections...)
	}
	return all, nil
}
