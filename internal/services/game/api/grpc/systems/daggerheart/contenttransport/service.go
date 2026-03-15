package contenttransport

import (
	"context"
	"errors"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultListDaggerheartContentPageSize = 50
	maxListDaggerheartContentPageSize     = 200
)

type contentListRequestInput interface {
	GetPageSize() int32
	GetPageToken() string
	GetOrderBy() string
	GetFilter() string
}

func newContentListRequest(in contentListRequestInput) contentListRequest {
	return contentListRequest{
		PageSize:  in.GetPageSize(),
		PageToken: in.GetPageToken(),
		OrderBy:   in.GetOrderBy(),
		Filter:    in.GetFilter(),
	}
}

// Handler owns Daggerheart content and asset-map transport logic.
//
// The root Daggerheart gRPC package constructs this handler from its validated
// content-store dependency and keeps only thin gRPC wrappers at the package
// root.
type Handler struct {
	store contentstore.DaggerheartContentReadStore
}

// NewHandler binds content/catalog transport logic to one content store.
func NewHandler(store contentstore.DaggerheartContentReadStore) *Handler {
	return &Handler{store: store}
}

func (h *Handler) GetContentCatalog(ctx context.Context, in *pb.GetDaggerheartContentCatalogRequest) (*pb.GetDaggerheartContentCatalogResponse, error) {
	return newContentApplication(h).runGetContentCatalog(ctx, in)
}

func (h *Handler) GetClass(ctx context.Context, in *pb.GetDaggerheartClassRequest) (*pb.GetDaggerheartClassResponse, error) {
	return newContentApplication(h).runGetClass(ctx, in)
}

func (h *Handler) ListClasses(ctx context.Context, in *pb.ListDaggerheartClassesRequest) (*pb.ListDaggerheartClassesResponse, error) {
	return newContentApplication(h).runListClasses(ctx, in)
}

func (h *Handler) GetSubclass(ctx context.Context, in *pb.GetDaggerheartSubclassRequest) (*pb.GetDaggerheartSubclassResponse, error) {
	return newContentApplication(h).runGetSubclass(ctx, in)
}

func (h *Handler) ListSubclasses(ctx context.Context, in *pb.ListDaggerheartSubclassesRequest) (*pb.ListDaggerheartSubclassesResponse, error) {
	return newContentApplication(h).runListSubclasses(ctx, in)
}

func (h *Handler) GetHeritage(ctx context.Context, in *pb.GetDaggerheartHeritageRequest) (*pb.GetDaggerheartHeritageResponse, error) {
	return newContentApplication(h).runGetHeritage(ctx, in)
}

func (h *Handler) ListHeritages(ctx context.Context, in *pb.ListDaggerheartHeritagesRequest) (*pb.ListDaggerheartHeritagesResponse, error) {
	return newContentApplication(h).runListHeritages(ctx, in)
}

func (h *Handler) GetExperience(ctx context.Context, in *pb.GetDaggerheartExperienceRequest) (*pb.GetDaggerheartExperienceResponse, error) {
	return newContentApplication(h).runGetExperience(ctx, in)
}

func (h *Handler) ListExperiences(ctx context.Context, in *pb.ListDaggerheartExperiencesRequest) (*pb.ListDaggerheartExperiencesResponse, error) {
	return newContentApplication(h).runListExperiences(ctx, in)
}

func (h *Handler) GetAdversary(ctx context.Context, in *pb.GetDaggerheartAdversaryRequest) (*pb.GetDaggerheartAdversaryResponse, error) {
	return newContentApplication(h).runGetAdversary(ctx, in)
}

func (h *Handler) ListAdversaries(ctx context.Context, in *pb.ListDaggerheartAdversariesRequest) (*pb.ListDaggerheartAdversariesResponse, error) {
	return newContentApplication(h).runListAdversaries(ctx, in)
}

func (h *Handler) GetBeastform(ctx context.Context, in *pb.GetDaggerheartBeastformRequest) (*pb.GetDaggerheartBeastformResponse, error) {
	return newContentApplication(h).runGetBeastform(ctx, in)
}

