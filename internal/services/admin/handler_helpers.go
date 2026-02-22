package admin

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	platformicons "github.com/louisbranch/fracturing.space/internal/platform/icons"
	routepath "github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func (h *Handler) renderUserDetail(w http.ResponseWriter, r *http.Request, view templates.UserDetailPageView, pageCtx templates.PageContext, loc *message.Printer, activePage string) {
	renderPage(
		w,
		r,
		templates.UserDetailPage(view, activePage, loc),
		templates.UserDetailFullPage(view, activePage, pageCtx),
		htmxLocalizedPageTitle(loc, "title.user", templates.AppName()),
	)
}

func (h *Handler) redirectToUserDetail(w http.ResponseWriter, r *http.Request, userID string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		http.NotFound(w, r)
		return
	}
	redirectURL := routepath.UserDetail(userID)
	if isHTMXRequest(r) {
		w.Header().Set("Location", redirectURL)
		w.Header().Set("HX-Redirect", redirectURL)
		w.WriteHeader(http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (h *Handler) loadUserDetail(ctx context.Context, userID string, loc *message.Printer) (*templates.UserDetail, string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, loc.Sprintf("error.user_id_required")
	}
	client := h.authClient()
	if client == nil {
		return nil, loc.Sprintf("error.user_service_unavailable")
	}
	response, err := client.GetUser(ctx, &authv1.GetUserRequest{UserId: userID})
	if err != nil || response.GetUser() == nil {
		log.Printf("get user: %v", err)
		return nil, loc.Sprintf("error.user_not_found")
	}
	detail := buildUserDetail(response.GetUser())
	if detail != nil {
		emails, err := client.ListUserEmails(ctx, &authv1.ListUserEmailsRequest{UserId: userID})
		if err != nil {
			log.Printf("list user emails: %v", err)
		} else {
			detail.Emails = buildUserEmailRows(emails.GetEmails(), loc)
		}
	}
	return detail, ""
}

// buildCampaignRows formats campaign rows for the table.
func buildCampaignRows(campaigns []*statev1.Campaign, loc *message.Printer) []templates.CampaignRow {
	rows := make([]templates.CampaignRow, 0, len(campaigns))
	for _, campaign := range campaigns {
		if campaign == nil {
			continue
		}
		rows = append(rows, templates.CampaignRow{
			ID:               campaign.GetId(),
			Name:             campaign.GetName(),
			System:           formatGameSystem(campaign.GetSystem(), loc),
			GMMode:           formatGmMode(campaign.GetGmMode(), loc),
			ParticipantCount: strconv.FormatInt(int64(campaign.GetParticipantCount()), 10),
			CharacterCount:   strconv.FormatInt(int64(campaign.GetCharacterCount()), 10),
			ThemePrompt:      truncateText(campaign.GetThemePrompt(), campaignThemePromptLimit),
			CreatedDate:      formatCreatedDate(campaign.GetCreatedAt()),
		})
	}
	return rows
}

// buildSystemRows formats system rows for the systems table.
func buildSystemRows(systemsList []*statev1.GameSystemInfo, loc *message.Printer) []templates.SystemRow {
	rows := make([]templates.SystemRow, 0, len(systemsList))
	for _, system := range systemsList {
		if system == nil {
			continue
		}
		detailURL := routepath.System(system.GetId().String())
		version := strings.TrimSpace(system.GetVersion())
		if version != "" {
			detailURL = detailURL + "?version=" + url.QueryEscape(version)
		}
		rows = append(rows, templates.SystemRow{
			Name:                system.GetName(),
			Version:             version,
			ImplementationStage: formatImplementationStage(system.GetImplementationStage(), loc),
			OperationalStatus:   formatOperationalStatus(system.GetOperationalStatus(), loc),
			AccessLevel:         formatAccessLevel(system.GetAccessLevel(), loc),
			IsDefault:           system.GetIsDefault(),
			DetailURL:           detailURL,
		})
	}
	return rows
}

// buildIconRows formats icon catalog rows for the icons table.
func buildIconRows(definitions []platformicons.Definition) []templates.IconRow {
	rows := make([]templates.IconRow, 0, len(definitions))
	for _, def := range definitions {
		rows = append(rows, templates.IconRow{
			ID:          def.ID,
			Name:        def.Name,
			Description: def.Description,
			LucideName:  platformicons.LucideNameOrDefault(def.ID),
		})
	}
	return rows
}

// buildCampaignDetail formats a campaign into detail view data.
func buildCampaignDetail(campaign *statev1.Campaign, loc *message.Printer) templates.CampaignDetail {
	if campaign == nil {
		return templates.CampaignDetail{}
	}
	return templates.CampaignDetail{
		ID:               campaign.GetId(),
		Name:             campaign.GetName(),
		System:           formatGameSystem(campaign.GetSystem(), loc),
		GMMode:           formatGmMode(campaign.GetGmMode(), loc),
		ParticipantCount: strconv.FormatInt(int64(campaign.GetParticipantCount()), 10),
		CharacterCount:   strconv.FormatInt(int64(campaign.GetCharacterCount()), 10),
		ThemePrompt:      campaign.GetThemePrompt(),
		CreatedAt:        formatTimestamp(campaign.GetCreatedAt()),
		UpdatedAt:        formatTimestamp(campaign.GetUpdatedAt()),
	}
}

// buildSystemDetail formats a system into detail view data.
func buildSystemDetail(system *statev1.GameSystemInfo, loc *message.Printer) templates.SystemDetail {
	if system == nil {
		return templates.SystemDetail{}
	}
	return templates.SystemDetail{
		ID:                  system.GetId().String(),
		Name:                system.GetName(),
		Version:             system.GetVersion(),
		ImplementationStage: formatImplementationStage(system.GetImplementationStage(), loc),
		OperationalStatus:   formatOperationalStatus(system.GetOperationalStatus(), loc),
		AccessLevel:         formatAccessLevel(system.GetAccessLevel(), loc),
		IsDefault:           system.GetIsDefault(),
	}
}

// buildCampaignSessionRows formats session rows for the detail view.
func buildCampaignSessionRows(sessions []*statev1.Session, loc *message.Printer) []templates.CampaignSessionRow {
	rows := make([]templates.CampaignSessionRow, 0, len(sessions))
	for _, session := range sessions {
		if session == nil {
			continue
		}
		statusBadge := "secondary"
		if session.GetStatus() == statev1.SessionStatus_SESSION_ACTIVE {
			statusBadge = "success"
		}
		row := templates.CampaignSessionRow{
			ID:          session.GetId(),
			CampaignID:  session.GetCampaignId(),
			Name:        session.GetName(),
			Status:      formatSessionStatus(session.GetStatus(), loc),
			StatusBadge: statusBadge,
			StartedAt:   formatTimestamp(session.GetStartedAt()),
		}
		if session.GetEndedAt() != nil {
			row.EndedAt = formatTimestamp(session.GetEndedAt())
		}
		rows = append(rows, row)
	}
	return rows
}

// buildUserRows formats user rows for the table.
func buildUserRows(users []*authv1.User) []templates.UserRow {
	rows := make([]templates.UserRow, 0, len(users))
	for _, u := range users {
		if u == nil {
			continue
		}
		rows = append(rows, templates.UserRow{
			ID:        u.GetId(),
			Email:     u.GetEmail(),
			CreatedAt: formatTimestamp(u.GetCreatedAt()),
			UpdatedAt: formatTimestamp(u.GetUpdatedAt()),
		})
	}
	return rows
}

// catalogSectionColumns defines which columns to show per catalog section.
var catalogSectionColumnKeys = map[string][]string{
	templates.CatalogSectionClasses:              {"catalog.table.starting_hp", "catalog.table.starting_evasion"},
	templates.CatalogSectionSubclasses:           {"catalog.table.spellcast_trait", "catalog.table.feature_count"},
	templates.CatalogSectionHeritages:            {"catalog.table.kind", "catalog.table.feature_count"},
	templates.CatalogSectionExperiences:          {"catalog.table.description"},
	templates.CatalogSectionDomains:              {"catalog.table.description"},
	templates.CatalogSectionDomainCards:          {"catalog.table.domain", "catalog.table.level", "catalog.table.type"},
	templates.CatalogSectionItems:                {"catalog.table.rarity", "catalog.table.kind", "catalog.table.stack_max"},
	templates.CatalogSectionWeapons:              {"catalog.table.category", "catalog.table.tier", "catalog.table.damage_type"},
	templates.CatalogSectionArmor:                {"catalog.table.tier", "catalog.table.armor_score"},
	templates.CatalogSectionLoot:                 {"catalog.table.roll", "catalog.table.description"},
	templates.CatalogSectionDamageTypes:          {"catalog.table.description"},
	templates.CatalogSectionAdversaries:          {"catalog.table.tier", "catalog.table.role"},
	templates.CatalogSectionBeastforms:           {"catalog.table.tier", "catalog.table.trait"},
	templates.CatalogSectionCompanionExperiences: {"catalog.table.description"},
	templates.CatalogSectionEnvironments:         {"catalog.table.type", "catalog.table.difficulty"},
}

func catalogSectionColumns(sectionID string, loc *message.Printer) []string {
	keys, ok := catalogSectionColumnKeys[sectionID]
	if !ok {
		return nil
	}
	columns := make([]string, 0, len(keys))
	for _, key := range keys {
		columns = append(columns, loc.Sprintf(key))
	}
	return columns
}

// buildCatalogClassRows formats class entries for tables.
func buildCatalogClassRows(classes []*daggerheartv1.DaggerheartClass) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(classes))
	for _, entry := range classes {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionClasses, entry.GetId(), entry.GetName(), []string{
			strconv.Itoa(int(entry.GetStartingHp())),
			strconv.Itoa(int(entry.GetStartingEvasion())),
		}))
	}
	return rows
}

