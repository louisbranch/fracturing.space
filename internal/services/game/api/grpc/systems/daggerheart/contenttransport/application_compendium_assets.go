package contenttransport

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetLootEntry returns a single Daggerheart loot catalog entry.
func (a contentApplication) runGetLootEntry(ctx context.Context, in *pb.GetDaggerheartLootEntryRequest) (*pb.GetDaggerheartLootEntryResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "loot entry request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}
	if _, err := validate.RequiredID(in.GetId(), "loot entry id"); err != nil {
		return nil, err
	}

	entry, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), lootEntryDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartLootEntryResponse{Entry: entry}, nil
}

// ListLootEntries returns Daggerheart loot catalog entries.
func (a contentApplication) runListLootEntries(ctx context.Context, in *pb.ListDaggerheartLootEntriesRequest) (*pb.ListDaggerheartLootEntriesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list loot entries request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}

	entries, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), lootEntryDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartLootEntriesResponse{
		Entries:           entries,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetWeapon returns a single Daggerheart weapon.
func (a contentApplication) runGetWeapon(ctx context.Context, in *pb.GetDaggerheartWeaponRequest) (*pb.GetDaggerheartWeaponResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "weapon request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}
	if _, err := validate.RequiredID(in.GetId(), "weapon id"); err != nil {
		return nil, err
	}

	weapon, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), weaponDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartWeaponResponse{Weapon: weapon}, nil
}

// ListWeapons returns Daggerheart weapons.
func (a contentApplication) runListWeapons(ctx context.Context, in *pb.ListDaggerheartWeaponsRequest) (*pb.ListDaggerheartWeaponsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list weapons request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}

	weapons, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), weaponDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartWeaponsResponse{
		Weapons:           weapons,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetArmor returns a single Daggerheart armor entry.
func (a contentApplication) runGetArmor(ctx context.Context, in *pb.GetDaggerheartArmorRequest) (*pb.GetDaggerheartArmorResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "armor request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}
	if _, err := validate.RequiredID(in.GetId(), "armor id"); err != nil {
		return nil, err
	}

	armor, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), armorDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartArmorResponse{Armor: armor}, nil
}

// ListArmor returns Daggerheart armor entries.
func (a contentApplication) runListArmor(ctx context.Context, in *pb.ListDaggerheartArmorRequest) (*pb.ListDaggerheartArmorResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list armor request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}

	armor, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), armorDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartArmorResponse{
		Armor:             armor,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetItem returns a single Daggerheart item.
func (a contentApplication) runGetItem(ctx context.Context, in *pb.GetDaggerheartItemRequest) (*pb.GetDaggerheartItemResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "item request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}
	if _, err := validate.RequiredID(in.GetId(), "item id"); err != nil {
		return nil, err
	}

	item, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), itemDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartItemResponse{Item: item}, nil
}

// ListItems returns Daggerheart items.
func (a contentApplication) runListItems(ctx context.Context, in *pb.ListDaggerheartItemsRequest) (*pb.ListDaggerheartItemsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list items request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}

	items, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), itemDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartItemsResponse{
		Items:             items,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetEnvironment returns a single Daggerheart environment.
func (a contentApplication) runGetEnvironment(ctx context.Context, in *pb.GetDaggerheartEnvironmentRequest) (*pb.GetDaggerheartEnvironmentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "environment request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}
	if _, err := validate.RequiredID(in.GetId(), "environment id"); err != nil {
		return nil, err
	}

	env, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), environmentDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartEnvironmentResponse{Environment: env}, nil
}

// ListEnvironments returns Daggerheart environments.
func (a contentApplication) runListEnvironments(ctx context.Context, in *pb.ListDaggerheartEnvironmentsRequest) (*pb.ListDaggerheartEnvironmentsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list environments request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}

	items, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), environmentDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartEnvironmentsResponse{
		Environments:      items,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}