func (h *Handler) ListBeastforms(ctx context.Context, in *pb.ListDaggerheartBeastformsRequest) (*pb.ListDaggerheartBeastformsResponse, error) {
	return newContentApplication(h).runListBeastforms(ctx, in)
}

func (h *Handler) GetCompanionExperience(ctx context.Context, in *pb.GetDaggerheartCompanionExperienceRequest) (*pb.GetDaggerheartCompanionExperienceResponse, error) {
	return newContentApplication(h).runGetCompanionExperience(ctx, in)
}

func (h *Handler) ListCompanionExperiences(ctx context.Context, in *pb.ListDaggerheartCompanionExperiencesRequest) (*pb.ListDaggerheartCompanionExperiencesResponse, error) {
	return newContentApplication(h).runListCompanionExperiences(ctx, in)
}

func (h *Handler) GetLootEntry(ctx context.Context, in *pb.GetDaggerheartLootEntryRequest) (*pb.GetDaggerheartLootEntryResponse, error) {
	return newContentApplication(h).runGetLootEntry(ctx, in)
}

func (h *Handler) ListLootEntries(ctx context.Context, in *pb.ListDaggerheartLootEntriesRequest) (*pb.ListDaggerheartLootEntriesResponse, error) {
	return newContentApplication(h).runListLootEntries(ctx, in)
}

func (h *Handler) GetDamageType(ctx context.Context, in *pb.GetDaggerheartDamageTypeRequest) (*pb.GetDaggerheartDamageTypeResponse, error) {
	return newContentApplication(h).runGetDamageType(ctx, in)
}

func (h *Handler) ListDamageTypes(ctx context.Context, in *pb.ListDaggerheartDamageTypesRequest) (*pb.ListDaggerheartDamageTypesResponse, error) {
	return newContentApplication(h).runListDamageTypes(ctx, in)
}

func (h *Handler) GetDomain(ctx context.Context, in *pb.GetDaggerheartDomainRequest) (*pb.GetDaggerheartDomainResponse, error) {
	return newContentApplication(h).runGetDomain(ctx, in)
}

func (h *Handler) ListDomains(ctx context.Context, in *pb.ListDaggerheartDomainsRequest) (*pb.ListDaggerheartDomainsResponse, error) {
	return newContentApplication(h).runListDomains(ctx, in)
}

func (h *Handler) GetDomainCard(ctx context.Context, in *pb.GetDaggerheartDomainCardRequest) (*pb.GetDaggerheartDomainCardResponse, error) {
	return newContentApplication(h).runGetDomainCard(ctx, in)
}

func (h *Handler) ListDomainCards(ctx context.Context, in *pb.ListDaggerheartDomainCardsRequest) (*pb.ListDaggerheartDomainCardsResponse, error) {
	return newContentApplication(h).runListDomainCards(ctx, in)
}

func (h *Handler) GetWeapon(ctx context.Context, in *pb.GetDaggerheartWeaponRequest) (*pb.GetDaggerheartWeaponResponse, error) {
	return newContentApplication(h).runGetWeapon(ctx, in)
}

func (h *Handler) ListWeapons(ctx context.Context, in *pb.ListDaggerheartWeaponsRequest) (*pb.ListDaggerheartWeaponsResponse, error) {
	return newContentApplication(h).runListWeapons(ctx, in)
}

func (h *Handler) GetArmor(ctx context.Context, in *pb.GetDaggerheartArmorRequest) (*pb.GetDaggerheartArmorResponse, error) {
	return newContentApplication(h).runGetArmor(ctx, in)
}

func (h *Handler) ListArmor(ctx context.Context, in *pb.ListDaggerheartArmorRequest) (*pb.ListDaggerheartArmorResponse, error) {
	return newContentApplication(h).runListArmor(ctx, in)
}

func (h *Handler) GetItem(ctx context.Context, in *pb.GetDaggerheartItemRequest) (*pb.GetDaggerheartItemResponse, error) {
	return newContentApplication(h).runGetItem(ctx, in)
}