// buildCatalogSubclassRows formats subclass entries for tables.
func buildCatalogSubclassRows(subclasses []*daggerheartv1.DaggerheartSubclass) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(subclasses))
	for _, entry := range subclasses {
		if entry == nil {
			continue
		}
		featureCount := len(entry.GetFoundationFeatures()) + len(entry.GetSpecializationFeatures()) + len(entry.GetMasteryFeatures())
		rows = append(rows, catalogRow(templates.CatalogSectionSubclasses, entry.GetId(), entry.GetName(), []string{
			entry.GetSpellcastTrait(),
			strconv.Itoa(featureCount),
		}))
	}
	return rows
}

// buildCatalogHeritageRows formats heritage entries for tables.
func buildCatalogHeritageRows(heritages []*daggerheartv1.DaggerheartHeritage) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(heritages))
	for _, entry := range heritages {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionHeritages, entry.GetId(), entry.GetName(), []string{
			formatEnumValue(entry.GetKind().String()),
			strconv.Itoa(len(entry.GetFeatures())),
		}))
	}
	return rows
}

// buildCatalogExperienceRows formats experience entries for tables.
func buildCatalogExperienceRows(entries []*daggerheartv1.DaggerheartExperienceEntry) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionExperiences, entry.GetId(), entry.GetName(), []string{
			truncateText(entry.GetDescription(), catalogDescriptionLimit),
		}))
	}
	return rows
}

