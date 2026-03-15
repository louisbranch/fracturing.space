package contenttransport

import "context"

func (catalog *contentCatalog) listClasses(ctx context.Context) error {
	classes, err := catalog.store.ListDaggerheartClasses(ctx)
	if err != nil {
		return err
	}
	catalog.classes = classes
	return nil
}

func (catalog *contentCatalog) listSubclasses(ctx context.Context) error {
	subclasses, err := catalog.store.ListDaggerheartSubclasses(ctx)
	if err != nil {
		return err
	}
	catalog.subclasses = subclasses
	return nil
}

func (catalog *contentCatalog) listHeritages(ctx context.Context) error {
	heritages, err := catalog.store.ListDaggerheartHeritages(ctx)
	if err != nil {
		return err
	}
	catalog.heritages = heritages
	return nil
}

func (catalog *contentCatalog) listExperiences(ctx context.Context) error {
	experiences, err := catalog.store.ListDaggerheartExperiences(ctx)
	if err != nil {
		return err
	}
	catalog.experiences = experiences
	return nil
}

func (catalog *contentCatalog) localizeClasses(ctx context.Context) error {
	return localizeClasses(ctx, catalog.store, catalog.locale, catalog.classes)
}

func (catalog *contentCatalog) localizeSubclasses(ctx context.Context) error {
	return localizeSubclasses(ctx, catalog.store, catalog.locale, catalog.subclasses)
}

func (catalog *contentCatalog) localizeHeritages(ctx context.Context) error {
	return localizeHeritages(ctx, catalog.store, catalog.locale, catalog.heritages)
}

func (catalog *contentCatalog) localizeExperiences(ctx context.Context) error {
	return localizeExperiences(ctx, catalog.store, catalog.locale, catalog.experiences)
}
