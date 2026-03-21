package orchestration

import (
	"context"
	"fmt"
	"strings"
)

// PromptBuilderConfig holds the dependencies for building campaign turn prompts.
type PromptBuilderConfig struct {
	// Collector gathers the typed session brief before rendering.
	Collector SessionBriefCollector
	// Renderer turns one collected brief into the final prompt text.
	Renderer PromptRenderer
}

type defaultPromptBuilder struct {
	collector SessionBriefCollector
	renderer  PromptRenderer
}

// newDegradedPromptBuilder creates an explicit degraded-mode prompt builder
// with the canonical context-source registry but no pre-loaded instruction
// content.
func newDegradedPromptBuilder() PromptBuilder {
	reg := NewContextSourceRegistry()
	for _, src := range CoreContextSources() {
		reg.Register(src)
	}
	for _, src := range DaggerheartContextSources() {
		reg.Register(src)
	}
	return NewPromptBuilder(PromptBuilderConfig{
		Collector: reg,
		Renderer: NewBriefPromptRenderer(BriefPromptRendererConfig{
			Policy: DefaultPromptRenderPolicy(),
		}),
	})
}

// NewPromptBuilder creates a prompt builder from one collector and one
// renderer. Nil collaborators degrade to the canonical default implementations.
func NewPromptBuilder(cfg PromptBuilderConfig) PromptBuilder {
	collector := cfg.Collector
	if collector == nil {
		collector = NewContextSourceRegistry()
	}
	renderer := cfg.Renderer
	if renderer == nil {
		renderer = NewBriefPromptRenderer(BriefPromptRendererConfig{
			Policy: DefaultPromptRenderPolicy(),
		})
	}
	return &defaultPromptBuilder{
		collector: collector,
		renderer:  renderer,
	}
}

func (pb *defaultPromptBuilder) Build(ctx context.Context, sess Session, input PromptInput) (string, error) {
	brief, err := pb.collector.CollectBrief(ctx, sess, input)
	if err != nil {
		return "", fmt.Errorf("collect context sources: %w", err)
	}
	return pb.renderer.Render(brief, input), nil
}

func readOptionalResource(ctx context.Context, sess Session, uri string) (string, error) {
	value, err := sess.ReadResource(ctx, uri)
	if err != nil {
		errText := strings.ToLower(err.Error())
		if strings.Contains(errText, "not found") || strings.Contains(errText, "missing resource") {
			return "", nil
		}
		return "", err
	}
	return value, nil
}