func (h *Handler) ListItems(ctx context.Context, in *pb.ListDaggerheartItemsRequest) (*pb.ListDaggerheartItemsResponse, error) {
	return newContentApplication(h).runListItems(ctx, in)
}

func (h *Handler) GetEnvironment(ctx context.Context, in *pb.GetDaggerheartEnvironmentRequest) (*pb.GetDaggerheartEnvironmentResponse, error) {
	return newContentApplication(h).runGetEnvironment(ctx, in)
}

func (h *Handler) ListEnvironments(ctx context.Context, in *pb.ListDaggerheartEnvironmentsRequest) (*pb.ListDaggerheartEnvironmentsResponse, error) {
	return newContentApplication(h).runListEnvironments(ctx, in)
}

// GetAssetMap returns resolved content-image selectors for Daggerheart entities.
func (h *Handler) GetAssetMap(ctx context.Context, in *pb.GetDaggerheartAssetMapRequest) (*pb.GetDaggerheartAssetMapResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "asset map request is required")
	}
	store, err := h.contentStore()
	if err != nil {
		return nil, err
	}

	assetMap, err := buildDaggerheartAssetMap(ctx, store, in.GetLocale())
	if err != nil {
		return nil, grpcerror.Internal("asset map pipeline", err)
	}
	return &pb.GetDaggerheartAssetMapResponse{AssetMap: assetMap}, nil
}

func (h *Handler) contentStore() (contentstore.DaggerheartContentReadStore, error) {
	if h == nil || h.store == nil {
		return nil, status.Error(codes.Internal, "content store is not configured")
	}
	return h.store, nil
}

func mapContentErr(action string, err error) error {
	if errors.Is(err, storage.ErrNotFound) {
		return status.Error(codes.NotFound, "content not found")
	}
	return grpcerror.Internal(action, err)
}

func toProtoDaggerheartClass(class contentstore.DaggerheartClass) *pb.DaggerheartClass {
	return &pb.DaggerheartClass{
		Id:              class.ID,
		Name:            class.Name,
		StartingEvasion: int32(class.StartingEvasion),
		StartingHp:      int32(class.StartingHP),
		StartingItems:   append([]string{}, class.StartingItems...),
		Features:        toProtoDaggerheartFeatures(class.Features),
		HopeFeature:     toProtoDaggerheartHopeFeature(class.HopeFeature),
		DomainIds:       append([]string{}, class.DomainIDs...),
	}
}

func toProtoDaggerheartClasses(classes []contentstore.DaggerheartClass) []*pb.DaggerheartClass {
	items := make([]*pb.DaggerheartClass, 0, len(classes))
	for _, class := range classes {
		items = append(items, toProtoDaggerheartClass(class))
	}
	return items
}

func toProtoDaggerheartSubclass(subclass contentstore.DaggerheartSubclass) *pb.DaggerheartSubclass {
	return &pb.DaggerheartSubclass{
		Id:                     subclass.ID,
		Name:                   subclass.Name,
		ClassId:                subclass.ClassID,
		SpellcastTrait:         subclass.SpellcastTrait,
		FoundationFeatures:     toProtoDaggerheartFeatures(subclass.FoundationFeatures),
		SpecializationFeatures: toProtoDaggerheartFeatures(subclass.SpecializationFeatures),
		MasteryFeatures:        toProtoDaggerheartFeatures(subclass.MasteryFeatures),
	}
}

func toProtoDaggerheartSubclasses(subclasses []contentstore.DaggerheartSubclass) []*pb.DaggerheartSubclass {
	items := make([]*pb.DaggerheartSubclass, 0, len(subclasses))
	for _, subclass := range subclasses {
		items = append(items, toProtoDaggerheartSubclass(subclass))
	}
	return items
}

func toProtoDaggerheartHeritage(heritage contentstore.DaggerheartHeritage) *pb.DaggerheartHeritage {
	return &pb.DaggerheartHeritage{
		Id:       heritage.ID,
		Name:     heritage.Name,
		Kind:     heritageKindToProto(heritage.Kind),
		Features: toProtoDaggerheartFeatures(heritage.Features),
	}
}

