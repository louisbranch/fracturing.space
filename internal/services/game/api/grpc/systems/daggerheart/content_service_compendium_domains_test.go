package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func TestGetDamageTypeEntry_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetDamageType(context.Background(), &pb.GetDaggerheartDamageTypeRequest{Id: "dt-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetDamageType().GetName() != "Fire" {
		t.Errorf("name = %q, want Fire", resp.GetDamageType().GetName())
	}
}

func TestListDamageTypes_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListDamageTypes(context.Background(), &pb.ListDaggerheartDamageTypesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetDamageTypes()) != 1 {
		t.Errorf("damage types = %d, want 1", len(resp.GetDamageTypes()))
	}
}

func TestGetDomain_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetDomain(context.Background(), &pb.GetDaggerheartDomainRequest{Id: "dom-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetDomain().GetName() != "Valor" {
		t.Errorf("name = %q, want Valor", resp.GetDomain().GetName())
	}
}

func TestListDomains_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListDomains(context.Background(), &pb.ListDaggerheartDomainsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetDomains()) != 1 {
		t.Errorf("domains = %d, want 1", len(resp.GetDomains()))
	}
}

func TestGetDomainCard_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetDomainCard(context.Background(), &pb.GetDaggerheartDomainCardRequest{Id: "card-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetDomainCard().GetName() != "Fireball" {
		t.Errorf("name = %q, want Fireball", resp.GetDomainCard().GetName())
	}
}

func TestListDomainCards_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListDomainCards(context.Background(), &pb.ListDaggerheartDomainCardsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetDomainCards()) != 1 {
		t.Errorf("domain cards = %d, want 1", len(resp.GetDomainCards()))
	}
}