// buildCatalogDomainRows formats domain entries for tables.
func buildCatalogDomainRows(domains []*daggerheartv1.DaggerheartDomain) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(domains))
	for _, entry := range domains {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionDomains, entry.GetId(), entry.GetName(), []string{
			truncateText(entry.GetDescription(), catalogDescriptionLimit),
		}))
	}
	return rows
}

// buildCatalogDomainCardRows formats domain card entries for tables.
func buildCatalogDomainCardRows(cards []*daggerheartv1.DaggerheartDomainCard) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(cards))
	for _, entry := range cards {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionDomainCards, entry.GetId(), entry.GetName(), []string{
			entry.GetDomainId(),
			strconv.Itoa(int(entry.GetLevel())),
			formatEnumValue(entry.GetType().String()),
		}))
	}
	return rows
}

// buildCatalogItemRows formats item entries for tables.
func buildCatalogItemRows(items []*daggerheartv1.DaggerheartItem) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(items))
	for _, entry := range items {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionItems, entry.GetId(), entry.GetName(), []string{
			formatEnumValue(entry.GetRarity().String()),
			formatEnumValue(entry.GetKind().String()),
			strconv.Itoa(int(entry.GetStackMax())),
		}))
	}
	return rows
}

// buildCatalogWeaponRows formats weapon entries for tables.
func buildCatalogWeaponRows(weapons []*daggerheartv1.DaggerheartWeapon) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(weapons))
	for _, entry := range weapons {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionWeapons, entry.GetId(), entry.GetName(), []string{
			formatEnumValue(entry.GetCategory().String()),
			strconv.Itoa(int(entry.GetTier())),
			formatEnumValue(entry.GetDamageType().String()),
		}))
	}
	return rows
}

