package admin

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	platformicons "github.com/louisbranch/fracturing.space/internal/platform/icons"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	sharedhtmx "github.com/louisbranch/fracturing.space/internal/services/shared/htmx"
	"golang.org/x/text/message"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	redirectURL := "/users/" + userID
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

// authClient returns the currently configured auth client.
func (h *Handler) authClient() authv1.AuthServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.AuthClient()
}

// accountClient returns the currently configured account client.
func (h *Handler) accountClient() authv1.AccountServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.AccountClient()
}

// daggerheartContentClient returns the Daggerheart content client.
func (h *Handler) daggerheartContentClient() daggerheartv1.DaggerheartContentServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.DaggerheartContentClient()
}

// campaignClient returns the currently configured campaign client.
func (h *Handler) campaignClient() statev1.CampaignServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.CampaignClient()
}

// sessionClient returns the currently configured session client.
func (h *Handler) sessionClient() statev1.SessionServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.SessionClient()
}

// characterClient returns the currently configured character client.
func (h *Handler) characterClient() statev1.CharacterServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.CharacterClient()
}

// participantClient returns the currently configured participant client.
func (h *Handler) participantClient() statev1.ParticipantServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.ParticipantClient()
}

// inviteClient returns the currently configured invite client.
func (h *Handler) inviteClient() statev1.InviteServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.InviteClient()
}

// snapshotClient returns the currently configured snapshot client.
func (h *Handler) snapshotClient() statev1.SnapshotServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.SnapshotClient()
}

// eventClient returns the currently configured event client.
func (h *Handler) eventClient() statev1.EventServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.EventClient()
}

// statisticsClient returns the currently configured statistics client.
func (h *Handler) statisticsClient() statev1.StatisticsServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.StatisticsClient()
}

// systemClient returns the currently configured system client.
func (h *Handler) systemClient() statev1.SystemServiceClient {
	if h == nil || h.clientProvider == nil {
		return nil
	}
	return h.clientProvider.SystemClient()
}

// isHTMXRequest reports whether the request originated from HTMX.
func isHTMXRequest(r *http.Request) bool {
	return sharedhtmx.IsHTMXRequest(r)
}

// splitPathParts returns non-empty path segments.
func splitPathParts(path string) []string {
	rawParts := strings.Split(path, "/")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		parts = append(parts, trimmed)
	}
	return parts
}

func htmxDefaultPageTitle() string {
	return sharedhtmx.TitleTag("Admin | " + templates.AppName())
}

func htmxLocalizedPageTitle(loc *message.Printer, title string, args ...any) string {
	if loc == nil {
		return htmxDefaultPageTitle()
	}
	return sharedhtmx.TitleTag(templates.ComposeAdminPageTitle(templates.T(loc, title, args...)))
}

