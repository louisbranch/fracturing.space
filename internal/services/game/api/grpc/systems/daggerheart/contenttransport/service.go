package contenttransport

import (
	"errors"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
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
