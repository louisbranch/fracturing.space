package orchestration

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"go.opentelemetry.io/otel/attribute"
)

// InteractionStateSnapshot carries the typed interaction facts the prompt
// builder needs without re-parsing rendered prompt sections.
type InteractionStateSnapshot struct {
	ActiveSceneID        string
	PlayerPhaseStatus    string
	OOCOpen              bool
	OOCResolutionPending bool
}

// Bootstrap reports whether the interaction state has no active scene yet.
func (s InteractionStateSnapshot) Bootstrap() bool {
	return s.ActiveSceneID == ""
}

type InteractionTurnMode string

const (
	InteractionTurnModeBootstrap          InteractionTurnMode = "bootstrap"
	InteractionTurnModeReviewResolution   InteractionTurnMode = "review_resolution"
	InteractionTurnModeOOCOpen            InteractionTurnMode = "ooc_open"
	InteractionTurnModeOOCCloseResolution InteractionTurnMode = "ooc_resume_resolution"
	InteractionTurnModeActiveScene        InteractionTurnMode = "active_scene"
)

func (s InteractionStateSnapshot) TurnMode() InteractionTurnMode {
	switch {
	case s.Bootstrap():
		return InteractionTurnModeBootstrap
	case s.OOCOpen:
		return InteractionTurnModeOOCOpen
	case s.OOCResolutionPending:
		return InteractionTurnModeOOCCloseResolution
	case strings.EqualFold(strings.TrimSpace(s.PlayerPhaseStatus), "gm_review"):
		return InteractionTurnModeReviewResolution
	default:
		return InteractionTurnModeActiveScene
	}
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

func (b SessionBrief) TurnMode() InteractionTurnMode {
	if b.InteractionState == nil {
		return InteractionTurnModeActiveScene
	}
	return b.InteractionState.TurnMode()
}

// ContextSourceFunc adapts a plain function to the ContextSource interface.
type ContextSourceFunc func(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error)

// Collect implements ContextSource.
func (f ContextSourceFunc) Collect(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error) {
	return f(ctx, sess, input)
}

// ContextSourceName derives a stable default name for the wrapped function so
// the registry can emit useful spans without every caller naming sources
// manually.
func (f ContextSourceFunc) ContextSourceName() string {
	value := reflect.ValueOf(f)
	if value.IsValid() && value.Kind() == reflect.Func {
		if fn := runtime.FuncForPC(value.Pointer()); fn != nil {
			return sanitizeContextSourceName(fn.Name())
		}
	}
	return fallbackContextSourceName(-1)
}

type registeredContextSource struct {
	name   string
	source ContextSource
}

// ContextSourceRegistry holds an ordered list of context sources that the
// prompt builder composes into the session brief.
type ContextSourceRegistry struct {
	sources []registeredContextSource
}

// NewContextSourceRegistry creates an empty registry.
func NewContextSourceRegistry() *ContextSourceRegistry {
	return &ContextSourceRegistry{}
}

// Register adds a context source to the registry.
func (r *ContextSourceRegistry) Register(src ContextSource) {
	r.RegisterNamed("", src)
}

// RegisterAll adds each non-nil source to the registry in order.
func (r *ContextSourceRegistry) RegisterAll(sources ...ContextSource) {
	for _, src := range sources {
		r.Register(src)
	}
}

// RegisterNamed adds a context source with an explicit span/debug name.
func (r *ContextSourceRegistry) RegisterNamed(name string, src ContextSource) {
	if src != nil {
		r.sources = append(r.sources, registeredContextSource{
			name:   resolveContextSourceName(name, src, len(r.sources)),
			source: src,
		})
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
	for i, entry := range r.sources {
		sourceCtx, sourceSpan := orchestrationTracer().Start(ctx, "ai.orchestration.context_source")
		sourceSpan.SetAttributes(
			attribute.String("ai.orchestration.context_source.name", entry.name),
			attribute.Int("ai.orchestration.context_source.index", i),
		)

		contribution, err := entry.source.Collect(sourceCtx, sess, input)
		if err != nil {
			recordSpanError(sourceSpan, err)
			sourceSpan.End()
			return SessionBrief{}, err
		}
		sourceSpan.SetAttributes(
			attribute.Int("ai.orchestration.context_source.section_count", len(contribution.Sections)),
			attribute.Bool("ai.orchestration.context_source.has_interaction_state", contribution.InteractionState != nil),
		)
		if err := brief.mergeContribution(entry.name, contribution); err != nil {
			recordSpanError(sourceSpan, err)
			sourceSpan.End()
			return SessionBrief{}, err
		}
		sourceSpan.End()
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

func (b *SessionBrief) mergeContribution(sourceName string, contribution BriefContribution) error {
	b.Sections = append(b.Sections, contribution.Sections...)
	if contribution.InteractionState == nil {
		return nil
	}
	if b.InteractionState != nil {
		return fmt.Errorf("context source %q attempted to overwrite interaction state", sourceName)
	}
	b.InteractionState = contribution.InteractionState
	return nil
}

func resolveContextSourceName(explicit string, src ContextSource, index int) string {
	if name := sanitizeContextSourceName(explicit); name != "" {
		return name
	}
	return inferContextSourceName(src, index)
}

func inferContextSourceName(src ContextSource, index int) string {
	if src == nil {
		return fallbackContextSourceName(index)
	}
	if named, ok := src.(interface{ ContextSourceName() string }); ok {
		if name := strings.TrimSpace(named.ContextSourceName()); name != "" {
			return sanitizeContextSourceName(name)
		}
	}
	value := reflect.ValueOf(src)
	if value.IsValid() && value.Kind() == reflect.Func {
		if fn := runtime.FuncForPC(value.Pointer()); fn != nil {
			return sanitizeContextSourceName(fn.Name())
		}
	}
	typ := reflect.TypeOf(src)
	if typ != nil {
		return sanitizeContextSourceName(typ.String())
	}
	return fallbackContextSourceName(index)
}

func sanitizeContextSourceName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if slash := strings.LastIndex(name, "/"); slash >= 0 {
		name = name[slash+1:]
	}
	name = strings.TrimSuffix(name, "-fm")
	if dot := strings.LastIndex(name, "."); dot >= 0 {
		name = name[dot+1:]
	}
	name = strings.TrimSpace(name)
	return name
}

func fallbackContextSourceName(index int) string {
	if index < 0 {
		return "context_source"
	}
	return fmt.Sprintf("context_source_%d", index+1)
}
