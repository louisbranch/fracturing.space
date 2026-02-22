package admin

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
)

type catalogSectionTableLoader func(
	context.Context,
	daggerheartv1.DaggerheartContentServiceClient,
	string,
	commonv1.Locale,
) ([]templates.CatalogTableRow, string, string, error)

type catalogSectionDetailLoader func(
	context.Context,
	daggerheartv1.DaggerheartContentServiceClient,
	string,
	string,
	commonv1.Locale,
	*message.Printer,
) templates.CatalogDetailView

var catalogSectionTableLoaders = map[string]catalogSectionTableLoader{
	templates.CatalogSectionClasses: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, pageToken string, locale commonv1.Locale) ([]templates.CatalogTableRow, string, string, error) {
		resp, err := contentClient.ListClasses(ctx, &daggerheartv1.ListDaggerheartClassesRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			return nil, "", "", err
		}
		if resp == nil {
			return nil, "", "", nil
		}
		return buildCatalogClassRows(resp.GetClasses()), resp.GetNextPageToken(), resp.GetPreviousPageToken(), nil
	},
	templates.CatalogSectionSubclasses: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, pageToken string, locale commonv1.Locale) ([]templates.CatalogTableRow, string, string, error) {
		resp, err := contentClient.ListSubclasses(ctx, &daggerheartv1.ListDaggerheartSubclassesRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			return nil, "", "", err
		}
		if resp == nil {
			return nil, "", "", nil
		}
		return buildCatalogSubclassRows(resp.GetSubclasses()), resp.GetNextPageToken(), resp.GetPreviousPageToken(), nil
	},
	templates.CatalogSectionHeritages: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, pageToken string, locale commonv1.Locale) ([]templates.CatalogTableRow, string, string, error) {
		resp, err := contentClient.ListHeritages(ctx, &daggerheartv1.ListDaggerheartHeritagesRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			return nil, "", "", err
		}
		if resp == nil {
			return nil, "", "", nil
		}
		return buildCatalogHeritageRows(resp.GetHeritages()), resp.GetNextPageToken(), resp.GetPreviousPageToken(), nil
	},
	templates.CatalogSectionExperiences: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, pageToken string, locale commonv1.Locale) ([]templates.CatalogTableRow, string, string, error) {
		resp, err := contentClient.ListExperiences(ctx, &daggerheartv1.ListDaggerheartExperiencesRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			return nil, "", "", err
		}
		if resp == nil {
			return nil, "", "", nil
		}
		return buildCatalogExperienceRows(resp.GetExperiences()), resp.GetNextPageToken(), resp.GetPreviousPageToken(), nil
	},
	templates.CatalogSectionDomains: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, pageToken string, locale commonv1.Locale) ([]templates.CatalogTableRow, string, string, error) {
		resp, err := contentClient.ListDomains(ctx, &daggerheartv1.ListDaggerheartDomainsRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			return nil, "", "", err
		}
		if resp == nil {
			return nil, "", "", nil
		}
		return buildCatalogDomainRows(resp.GetDomains()), resp.GetNextPageToken(), resp.GetPreviousPageToken(), nil
	},
	templates.CatalogSectionDomainCards: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, pageToken string, locale commonv1.Locale) ([]templates.CatalogTableRow, string, string, error) {
		resp, err := contentClient.ListDomainCards(ctx, &daggerheartv1.ListDaggerheartDomainCardsRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "level",
			Locale:    locale,
		})
		if err != nil {
			return nil, "", "", err
		}
		if resp == nil {
			return nil, "", "", nil
		}
		return buildCatalogDomainCardRows(resp.GetDomainCards()), resp.GetNextPageToken(), resp.GetPreviousPageToken(), nil
	},
	templates.CatalogSectionItems: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, pageToken string, locale commonv1.Locale) ([]templates.CatalogTableRow, string, string, error) {
		resp, err := contentClient.ListItems(ctx, &daggerheartv1.ListDaggerheartItemsRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			return nil, "", "", err
		}
		if resp == nil {
			return nil, "", "", nil
		}
		return buildCatalogItemRows(resp.GetItems()), resp.GetNextPageToken(), resp.GetPreviousPageToken(), nil
	},
	templates.CatalogSectionWeapons: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, pageToken string, locale commonv1.Locale) ([]templates.CatalogTableRow, string, string, error) {
		resp, err := contentClient.ListWeapons(ctx, &daggerheartv1.ListDaggerheartWeaponsRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			return nil, "", "", err
		}
		if resp == nil {
			return nil, "", "", nil
		}
		return buildCatalogWeaponRows(resp.GetWeapons()), resp.GetNextPageToken(), resp.GetPreviousPageToken(), nil
	},
	templates.CatalogSectionArmor: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, pageToken string, locale commonv1.Locale) ([]templates.CatalogTableRow, string, string, error) {
		resp, err := contentClient.ListArmor(ctx, &daggerheartv1.ListDaggerheartArmorRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			return nil, "", "", err
		}
		if resp == nil {
			return nil, "", "", nil
		}
		return buildCatalogArmorRows(resp.GetArmor()), resp.GetNextPageToken(), resp.GetPreviousPageToken(), nil
	},
	templates.CatalogSectionLoot: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, pageToken string, locale commonv1.Locale) ([]templates.CatalogTableRow, string, string, error) {
		resp, err := contentClient.ListLootEntries(ctx, &daggerheartv1.ListDaggerheartLootEntriesRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "roll",
			Locale:    locale,
		})
		if err != nil {
			return nil, "", "", err
		}
		if resp == nil {
			return nil, "", "", nil
		}
		return buildCatalogLootRows(resp.GetEntries()), resp.GetNextPageToken(), resp.GetPreviousPageToken(), nil
	},
	templates.CatalogSectionDamageTypes: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, pageToken string, locale commonv1.Locale) ([]templates.CatalogTableRow, string, string, error) {
		resp, err := contentClient.ListDamageTypes(ctx, &daggerheartv1.ListDaggerheartDamageTypesRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			return nil, "", "", err
		}
		if resp == nil {
			return nil, "", "", nil
		}
		return buildCatalogDamageTypeRows(resp.GetDamageTypes()), resp.GetNextPageToken(), resp.GetPreviousPageToken(), nil
	},
	templates.CatalogSectionAdversaries: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, pageToken string, locale commonv1.Locale) ([]templates.CatalogTableRow, string, string, error) {
		resp, err := contentClient.ListAdversaries(ctx, &daggerheartv1.ListDaggerheartAdversariesRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			return nil, "", "", err
		}
		if resp == nil {
			return nil, "", "", nil
		}
		return buildCatalogAdversaryRows(resp.GetAdversaries()), resp.GetNextPageToken(), resp.GetPreviousPageToken(), nil
	},
	templates.CatalogSectionBeastforms: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, pageToken string, locale commonv1.Locale) ([]templates.CatalogTableRow, string, string, error) {
		resp, err := contentClient.ListBeastforms(ctx, &daggerheartv1.ListDaggerheartBeastformsRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			return nil, "", "", err
		}
		if resp == nil {
			return nil, "", "", nil
		}
		return buildCatalogBeastformRows(resp.GetBeastforms()), resp.GetNextPageToken(), resp.GetPreviousPageToken(), nil
	},
	templates.CatalogSectionCompanionExperiences: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, pageToken string, locale commonv1.Locale) ([]templates.CatalogTableRow, string, string, error) {
		resp, err := contentClient.ListCompanionExperiences(ctx, &daggerheartv1.ListDaggerheartCompanionExperiencesRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			return nil, "", "", err
		}
		if resp == nil {
			return nil, "", "", nil
		}
		return buildCatalogCompanionExperienceRows(resp.GetExperiences()), resp.GetNextPageToken(), resp.GetPreviousPageToken(), nil
	},
	templates.CatalogSectionEnvironments: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, pageToken string, locale commonv1.Locale) ([]templates.CatalogTableRow, string, string, error) {
		resp, err := contentClient.ListEnvironments(ctx, &daggerheartv1.ListDaggerheartEnvironmentsRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			return nil, "", "", err
		}
		if resp == nil {
			return nil, "", "", nil
		}
		return buildCatalogEnvironmentRows(resp.GetEnvironments()), resp.GetNextPageToken(), resp.GetPreviousPageToken(), nil
	},
}