// buildCatalogArmorRows formats armor entries for tables.
func buildCatalogArmorRows(armor []*daggerheartv1.DaggerheartArmor) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(armor))
	for _, entry := range armor {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionArmor, entry.GetId(), entry.GetName(), []string{
			strconv.Itoa(int(entry.GetTier())),
			strconv.Itoa(int(entry.GetArmorScore())),
		}))
	}
	return rows
}

// buildCatalogLootRows formats loot entries for tables.
func buildCatalogLootRows(entries []*daggerheartv1.DaggerheartLootEntry) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionLoot, entry.GetId(), entry.GetName(), []string{
			strconv.Itoa(int(entry.GetRoll())),
			truncateText(entry.GetDescription(), catalogDescriptionLimit),
		}))
	}
	return rows
}

// buildCatalogDamageTypeRows formats damage type entries for tables.
func buildCatalogDamageTypeRows(entries []*daggerheartv1.DaggerheartDamageTypeEntry) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionDamageTypes, entry.GetId(), entry.GetName(), []string{
			truncateText(entry.GetDescription(), catalogDescriptionLimit),
		}))
	}
	return rows
}

// buildCatalogAdversaryRows formats adversary entries for tables.
func buildCatalogAdversaryRows(entries []*daggerheartv1.DaggerheartAdversaryEntry) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionAdversaries, entry.GetId(), entry.GetName(), []string{
			strconv.Itoa(int(entry.GetTier())),
			entry.GetRole(),
		}))
	}
	return rows
}

// buildCatalogBeastformRows formats beastform entries for tables.
func buildCatalogBeastformRows(entries []*daggerheartv1.DaggerheartBeastformEntry) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionBeastforms, entry.GetId(), entry.GetName(), []string{
			strconv.Itoa(int(entry.GetTier())),
			entry.GetTrait(),
		}))
	}
	return rows
}

// buildCatalogCompanionExperienceRows formats companion experience entries for tables.
func buildCatalogCompanionExperienceRows(entries []*daggerheartv1.DaggerheartCompanionExperienceEntry) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionCompanionExperiences, entry.GetId(), entry.GetName(), []string{
			truncateText(entry.GetDescription(), catalogDescriptionLimit),
		}))
	}
	return rows
}

// buildCatalogEnvironmentRows formats environment entries for tables.
func buildCatalogEnvironmentRows(entries []*daggerheartv1.DaggerheartEnvironment) []templates.CatalogTableRow {
	rows := make([]templates.CatalogTableRow, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		rows = append(rows, catalogRow(templates.CatalogSectionEnvironments, entry.GetId(), entry.GetName(), []string{
			formatEnumValue(entry.GetType().String()),
			strconv.Itoa(int(entry.GetDifficulty())),
		}))
	}
	return rows
}