func toProtoDaggerheartExperience(experience contentstore.DaggerheartExperienceEntry) *pb.DaggerheartExperienceEntry {
	return &pb.DaggerheartExperienceEntry{
		Id:          experience.ID,
		Name:        experience.Name,
		Description: experience.Description,
	}
}

func toProtoDaggerheartExperiences(experiences []contentstore.DaggerheartExperienceEntry) []*pb.DaggerheartExperienceEntry {
	items := make([]*pb.DaggerheartExperienceEntry, 0, len(experiences))
	for _, experience := range experiences {
		items = append(items, toProtoDaggerheartExperience(experience))
	}
	return items
}

func toProtoDaggerheartAdversaryAttack(attack contentstore.DaggerheartAdversaryAttack) *pb.DaggerheartAdversaryAttack {
	return &pb.DaggerheartAdversaryAttack{
		Name:        attack.Name,
		Range:       attack.Range,
		DamageDice:  toProtoDaggerheartDamageDice(attack.DamageDice),
		DamageBonus: int32(attack.DamageBonus),
		DamageType:  damageTypeToProto(attack.DamageType),
	}
}

func toProtoDaggerheartAdversaryExperiences(experiences []contentstore.DaggerheartAdversaryExperience) []*pb.DaggerheartAdversaryExperience {
	items := make([]*pb.DaggerheartAdversaryExperience, 0, len(experiences))
	for _, experience := range experiences {
		items = append(items, &pb.DaggerheartAdversaryExperience{
			Name:     experience.Name,
			Modifier: int32(experience.Modifier),
		})
	}
	return items
}

func toProtoDaggerheartAdversaryFeatures(features []contentstore.DaggerheartAdversaryFeature) []*pb.DaggerheartAdversaryFeature {
	items := make([]*pb.DaggerheartAdversaryFeature, 0, len(features))
	for _, feature := range features {
		items = append(items, &pb.DaggerheartAdversaryFeature{
			Id:          feature.ID,
			Name:        feature.Name,
			Kind:        feature.Kind,
			Description: feature.Description,
			CostType:    feature.CostType,
			Cost:        int32(feature.Cost),
		})
	}
	return items
}

func toProtoDaggerheartAdversaryEntry(entry contentstore.DaggerheartAdversaryEntry) *pb.DaggerheartAdversaryEntry {
	return &pb.DaggerheartAdversaryEntry{
		Id:              entry.ID,
		Name:            entry.Name,
		Tier:            int32(entry.Tier),
		Role:            entry.Role,
		Description:     entry.Description,
		Motives:         entry.Motives,
		Difficulty:      int32(entry.Difficulty),
		MajorThreshold:  int32(entry.MajorThreshold),
		SevereThreshold: int32(entry.SevereThreshold),
		Hp:              int32(entry.HP),
		Stress:          int32(entry.Stress),
		Armor:           int32(entry.Armor),
		AttackModifier:  int32(entry.AttackModifier),
		StandardAttack:  toProtoDaggerheartAdversaryAttack(entry.StandardAttack),
		Experiences:     toProtoDaggerheartAdversaryExperiences(entry.Experiences),
		Features:        toProtoDaggerheartAdversaryFeatures(entry.Features),
	}
}

func toProtoDaggerheartAdversaryEntries(entries []contentstore.DaggerheartAdversaryEntry) []*pb.DaggerheartAdversaryEntry {
	items := make([]*pb.DaggerheartAdversaryEntry, 0, len(entries))
	for _, entry := range entries {
		items = append(items, toProtoDaggerheartAdversaryEntry(entry))
	}
	return items
}

func toProtoDaggerheartBeastformAttack(attack contentstore.DaggerheartBeastformAttack) *pb.DaggerheartBeastformAttack {
	return &pb.DaggerheartBeastformAttack{
		Range:       attack.Range,
		Trait:       attack.Trait,
		DamageDice:  toProtoDaggerheartDamageDice(attack.DamageDice),
		DamageBonus: int32(attack.DamageBonus),
		DamageType:  damageTypeToProto(attack.DamageType),
	}
}

