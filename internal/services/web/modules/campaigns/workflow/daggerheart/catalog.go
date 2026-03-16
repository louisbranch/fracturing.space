package daggerheart

import (
	"sort"
	"strings"

	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
)

// BuildView maps explicit workflow inputs directly to the workflow-owned page
// model.
func (w Workflow) BuildView(
	progress campaignworkflow.Progress,
	catalog campaignworkflow.Catalog,
	profile campaignworkflow.Profile,
) campaignworkflow.CharacterCreationView {
	return w.CreationView(w.assembleCatalog(progress, catalog, profile))
}

// assembleCatalog builds the Daggerheart-specific catalog state from generic
// gateway data (progress, catalog, profile).
func (Workflow) assembleCatalog(
	progress campaignworkflow.Progress,
	catalog campaignworkflow.Catalog,
	profile campaignworkflow.Profile,
) catalogCreation {
	creation := buildCatalogCreation(progress, profile)
	var domainNameByID map[string]string
	creation.Domains, domainNameByID = buildDomains(catalog.Domains)
	var classDomainsByID map[string]map[string]struct{}
	creation.Classes, classDomainsByID = buildClasses(catalog.Classes)
	creation.Subclasses = buildSubclasses(catalog.Subclasses)
	creation.Ancestries, creation.Communities = buildHeritages(catalog.Heritages)
	creation.CompanionExperiences = append([]campaignworkflow.CompanionExperience(nil), catalog.CompanionExperiences...)
	creation.PrimaryWeapons, creation.SecondaryWeapons = buildWeapons(catalog.Weapons)
	creation.Armor = buildArmor(catalog.Armor)
	creation.PotionItems = buildPotionItems(catalog.Items)
	creation.DomainCards = buildDomainCards(
		catalog.DomainCards,
		creation.Profile.ClassID,
		classDomainsByID,
		domainNameByID,
	)
	return creation
}

// sortByName centralizes this web behavior in one helper seam.
func sortByName[T any](items []T, nameOf func(T) string, idOf func(T) string) {
	sort.SliceStable(items, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(nameOf(items[i])))
		right := strings.ToLower(strings.TrimSpace(nameOf(items[j])))
		if left == right {
			return strings.TrimSpace(idOf(items[i])) < strings.TrimSpace(idOf(items[j]))
		}
		return left < right
	})
}
