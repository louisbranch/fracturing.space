package contenttransport

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type contentDescriptor[T any, P any] struct {
	getAction      string
	listAction     string
	localizeAction string
	get            func(context.Context, contentstore.DaggerheartContentReadStore, string) (T, error)
	list           func(context.Context, contentstore.DaggerheartContentReadStore) ([]T, error)
	listByRequest  func(context.Context, contentstore.DaggerheartContentReadStore, contentListRequest) ([]T, error)
	filterHashSeed func(contentListRequest) string
	localize       func(context.Context, contentstore.DaggerheartContentReadStore, commonv1.Locale, []T) error
	toProto        func(T) *P
	toProtoList    func([]T) []*P
	listConfig     contentListConfig[T]
}

func getContentEntry[T any, P any](
	ctx context.Context,
	store contentstore.DaggerheartContentReadStore,
	id string,
	locale commonv1.Locale,
	descriptor contentDescriptor[T, P],
) (*P, error) {
	item, err := descriptor.get(ctx, store, id)
	if err != nil {
		return nil, mapContentErr(descriptor.getAction, err)
	}
	items := []T{item}
	if err := descriptor.localize(ctx, store, locale, items); err != nil {
		return nil, grpcerror.Internal(descriptor.localizeAction, err)
	}
	return descriptor.toProto(items[0]), nil
}

func listContentEntries[T any, P any](
	ctx context.Context,
	store contentstore.DaggerheartContentReadStore,
	req contentListRequest,
	locale commonv1.Locale,
	descriptor contentDescriptor[T, P],
) ([]*P, contentPage[T], error) {
	listFunc := descriptor.list
	if descriptor.listByRequest != nil {
		listFunc = func(listCtx context.Context, listStore contentstore.DaggerheartContentReadStore) ([]T, error) {
			return descriptor.listByRequest(listCtx, listStore, req)
		}
	}
	items, err := listFunc(ctx, store)
	if err != nil {
		return nil, contentPage[T]{}, grpcerror.Internal(descriptor.listAction, err)
	}
	listConfig := descriptor.listConfig
	if descriptor.filterHashSeed != nil {
		listConfig.FilterHashSeed = descriptor.filterHashSeed(req)
	}
	page, err := listContentPage(items, req, listConfig)
	if err != nil {
		return nil, contentPage[T]{}, status.Errorf(codes.InvalidArgument, "%s: %v", descriptor.listAction, err)
	}
	if err := descriptor.localize(ctx, store, locale, page.Items); err != nil {
		return nil, contentPage[T]{}, grpcerror.Internal(descriptor.localizeAction, err)
	}
	return descriptor.toProtoList(page.Items), page, nil
}