func toProtoDaggerheartBeastformFeatures(features []contentstore.DaggerheartBeastformFeature) []*pb.DaggerheartBeastformFeature {
	items := make([]*pb.DaggerheartBeastformFeature, 0, len(features))
	for _, feature := range features {
		items = append(items, &pb.DaggerheartBeastformFeature{
			Id:          feature.ID,
			Name:        feature.Name,
			Description: feature.Description,
		})
	}
	return items
}

func toProtoDaggerheartBeastform(beastform contentstore.DaggerheartBeastformEntry) *pb.DaggerheartBeastformEntry {
	return &pb.DaggerheartBeastformEntry{
		Id:           beastform.ID,
		Name:         beastform.Name,
		Tier:         int32(beastform.Tier),
		Examples:     beastform.Examples,
		Trait:        beastform.Trait,
		TraitBonus:   int32(beastform.TraitBonus),
		EvasionBonus: int32(beastform.EvasionBonus),
		Attack:       toProtoDaggerheartBeastformAttack(beastform.Attack),
		Advantages:   append([]string{}, beastform.Advantages...),
		Features:     toProtoDaggerheartBeastformFeatures(beastform.Features),
	}
}

func toProtoDaggerheartBeastforms(beastforms []contentstore.DaggerheartBeastformEntry) []*pb.DaggerheartBeastformEntry {
	items := make([]*pb.DaggerheartBeastformEntry, 0, len(beastforms))
	for _, beastform := range beastforms {
		items = append(items, toProtoDaggerheartBeastform(beastform))
	}
	return items
}

func toProtoDaggerheartCompanionExperience(experience contentstore.DaggerheartCompanionExperienceEntry) *pb.DaggerheartCompanionExperienceEntry {
	return &pb.DaggerheartCompanionExperienceEntry{
		Id:          experience.ID,
		Name:        experience.Name,
		Description: experience.Description,
	}
}

func toProtoDaggerheartCompanionExperiences(experiences []contentstore.DaggerheartCompanionExperienceEntry) []*pb.DaggerheartCompanionExperienceEntry {
	items := make([]*pb.DaggerheartCompanionExperienceEntry, 0, len(experiences))
	for _, experience := range experiences {
		items = append(items, toProtoDaggerheartCompanionExperience(experience))
	}
	return items
}

func toProtoDaggerheartLootEntry(entry contentstore.DaggerheartLootEntry) *pb.DaggerheartLootEntry {
	return &pb.DaggerheartLootEntry{
		Id:          entry.ID,
		Name:        entry.Name,
		Roll:        int32(entry.Roll),
		Description: entry.Description,
	}
}

func toProtoDaggerheartLootEntries(entries []contentstore.DaggerheartLootEntry) []*pb.DaggerheartLootEntry {
	items := make([]*pb.DaggerheartLootEntry, 0, len(entries))
	for _, entry := range entries {
		items = append(items, toProtoDaggerheartLootEntry(entry))
	}
	return items
}

func toProtoDaggerheartDamageType(entry contentstore.DaggerheartDamageTypeEntry) *pb.DaggerheartDamageTypeEntry {
	return &pb.DaggerheartDamageTypeEntry{
		Id:          entry.ID,
		Name:        entry.Name,
		Description: entry.Description,
	}
}

func toProtoDaggerheartDamageTypes(entries []contentstore.DaggerheartDamageTypeEntry) []*pb.DaggerheartDamageTypeEntry {
	items := make([]*pb.DaggerheartDamageTypeEntry, 0, len(entries))
	for _, entry := range entries {
		items = append(items, toProtoDaggerheartDamageType(entry))
	}
	return items
}

func toProtoDaggerheartHeritages(heritages []contentstore.DaggerheartHeritage) []*pb.DaggerheartHeritage {
	items := make([]*pb.DaggerheartHeritage, 0, len(heritages))
	for _, heritage := range heritages {
		items = append(items, toProtoDaggerheartHeritage(heritage))
	}
	return items
}

func toProtoDaggerheartDomain(domain contentstore.DaggerheartDomain) *pb.DaggerheartDomain {
	return &pb.DaggerheartDomain{
		Id:          domain.ID,
		Name:        domain.Name,
		Description: domain.Description,
	}
}