// buildCatalogClassDetail formats class details for the catalog panel.
func buildCatalogClassDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartClass, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get class: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.starting_hp"), Value: strconv.Itoa(int(entry.GetStartingHp()))},
		{Label: loc.Sprintf("catalog.table.starting_evasion"), Value: strconv.Itoa(int(entry.GetStartingEvasion()))},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogSubclassDetail formats subclass details for the catalog panel.
func buildCatalogSubclassDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartSubclass, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get subclass: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	featureCount := len(entry.GetFoundationFeatures()) + len(entry.GetSpecializationFeatures()) + len(entry.GetMasteryFeatures())
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.spellcast_trait"), Value: entry.GetSpellcastTrait()},
		{Label: loc.Sprintf("catalog.table.feature_count"), Value: strconv.Itoa(featureCount)},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogHeritageDetail formats heritage details for the catalog panel.
func buildCatalogHeritageDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartHeritage, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get heritage: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.kind"), Value: formatEnumValue(entry.GetKind().String())},
		{Label: loc.Sprintf("catalog.table.feature_count"), Value: strconv.Itoa(len(entry.GetFeatures()))},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogExperienceDetail formats experience details for the catalog panel.
func buildCatalogExperienceDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartExperienceEntry, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get experience: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.description"), Value: entry.GetDescription()},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogDomainDetail formats domain details for the catalog panel.
func buildCatalogDomainDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartDomain, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get domain: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.description"), Value: entry.GetDescription()},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogDomainCardDetail formats domain card details for the catalog panel.
func buildCatalogDomainCardDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartDomainCard, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get domain card: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.domain"), Value: entry.GetDomainId()},
		{Label: loc.Sprintf("catalog.table.level"), Value: strconv.Itoa(int(entry.GetLevel()))},
		{Label: loc.Sprintf("catalog.table.type"), Value: formatEnumValue(entry.GetType().String())},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogItemDetail formats item details for the catalog panel.
func buildCatalogItemDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartItem, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get item: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.rarity"), Value: formatEnumValue(entry.GetRarity().String())},
		{Label: loc.Sprintf("catalog.table.kind"), Value: formatEnumValue(entry.GetKind().String())},
		{Label: loc.Sprintf("catalog.table.stack_max"), Value: strconv.Itoa(int(entry.GetStackMax()))},
		{Label: loc.Sprintf("catalog.table.description"), Value: entry.GetDescription()},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogWeaponDetail formats weapon details for the catalog panel.
func buildCatalogWeaponDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartWeapon, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get weapon: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.category"), Value: formatEnumValue(entry.GetCategory().String())},
		{Label: loc.Sprintf("catalog.table.tier"), Value: strconv.Itoa(int(entry.GetTier()))},
		{Label: loc.Sprintf("catalog.table.damage_type"), Value: formatEnumValue(entry.GetDamageType().String())},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogArmorDetail formats armor details for the catalog panel.
func buildCatalogArmorDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartArmor, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get armor: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.tier"), Value: strconv.Itoa(int(entry.GetTier()))},
		{Label: loc.Sprintf("catalog.table.armor_score"), Value: strconv.Itoa(int(entry.GetArmorScore()))},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogLootDetail formats loot details for the catalog panel.
func buildCatalogLootDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartLootEntry, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get loot entry: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.roll"), Value: strconv.Itoa(int(entry.GetRoll()))},
		{Label: loc.Sprintf("catalog.table.description"), Value: entry.GetDescription()},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogDamageTypeDetail formats damage type details for the catalog panel.
func buildCatalogDamageTypeDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartDamageTypeEntry, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get damage type: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.description"), Value: entry.GetDescription()},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogAdversaryDetail formats adversary details for the catalog panel.
func buildCatalogAdversaryDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartAdversaryEntry, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get adversary: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.tier"), Value: strconv.Itoa(int(entry.GetTier()))},
		{Label: loc.Sprintf("catalog.table.role"), Value: entry.GetRole()},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogBeastformDetail formats beastform details for the catalog panel.
func buildCatalogBeastformDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartBeastformEntry, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get beastform: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.tier"), Value: strconv.Itoa(int(entry.GetTier()))},
		{Label: loc.Sprintf("catalog.table.trait"), Value: entry.GetTrait()},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogCompanionExperienceDetail formats companion experience details for the catalog panel.
func buildCatalogCompanionExperienceDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartCompanionExperienceEntry, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get companion experience: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.description"), Value: entry.GetDescription()},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// buildCatalogEnvironmentDetail formats environment details for the catalog panel.
func buildCatalogEnvironmentDetail(sectionID, entryID string, entry *daggerheartv1.DaggerheartEnvironment, err error, loc *message.Printer) templates.CatalogDetailView {
	if err != nil {
		log.Printf("get environment: %v", err)
	}
	message := catalogDetailErrorMessage(err, loc, entry == nil)
	if entry == nil {
		return catalogDetailView(sectionID, entryID, "", nil, nil, message, loc)
	}
	fields := []templates.CatalogDetailField{
		{Label: loc.Sprintf("catalog.detail.id"), Value: entry.GetId()},
		{Label: loc.Sprintf("catalog.table.name"), Value: entry.GetName()},
		{Label: loc.Sprintf("catalog.table.type"), Value: formatEnumValue(entry.GetType().String())},
		{Label: loc.Sprintf("catalog.table.difficulty"), Value: strconv.Itoa(int(entry.GetDifficulty()))},
	}
	return catalogDetailView(sectionID, entry.GetId(), catalogPrimaryLabel(entry.GetName(), entry.GetId()), fields, entry, message, loc)
}

// catalogRow builds a table row with a detail link for a section.
func catalogRow(sectionID, entryID, name string, cells []string) templates.CatalogTableRow {
	return templates.CatalogTableRow{
		Primary:   catalogPrimaryLabel(name, entryID),
		DetailURL: routepath.CatalogEntry("daggerheart", sectionID, entryID),
		Cells:     cells,
	}
}

// catalogPrimaryLabel picks a user-facing label for a row.
func catalogPrimaryLabel(name, entryID string) string {
	if strings.TrimSpace(name) == "" {
		return entryID
	}
	return name
}

// catalogDetailErrorMessage normalizes not-found vs. unavailable states.
func catalogDetailErrorMessage(err error, loc *message.Printer, missing bool) string {
	if !missing {
		return ""
	}
	if err == nil {
		return loc.Sprintf("catalog.error.not_found")
	}
	if status.Code(err) == codes.NotFound {
		return loc.Sprintf("catalog.error.not_found")
	}
	return loc.Sprintf("catalog.error.entry_unavailable")
}

// catalogDetailView builds the shared detail view model with raw JSON.
func catalogDetailView(sectionID, entryID, title string, fields []templates.CatalogDetailField, raw proto.Message, message string, loc *message.Printer) templates.CatalogDetailView {
	if title == "" {
		title = templates.DaggerheartCatalogSectionLabel(loc, sectionID)
	}
	return templates.CatalogDetailView{
		SectionID: sectionID,
		Title:     title,
		ID:        entryID,
		Fields:    fields,
		Message:   message,
		RawJSON:   formatProtoJSON(raw),
		BackURL:   routepath.CatalogSection("daggerheart", sectionID),
	}
}

// formatProtoJSON renders raw proto data for the detail panel.
func formatProtoJSON(message proto.Message) string {
	if message == nil {
		return ""
	}
	data, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(message)
	if err != nil {
		return ""
	}
	return string(data)
}

// formatEnumValue normalizes enum strings to title case labels.
func formatEnumValue(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.Split(value, "_")
	label := parts[len(parts)-1]
	if label == "UNSPECIFIED" {
		return ""
	}
	label = strings.ToLower(label)
	return strings.ToUpper(label[:1]) + label[1:]
}

