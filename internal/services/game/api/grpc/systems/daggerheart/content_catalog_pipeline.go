package daggerheart

import (
	"context"
	"fmt"
)

type contentCatalogStep struct {
	name string
	run  func(context.Context) error
}

func runContentCatalogSteps(ctx context.Context, steps []contentCatalogStep) error {
	for _, step := range steps {
		if step.run == nil {
			continue
		}
		if err := step.run(ctx); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
	}
	return nil
}
