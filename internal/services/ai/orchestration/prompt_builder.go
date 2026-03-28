package orchestration

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PromptBuilderConfig holds the dependencies for building campaign turn prompts.
type PromptBuilderConfig struct {
	// Collector gathers the typed session brief before rendering.
	Collector SessionBriefCollector
	// Augmenter adds optional supplemental brief sections after collection.
	Augmenter PromptAugmenter
	// Renderer turns one collected brief into the final prompt text.
	Renderer PromptRenderer
}

type defaultPromptBuilder struct {
	collector SessionBriefCollector
	augmenter PromptAugmenter
	renderer  PromptRenderer
}

// newDegradedPromptBuilder creates an explicit degraded-mode prompt builder
// with core context sources but no game-system-specific sources or pre-loaded
// instruction content. Production callers should inject a fully configured
// PromptBuilder via RunnerConfig.
func newDegradedPromptBuilder() PromptBuilder {
	return NewPromptBuilder(PromptBuilderConfig{
		Collector: NewCoreContextSourceRegistry(),
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
		augmenter: cfg.Augmenter,
		renderer:  renderer,
	}
}

func (pb *defaultPromptBuilder) Build(ctx context.Context, sess Session, input PromptInput) (string, error) {
	brief, err := pb.collector.CollectBrief(ctx, sess, input)
	if err != nil {
		return "", fmt.Errorf("collect context sources: %w", err)
	}
	if pb.augmenter != nil {
		contribution, err := pb.augmenter.Augment(ctx, sess, brief, input)
		if err != nil {
			return "", fmt.Errorf("augment prompt context: %w", err)
		}
		if err := brief.mergeContribution("prompt_augmenter", contribution); err != nil {
			return "", fmt.Errorf("merge prompt augmentation: %w", err)
		}
	}
	return pb.renderer.Render(brief, input), nil
}

func readOptionalResource(ctx context.Context, sess Session, uri string) (string, error) {
	value, err := sess.ReadResource(ctx, uri)
	if err != nil {
		if isResourceNotFound(err) {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

// isResourceNotFound checks gRPC status codes first and falls back to string
// matching for errors that have already been unwrapped by intermediaries.
func isResourceNotFound(err error) bool {
	if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
		return true
	}
	errText := strings.ToLower(err.Error())
	return strings.Contains(errText, "not found") || strings.Contains(errText, "missing resource")
}