// renderPage renders page components with consistent HTMX and non-HTMX behavior.
func renderPage(w http.ResponseWriter, r *http.Request, fragment templ.Component, full templ.Component, htmxTitle string) {
	sharedhtmx.RenderPage(w, r, fragment, full, htmxTitle)
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
		detailURL := "/systems/" + system.GetId().String()
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
func catalogSectionColumns(sectionID string, loc *message.Printer) []string {
	switch sectionID {
	case templates.CatalogSectionClasses:
		return []string{loc.Sprintf("catalog.table.starting_hp"), loc.Sprintf("catalog.table.starting_evasion")}
	case templates.CatalogSectionSubclasses:
		return []string{loc.Sprintf("catalog.table.spellcast_trait"), loc.Sprintf("catalog.table.feature_count")}
	case templates.CatalogSectionHeritages:
		return []string{loc.Sprintf("catalog.table.kind"), loc.Sprintf("catalog.table.feature_count")}
	case templates.CatalogSectionExperiences:
		return []string{loc.Sprintf("catalog.table.description")}
	case templates.CatalogSectionDomains:
		return []string{loc.Sprintf("catalog.table.description")}
	case templates.CatalogSectionDomainCards:
		return []string{loc.Sprintf("catalog.table.domain"), loc.Sprintf("catalog.table.level"), loc.Sprintf("catalog.table.type")}
	case templates.CatalogSectionItems:
		return []string{loc.Sprintf("catalog.table.rarity"), loc.Sprintf("catalog.table.kind"), loc.Sprintf("catalog.table.stack_max")}
	case templates.CatalogSectionWeapons:
		return []string{loc.Sprintf("catalog.table.category"), loc.Sprintf("catalog.table.tier"), loc.Sprintf("catalog.table.damage_type")}
	case templates.CatalogSectionArmor:
		return []string{loc.Sprintf("catalog.table.tier"), loc.Sprintf("catalog.table.armor_score")}
	case templates.CatalogSectionLoot:
		return []string{loc.Sprintf("catalog.table.roll"), loc.Sprintf("catalog.table.description")}
	case templates.CatalogSectionDamageTypes:
		return []string{loc.Sprintf("catalog.table.description")}
	case templates.CatalogSectionAdversaries:
		return []string{loc.Sprintf("catalog.table.tier"), loc.Sprintf("catalog.table.role")}
	case templates.CatalogSectionBeastforms:
		return []string{loc.Sprintf("catalog.table.tier"), loc.Sprintf("catalog.table.trait")}
	case templates.CatalogSectionCompanionExperiences:
		return []string{loc.Sprintf("catalog.table.description")}
	case templates.CatalogSectionEnvironments:
		return []string{loc.Sprintf("catalog.table.type"), loc.Sprintf("catalog.table.difficulty")}
	default:
		return nil
	}
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
		DetailURL: "/catalog/daggerheart/" + sectionID + "/" + entryID,
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
		BackURL:   "/catalog/daggerheart/" + sectionID,
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

// formatGmMode returns a display label for a GM mode enum.
func formatGmMode(mode statev1.GmMode, loc *message.Printer) string {
	switch mode {
	case statev1.GmMode_HUMAN:
		return loc.Sprintf("label.human")
	case statev1.GmMode_AI:
		return loc.Sprintf("label.ai")
	case statev1.GmMode_HYBRID:
		return loc.Sprintf("label.hybrid")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatGameSystem(system commonv1.GameSystem, loc *message.Printer) string {
	switch system {
	case commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART:
		return loc.Sprintf("label.daggerheart")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatImplementationStage(stage commonv1.GameSystemImplementationStage, loc *message.Printer) string {
	switch stage {
	case commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PLANNED:
		return loc.Sprintf("label.system_stage_planned")
	case commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PARTIAL:
		return loc.Sprintf("label.system_stage_partial")
	case commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_COMPLETE:
		return loc.Sprintf("label.system_stage_complete")
	case commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_DEPRECATED:
		return loc.Sprintf("label.system_stage_deprecated")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatOperationalStatus(status commonv1.GameSystemOperationalStatus, loc *message.Printer) string {
	switch status {
	case commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OFFLINE:
		return loc.Sprintf("label.system_status_offline")
	case commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_DEGRADED:
		return loc.Sprintf("label.system_status_degraded")
	case commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL:
		return loc.Sprintf("label.system_status_operational")
	case commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_MAINTENANCE:
		return loc.Sprintf("label.system_status_maintenance")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatAccessLevel(level commonv1.GameSystemAccessLevel, loc *message.Printer) string {
	switch level {
	case commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_INTERNAL:
		return loc.Sprintf("label.system_access_internal")
	case commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_BETA:
		return loc.Sprintf("label.system_access_beta")
	case commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_PUBLIC:
		return loc.Sprintf("label.system_access_public")
	case commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_RETIRED:
		return loc.Sprintf("label.system_access_retired")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func parseSystemID(value string) commonv1.GameSystem {
	trimmed := strings.ToUpper(strings.TrimSpace(value))
	if trimmed == "" {
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED
	}
	if trimmed == "DAGGERHEART" {
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
	}
	if enumValue, ok := commonv1.GameSystem_value[trimmed]; ok {
		return commonv1.GameSystem(enumValue)
	}
	return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED
}

func parseGameSystem(value string) (commonv1.GameSystem, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "daggerheart":
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, true
	default:
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, false
	}
}

func parseGmMode(value string) (statev1.GmMode, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "human":
		return statev1.GmMode_HUMAN, true
	case "ai":
		return statev1.GmMode_AI, true
	case "hybrid":
		return statev1.GmMode_HYBRID, true
	default:
		return statev1.GmMode_GM_MODE_UNSPECIFIED, false
	}
}

// formatSessionStatus returns a display label for a session status.
func formatSessionStatus(status statev1.SessionStatus, loc *message.Printer) string {
	switch status {
	case statev1.SessionStatus_SESSION_ACTIVE:
		return loc.Sprintf("label.active")
	case statev1.SessionStatus_SESSION_ENDED:
		return loc.Sprintf("label.ended")
	default:
		return loc.Sprintf("label.unspecified")
	}
}

func formatInviteStatus(status statev1.InviteStatus, loc *message.Printer) (string, string) {
	switch status {
	case statev1.InviteStatus_PENDING:
		return loc.Sprintf("label.invite_pending"), "warning"
	case statev1.InviteStatus_CLAIMED:
		return loc.Sprintf("label.invite_claimed"), "success"
	case statev1.InviteStatus_REVOKED:
		return loc.Sprintf("label.invite_revoked"), "error"
	default:
		return loc.Sprintf("label.unspecified"), "secondary"
	}
}

// formatCreatedDate returns a YYYY-MM-DD string for a timestamp.
func formatCreatedDate(createdAt *timestamppb.Timestamp) string {
	if createdAt == nil {
		return ""
	}
	return createdAt.AsTime().Format("2006-01-02")
}

// formatTimestamp returns a YYYY-MM-DD HH:MM:SS string for a timestamp.
func formatTimestamp(value *timestamppb.Timestamp) string {
	if value == nil {
		return ""
	}
	return value.AsTime().Format("2006-01-02 15:04:05")
}

// truncateText shortens text to a maximum length with an ellipsis.
func truncateText(text string, limit int) string {
	if limit <= 0 || text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "..."
}

// handleDashboard renders the dashboard page.
