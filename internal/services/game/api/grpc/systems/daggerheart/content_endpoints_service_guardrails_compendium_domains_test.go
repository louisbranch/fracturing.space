package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc/codes"
)

func TestListDomains_WithFilter(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListDomains(context.Background(), &pb.ListDaggerheartDomainsRequest{
		Filter: `name = "Valor"`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetDomains()) != 1 {
		t.Fatalf("domains = %d, want 1", len(resp.GetDomains()))
	}
}

func TestListDomainCards_WithFilter(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListDomainCards(context.Background(), &pb.ListDaggerheartDomainCardsRequest{
		Filter: `name = "Fireball"`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetDomainCards()) != 1 {
		t.Fatalf("domain cards = %d, want 1", len(resp.GetDomainCards()))
	}
}

func TestListDamageTypes_DescOrder(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListDamageTypes(context.Background(), &pb.ListDaggerheartDamageTypesRequest{
		OrderBy: "name desc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetDamageTypes()) != 1 {
		t.Fatalf("damage types = %d, want 1", len(resp.GetDamageTypes()))
	}
}

func TestListCompendiumDomainEndpoints_NilRequests(t *testing.T) {
	svc := newContentTestService()
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"ListDamageTypes", func() error { _, err := svc.ListDamageTypes(ctx, nil); return err }},
		{"ListDomains", func() error { _, err := svc.ListDomains(ctx, nil); return err }},
		{"ListDomainCards", func() error { _, err := svc.ListDomainCards(ctx, nil); return err }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertStatusCode(t, tc.fn(), codes.InvalidArgument)
		})
	}
}

func TestListCompendiumDomainEndpoints_NoStore(t *testing.T) {
	svc := &DaggerheartContentService{}
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"ListDamageTypes", func() error { _, err := svc.ListDamageTypes(ctx, &pb.ListDaggerheartDamageTypesRequest{}); return err }},
		{"ListDomains", func() error { _, err := svc.ListDomains(ctx, &pb.ListDaggerheartDomainsRequest{}); return err }},
		{"ListDomainCards", func() error { _, err := svc.ListDomainCards(ctx, &pb.ListDaggerheartDomainCardsRequest{}); return err }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertStatusCode(t, tc.fn(), codes.Internal)
		})
	}
}

func TestGetCompendiumDomainEndpoints_NilRequests(t *testing.T) {
	svc := newContentTestService()
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetDamageType", func() error { _, err := svc.GetDamageType(ctx, nil); return err }},
		{"GetDomain", func() error { _, err := svc.GetDomain(ctx, nil); return err }},
		{"GetDomainCard", func() error { _, err := svc.GetDomainCard(ctx, nil); return err }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertStatusCode(t, tc.fn(), codes.InvalidArgument)
		})
	}
}

func TestGetCompendiumDomainEndpoints_NoStore(t *testing.T) {
	svc := &DaggerheartContentService{}
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetDamageType", func() error {
			_, err := svc.GetDamageType(ctx, &pb.GetDaggerheartDamageTypeRequest{Id: "x"})
			return err
		}},
		{"GetDomain", func() error { _, err := svc.GetDomain(ctx, &pb.GetDaggerheartDomainRequest{Id: "x"}); return err }},
		{"GetDomainCard", func() error {
			_, err := svc.GetDomainCard(ctx, &pb.GetDaggerheartDomainCardRequest{Id: "x"})
			return err
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertStatusCode(t, tc.fn(), codes.Internal)
		})
	}
}

func TestGetCompendiumDomainEndpoints_EmptyID(t *testing.T) {
	svc := newContentTestService()
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetDamageType", func() error {
			_, err := svc.GetDamageType(ctx, &pb.GetDaggerheartDamageTypeRequest{Id: ""})
			return err
		}},
		{"GetDomain", func() error { _, err := svc.GetDomain(ctx, &pb.GetDaggerheartDomainRequest{Id: ""}); return err }},
		{"GetDomainCard", func() error {
			_, err := svc.GetDomainCard(ctx, &pb.GetDaggerheartDomainCardRequest{Id: ""})
			return err
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertStatusCode(t, tc.fn(), codes.InvalidArgument)
		})
	}
}

func TestGetCompendiumDomainEndpoints_NotFound(t *testing.T) {
	svc := newContentTestService()
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetDamageType", func() error {
			_, err := svc.GetDamageType(ctx, &pb.GetDaggerheartDamageTypeRequest{Id: "missing"})
			return err
		}},
		{"GetDomain", func() error { _, err := svc.GetDomain(ctx, &pb.GetDaggerheartDomainRequest{Id: "missing"}); return err }},
		{"GetDomainCard", func() error {
			_, err := svc.GetDomainCard(ctx, &pb.GetDaggerheartDomainCardRequest{Id: "missing"})
			return err
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertStatusCode(t, tc.fn(), codes.NotFound)
		})
	}
}