// buildUserDetail formats a user detail view.
func buildUserDetail(u *authv1.User) *templates.UserDetail {
	if u == nil {
		return nil
	}
	return &templates.UserDetail{
		ID:        u.GetId(),
		Email:     u.GetEmail(),
		CreatedAt: formatTimestamp(u.GetCreatedAt()),
		UpdatedAt: formatTimestamp(u.GetUpdatedAt()),
	}
}

func buildUserEmailRows(emails []*authv1.UserEmail, loc *message.Printer) []templates.UserEmailRow {
	rows := make([]templates.UserEmailRow, 0, len(emails))
	for _, email := range emails {
		if email == nil {
			continue
		}
		verified := "-"
		if email.GetVerifiedAt() != nil {
			verified = formatTimestamp(email.GetVerifiedAt())
		}
		rows = append(rows, templates.UserEmailRow{
			Email:      email.GetEmail(),
			VerifiedAt: verified,
			CreatedAt:  formatTimestamp(email.GetCreatedAt()),
			UpdatedAt:  formatTimestamp(email.GetUpdatedAt()),
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return rows
}

func (h *Handler) populateUserInvites(ctx context.Context, detail *templates.UserDetail, loc *message.Printer) {
	if detail == nil {
		return
	}
	rows, message := h.listPendingInvitesForUser(ctx, detail.ID, loc)
	detail.PendingInvites = rows
	detail.PendingInvitesMessage = message
}

func (h *Handler) listPendingInvitesForUser(ctx context.Context, userID string, loc *message.Printer) ([]templates.InviteRow, string) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, loc.Sprintf("users.invites.empty")
	}
	inviteClient := h.inviteClient()
	if inviteClient == nil {
		return nil, loc.Sprintf("error.pending_invites_unavailable")
	}

	rows := make([]templates.InviteRow, 0)
	pageToken := ""
	for {
		resp, err := inviteClient.ListPendingInvitesForUser(ctx, &statev1.ListPendingInvitesForUserRequest{
			PageSize:  inviteListPageSize,
			PageToken: pageToken,
		})
		if err != nil {
			log.Printf("list pending invites for user: %v", err)
			return nil, loc.Sprintf("error.pending_invites_unavailable")
		}
		for _, pending := range resp.GetInvites() {
			if pending == nil {
				continue
			}
			inv := pending.GetInvite()
			campaign := pending.GetCampaign()
			participant := pending.GetParticipant()

			campaignID := strings.TrimSpace(campaign.GetId())
			if campaignID == "" && inv != nil {
				campaignID = strings.TrimSpace(inv.GetCampaignId())
			}
			campaignName := strings.TrimSpace(campaign.GetName())
			if campaignName == "" {
				if campaignID != "" {
					campaignName = campaignID
				} else {
					campaignName = loc.Sprintf("label.unknown")
				}
			}

			participantLabel := strings.TrimSpace(participant.GetName())
			if participantLabel == "" {
				participantLabel = loc.Sprintf("label.unknown")
			}

			inviteID := ""
			status := statev1.InviteStatus_INVITE_STATUS_UNSPECIFIED
			createdAt := ""
			if inv != nil {
				inviteID = inv.GetId()
				status = inv.GetStatus()
				createdAt = formatTimestamp(inv.GetCreatedAt())
			}
			statusLabel, statusVariant := formatInviteStatus(status, loc)

			rows = append(rows, templates.InviteRow{
				ID:            inviteID,
				CampaignID:    campaignID,
				CampaignName:  campaignName,
				Participant:   participantLabel,
				Status:        statusLabel,
				StatusVariant: statusVariant,
				CreatedAt:     createdAt,
			})
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}

	if len(rows) == 0 {
		return nil, loc.Sprintf("users.invites.empty")
	}
	return rows, ""
}

// handleDashboard renders the dashboard page.
