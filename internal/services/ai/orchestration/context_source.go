package orchestration

import (
	"context"
	"strings"
)

// InteractionStateSnapshot carries the typed interaction facts the prompt
// builder needs without re-parsing rendered prompt sections.
type InteractionStateSnapshot struct {
	ActiveSceneID string
}

// Bootstrap reports whether the interaction state has no active scene yet.
func (s InteractionStateSnapshot) Bootstrap() bool {
	return strings.TrimSpace(s.ActiveSceneID) == ""
}

// BriefContribution is one source's contribution to the collected session
// brief: rendered sections plus any typed facts discovered while reading
// authoritative resources.
type BriefContribution struct {
	Sections         []BriefSection
	InteractionState *InteractionStateSnapshot
}

// SessionBrief is the typed result collected before prompt rendering.
type SessionBrief struct {
	Sections         []BriefSection
	InteractionState *InteractionStateSnapshot
}

// Bootstrap reports whether the collected session brief represents bootstrap
// mode with no active scene selected yet.
func (b SessionBrief) Bootstrap() bool {
	return b.InteractionState != nil && b.InteractionState.Bootstrap()
}

// ContextSource contributes to the typed session brief used for prompt
// assembly. Game systems implement this interface to inject system-specific
// context alongside the core campaign context.
type ContextSource interface {
	Collect(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error)
}

// ContextSourceFunc adapts a plain function to the ContextSource interface.
type ContextSourceFunc func(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error)

// Collect implements ContextSource.
func (f ContextSourceFunc) Collect(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error) {
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

// CollectBrief invokes all registered sources and returns the combined typed
// session brief. If any source returns an error, collection stops and the
// error is returned.
func (r *ContextSourceRegistry) CollectBrief(ctx context.Context, sess Session, input PromptInput) (SessionBrief, error) {
	if r == nil {
		return SessionBrief{}, nil
	}
	var brief SessionBrief
	for _, src := range r.sources {
		contribution, err := src.Collect(ctx, sess, input)
		if err != nil {
			return SessionBrief{}, err
		}
		brief.Sections = append(brief.Sections, contribution.Sections...)
		if contribution.InteractionState != nil {
			brief.InteractionState = contribution.InteractionState
		}
	}
	return brief, nil
}

// CollectSections returns only the rendered sections from the collected brief.
func (r *ContextSourceRegistry) CollectSections(ctx context.Context, sess Session, input PromptInput) ([]BriefSection, error) {
	brief, err := r.CollectBrief(ctx, sess, input)
	if err != nil {
		return nil, err
	}
	return brief.Sections, nil
}