func toProtoDaggerheartDomains(domains []contentstore.DaggerheartDomain) []*pb.DaggerheartDomain {
	items := make([]*pb.DaggerheartDomain, 0, len(domains))
	for _, domain := range domains {
		items = append(items, toProtoDaggerheartDomain(domain))
	}
	return items
}

func toProtoDaggerheartDomainCard(card contentstore.DaggerheartDomainCard) *pb.DaggerheartDomainCard {
	return &pb.DaggerheartDomainCard{
		Id:          card.ID,
		Name:        card.Name,
		DomainId:    card.DomainID,
		Level:       int32(card.Level),
		Type:        domainCardTypeToProto(card.Type),
		RecallCost:  int32(card.RecallCost),
		UsageLimit:  card.UsageLimit,
		FeatureText: card.FeatureText,
	}
}

func toProtoDaggerheartDomainCards(cards []contentstore.DaggerheartDomainCard) []*pb.DaggerheartDomainCard {
	items := make([]*pb.DaggerheartDomainCard, 0, len(cards))
	for _, card := range cards {
		items = append(items, toProtoDaggerheartDomainCard(card))
	}
	return items
}

func toProtoDaggerheartWeapon(weapon contentstore.DaggerheartWeapon) *pb.DaggerheartWeapon {
	return &pb.DaggerheartWeapon{
		Id:         weapon.ID,
		Name:       weapon.Name,
		Category:   weaponCategoryToProto(weapon.Category),
		Tier:       int32(weapon.Tier),
		Trait:      weapon.Trait,
		Range:      weapon.Range,
		DamageDice: toProtoDaggerheartDamageDice(weapon.DamageDice),
		DamageType: damageTypeToProto(weapon.DamageType),
		Burden:     int32(weapon.Burden),
		Feature:    weapon.Feature,
	}
}

func toProtoDaggerheartWeapons(weapons []contentstore.DaggerheartWeapon) []*pb.DaggerheartWeapon {
	items := make([]*pb.DaggerheartWeapon, 0, len(weapons))
	for _, weapon := range weapons {
		items = append(items, toProtoDaggerheartWeapon(weapon))
	}
	return items
}

func toProtoDaggerheartArmor(armor contentstore.DaggerheartArmor) *pb.DaggerheartArmor {
	return &pb.DaggerheartArmor{
		Id:                  armor.ID,
		Name:                armor.Name,
		Tier:                int32(armor.Tier),
		BaseMajorThreshold:  int32(armor.BaseMajorThreshold),
		BaseSevereThreshold: int32(armor.BaseSevereThreshold),
		ArmorScore:          int32(armor.ArmorScore),
		Feature:             armor.Feature,
	}
}

func toProtoDaggerheartArmorList(items []contentstore.DaggerheartArmor) []*pb.DaggerheartArmor {
	armor := make([]*pb.DaggerheartArmor, 0, len(items))
	for _, item := range items {
		armor = append(armor, toProtoDaggerheartArmor(item))
	}
	return armor
}

func toProtoDaggerheartItem(item contentstore.DaggerheartItem) *pb.DaggerheartItem {
	return &pb.DaggerheartItem{
		Id:          item.ID,
		Name:        item.Name,
		Rarity:      itemRarityToProto(item.Rarity),
		Kind:        itemKindToProto(item.Kind),
		StackMax:    int32(item.StackMax),
		Description: item.Description,
		EffectText:  item.EffectText,
	}
}

func toProtoDaggerheartItems(items []contentstore.DaggerheartItem) []*pb.DaggerheartItem {
	results := make([]*pb.DaggerheartItem, 0, len(items))
	for _, item := range items {
		results = append(results, toProtoDaggerheartItem(item))
	}
	return results
}

