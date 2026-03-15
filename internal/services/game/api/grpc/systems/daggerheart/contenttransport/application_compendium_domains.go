package contenttransport

import (
	"context"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetDamageType returns a single Daggerheart damage type catalog entry.
func (a contentApplication) runGetDamageType(ctx context.Context, in *pb.GetDaggerheartDamageTypeRequest) (*pb.GetDaggerheartDamageTypeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "damage type request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}
	if _, err := validate.RequiredID(in.GetId(), "damage type id"); err != nil {
		return nil, err
	}

	entry, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), damageTypeDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartDamageTypeResponse{DamageType: entry}, nil
}

// ListDamageTypes returns Daggerheart damage type catalog entries.
func (a contentApplication) runListDamageTypes(ctx context.Context, in *pb.ListDaggerheartDamageTypesRequest) (*pb.ListDaggerheartDamageTypesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list damage types request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}

	entries, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), damageTypeDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartDamageTypesResponse{
		DamageTypes:       entries,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetDomain returns a single Daggerheart domain.
func (a contentApplication) runGetDomain(ctx context.Context, in *pb.GetDaggerheartDomainRequest) (*pb.GetDaggerheartDomainResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "domain request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}
	if _, err := validate.RequiredID(in.GetId(), "domain id"); err != nil {
		return nil, err
	}

	domain, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), domainDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartDomainResponse{Domain: domain}, nil
}

// ListDomains returns Daggerheart domains.
func (a contentApplication) runListDomains(ctx context.Context, in *pb.ListDaggerheartDomainsRequest) (*pb.ListDaggerheartDomainsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list domains request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}

	domains, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), domainDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartDomainsResponse{
		Domains:           domains,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetDomainCard returns a single Daggerheart domain card.
func (a contentApplication) runGetDomainCard(ctx context.Context, in *pb.GetDaggerheartDomainCardRequest) (*pb.GetDaggerheartDomainCardResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "domain card request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}
	if _, err := validate.RequiredID(in.GetId(), "domain card id"); err != nil {
		return nil, err
	}

	card, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), domainCardDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartDomainCardResponse{DomainCard: card}, nil
}

// ListDomainCards returns Daggerheart domain cards, optionally filtered by domain.
func (a contentApplication) runListDomainCards(ctx context.Context, in *pb.ListDaggerheartDomainCardsRequest) (*pb.ListDaggerheartDomainCardsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list domain cards request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}

	domainID := strings.TrimSpace(in.GetDomainId())
	req := newContentListRequest(in)
	req.DomainID = domainID
	cards, page, err := listContentEntries(ctx, store, req, in.GetLocale(), domainCardDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartDomainCardsResponse{
		DomainCards:       cards,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}
