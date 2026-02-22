package daggerheart

import "fmt"

type contentCatalogStep struct {
	name string
	run  func() error
}

func runContentCatalogSteps(steps []contentCatalogStep) error {
	for _, step := range steps {
		if step.run == nil {
			continue
		}
		if err := step.run(); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
	}
	return nil
}