func toProtoDaggerheartEnvironment(env contentstore.DaggerheartEnvironment) *pb.DaggerheartEnvironment {
	return &pb.DaggerheartEnvironment{
		Id:                    env.ID,
		Name:                  env.Name,
		Tier:                  int32(env.Tier),
		Type:                  environmentTypeToProto(env.Type),
		Difficulty:            int32(env.Difficulty),
		Impulses:              append([]string{}, env.Impulses...),
		PotentialAdversaryIds: append([]string{}, env.PotentialAdversaryIDs...),
		Features:              toProtoDaggerheartFeatures(env.Features),
		Prompts:               append([]string{}, env.Prompts...),
	}
}

func toProtoDaggerheartEnvironments(envs []contentstore.DaggerheartEnvironment) []*pb.DaggerheartEnvironment {
	results := make([]*pb.DaggerheartEnvironment, 0, len(envs))
	for _, env := range envs {
		results = append(results, toProtoDaggerheartEnvironment(env))
	}
	return results
}

func toProtoDaggerheartFeatures(features []contentstore.DaggerheartFeature) []*pb.DaggerheartFeature {
	items := make([]*pb.DaggerheartFeature, 0, len(features))
	for _, feature := range features {
		items = append(items, &pb.DaggerheartFeature{
			Id:          feature.ID,
			Name:        feature.Name,
			Description: feature.Description,
			Level:       int32(feature.Level),
		})
	}
	return items
}

func toProtoDaggerheartHopeFeature(feature contentstore.DaggerheartHopeFeature) *pb.DaggerheartHopeFeature {
	return &pb.DaggerheartHopeFeature{
		Name:        feature.Name,
		Description: feature.Description,
		HopeCost:    int32(feature.HopeCost),
	}
}

func toProtoDaggerheartDamageDice(dice []contentstore.DaggerheartDamageDie) []*pb.DiceSpec {
	results := make([]*pb.DiceSpec, 0, len(dice))
	for _, die := range dice {
		results = append(results, &pb.DiceSpec{
			Sides: int32(die.Sides),
			Count: int32(die.Count),
		})
	}
	return results
}

func heritageKindToProto(kind string) pb.DaggerheartHeritageKind {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "ancestry":
		return pb.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_ANCESTRY
	case "community":
		return pb.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_COMMUNITY
	default:
		return pb.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_UNSPECIFIED
	}
}

func domainCardTypeToProto(kind string) pb.DaggerheartDomainCardType {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "ability":
		return pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_ABILITY
	case "spell":
		return pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_SPELL
	case "grimoire":
		return pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_GRIMOIRE
	default:
		return pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_UNSPECIFIED
	}
}

func weaponCategoryToProto(kind string) pb.DaggerheartWeaponCategory {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "primary":
		return pb.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_PRIMARY
	case "secondary":
		return pb.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_SECONDARY
	default:
		return pb.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_UNSPECIFIED
	}
}

func itemRarityToProto(kind string) pb.DaggerheartItemRarity {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "common":
		return pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_COMMON
	case "uncommon":
		return pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_UNCOMMON
	case "rare":
		return pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_RARE
	case "unique":
		return pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_UNIQUE
	case "legendary":
		return pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_LEGENDARY
	default:
		return pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_UNSPECIFIED
	}
}

func itemKindToProto(kind string) pb.DaggerheartItemKind {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "consumable":
		return pb.DaggerheartItemKind_DAGGERHEART_ITEM_KIND_CONSUMABLE
	case "equipment":
		return pb.DaggerheartItemKind_DAGGERHEART_ITEM_KIND_EQUIPMENT
	case "treasure":
		return pb.DaggerheartItemKind_DAGGERHEART_ITEM_KIND_TREASURE
	default:
		return pb.DaggerheartItemKind_DAGGERHEART_ITEM_KIND_UNSPECIFIED
	}
}

func environmentTypeToProto(kind string) pb.DaggerheartEnvironmentType {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "exploration":
		return pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_EXPLORATION
	case "social":
		return pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_SOCIAL
	case "traversal":
		return pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_TRAVERSAL
	case "event":
		return pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_EVENT
	default:
		return pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_UNSPECIFIED
	}
}

func damageTypeToProto(kind string) pb.DaggerheartDamageType {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "physical":
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL
	case "magic":
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC
	case "mixed":
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED
	default:
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED
	}
}