var catalogSectionDetailLoaders = map[string]catalogSectionDetailLoader{
	templates.CatalogSectionClasses: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, sectionID, entryID string, locale commonv1.Locale, loc *message.Printer) templates.CatalogDetailView {
		resp, err := contentClient.GetClass(ctx, &daggerheartv1.GetDaggerheartClassRequest{Id: entryID, Locale: locale})
		return buildCatalogClassDetail(sectionID, entryID, resp.GetClass(), err, loc)
	},
	templates.CatalogSectionSubclasses: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, sectionID, entryID string, locale commonv1.Locale, loc *message.Printer) templates.CatalogDetailView {
		resp, err := contentClient.GetSubclass(ctx, &daggerheartv1.GetDaggerheartSubclassRequest{Id: entryID, Locale: locale})
		return buildCatalogSubclassDetail(sectionID, entryID, resp.GetSubclass(), err, loc)
	},
	templates.CatalogSectionHeritages: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, sectionID, entryID string, locale commonv1.Locale, loc *message.Printer) templates.CatalogDetailView {
		resp, err := contentClient.GetHeritage(ctx, &daggerheartv1.GetDaggerheartHeritageRequest{Id: entryID, Locale: locale})
		return buildCatalogHeritageDetail(sectionID, entryID, resp.GetHeritage(), err, loc)
	},
	templates.CatalogSectionExperiences: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, sectionID, entryID string, locale commonv1.Locale, loc *message.Printer) templates.CatalogDetailView {
		resp, err := contentClient.GetExperience(ctx, &daggerheartv1.GetDaggerheartExperienceRequest{Id: entryID, Locale: locale})
		return buildCatalogExperienceDetail(sectionID, entryID, resp.GetExperience(), err, loc)
	},
	templates.CatalogSectionDomains: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, sectionID, entryID string, locale commonv1.Locale, loc *message.Printer) templates.CatalogDetailView {
		resp, err := contentClient.GetDomain(ctx, &daggerheartv1.GetDaggerheartDomainRequest{Id: entryID, Locale: locale})
		return buildCatalogDomainDetail(sectionID, entryID, resp.GetDomain(), err, loc)
	},
	templates.CatalogSectionDomainCards: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, sectionID, entryID string, locale commonv1.Locale, loc *message.Printer) templates.CatalogDetailView {
		resp, err := contentClient.GetDomainCard(ctx, &daggerheartv1.GetDaggerheartDomainCardRequest{Id: entryID, Locale: locale})
		return buildCatalogDomainCardDetail(sectionID, entryID, resp.GetDomainCard(), err, loc)
	},
	templates.CatalogSectionItems: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, sectionID, entryID string, locale commonv1.Locale, loc *message.Printer) templates.CatalogDetailView {
		resp, err := contentClient.GetItem(ctx, &daggerheartv1.GetDaggerheartItemRequest{Id: entryID, Locale: locale})
		return buildCatalogItemDetail(sectionID, entryID, resp.GetItem(), err, loc)
	},
	templates.CatalogSectionWeapons: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, sectionID, entryID string, locale commonv1.Locale, loc *message.Printer) templates.CatalogDetailView {
		resp, err := contentClient.GetWeapon(ctx, &daggerheartv1.GetDaggerheartWeaponRequest{Id: entryID, Locale: locale})
		return buildCatalogWeaponDetail(sectionID, entryID, resp.GetWeapon(), err, loc)
	},
	templates.CatalogSectionArmor: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, sectionID, entryID string, locale commonv1.Locale, loc *message.Printer) templates.CatalogDetailView {
		resp, err := contentClient.GetArmor(ctx, &daggerheartv1.GetDaggerheartArmorRequest{Id: entryID, Locale: locale})
		return buildCatalogArmorDetail(sectionID, entryID, resp.GetArmor(), err, loc)
	},
	templates.CatalogSectionLoot: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, sectionID, entryID string, locale commonv1.Locale, loc *message.Printer) templates.CatalogDetailView {
		resp, err := contentClient.GetLootEntry(ctx, &daggerheartv1.GetDaggerheartLootEntryRequest{Id: entryID, Locale: locale})
		return buildCatalogLootDetail(sectionID, entryID, resp.GetEntry(), err, loc)
	},
	templates.CatalogSectionDamageTypes: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, sectionID, entryID string, locale commonv1.Locale, loc *message.Printer) templates.CatalogDetailView {
		resp, err := contentClient.GetDamageType(ctx, &daggerheartv1.GetDaggerheartDamageTypeRequest{Id: entryID, Locale: locale})
		return buildCatalogDamageTypeDetail(sectionID, entryID, resp.GetDamageType(), err, loc)
	},
	templates.CatalogSectionAdversaries: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, sectionID, entryID string, locale commonv1.Locale, loc *message.Printer) templates.CatalogDetailView {
		resp, err := contentClient.GetAdversary(ctx, &daggerheartv1.GetDaggerheartAdversaryRequest{Id: entryID, Locale: locale})
		return buildCatalogAdversaryDetail(sectionID, entryID, resp.GetAdversary(), err, loc)
	},
	templates.CatalogSectionBeastforms: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, sectionID, entryID string, locale commonv1.Locale, loc *message.Printer) templates.CatalogDetailView {
		resp, err := contentClient.GetBeastform(ctx, &daggerheartv1.GetDaggerheartBeastformRequest{Id: entryID, Locale: locale})
		return buildCatalogBeastformDetail(sectionID, entryID, resp.GetBeastform(), err, loc)
	},
	templates.CatalogSectionCompanionExperiences: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, sectionID, entryID string, locale commonv1.Locale, loc *message.Printer) templates.CatalogDetailView {
		resp, err := contentClient.GetCompanionExperience(ctx, &daggerheartv1.GetDaggerheartCompanionExperienceRequest{Id: entryID, Locale: locale})
		return buildCatalogCompanionExperienceDetail(sectionID, entryID, resp.GetExperience(), err, loc)
	},
	templates.CatalogSectionEnvironments: func(ctx context.Context, contentClient daggerheartv1.DaggerheartContentServiceClient, sectionID, entryID string, locale commonv1.Locale, loc *message.Printer) templates.CatalogDetailView {
		resp, err := contentClient.GetEnvironment(ctx, &daggerheartv1.GetDaggerheartEnvironmentRequest{Id: entryID, Locale: locale})
		return buildCatalogEnvironmentDetail(sectionID, entryID, resp.GetEnvironment(), err, loc)
	},
}
