package daggerheart

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc/codes"
)

func TestGetContentCatalog_NoStore(t *testing.T) {
	svc := &DaggerheartContentService{}
	_, err := svc.GetContentCatalog(context.Background(), &pb.GetDaggerheartContentCatalogRequest{})
	assertStatusCode(t, err, codes.Internal)
}

func TestGetContentCatalog_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetContentCatalog(context.Background(), &pb.GetDaggerheartContentCatalogRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	catalog := resp.GetCatalog()
	if len(catalog.GetClasses()) != 2 {
		t.Errorf("classes = %d, want 2", len(catalog.GetClasses()))
	}
	if len(catalog.GetSubclasses()) != 1 {
		t.Errorf("subclasses = %d, want 1", len(catalog.GetSubclasses()))
	}
	if len(catalog.GetHeritages()) != 1 {
		t.Errorf("heritages = %d, want 1", len(catalog.GetHeritages()))
	}
	if len(catalog.GetWeapons()) != 1 {
		t.Errorf("weapons = %d, want 1", len(catalog.GetWeapons()))
	}
	if len(catalog.GetEnvironments()) != 1 {
		t.Errorf("environments = %d, want 1", len(catalog.GetEnvironments()))
	}
}

func TestGetAssetMap_NoStore(t *testing.T) {
	svc := &DaggerheartAssetService{}
	_, err := svc.GetAssetMap(context.Background(), &pb.GetDaggerheartAssetMapRequest{})
	assertStatusCode(t, err, codes.Internal)
}

func TestGetAssetMap_NilRequest(t *testing.T) {
	svc := newAssetTestService()
	_, err := svc.GetAssetMap(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetAssetMap_Success(t *testing.T) {
	svc := newAssetTestService()
	resp, err := svc.GetAssetMap(context.Background(), &pb.GetDaggerheartAssetMapRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assetMap := resp.GetAssetMap()
	if assetMap == nil {
		t.Fatal("expected non-nil asset map")
	}
	if assetMap.GetSystemId() != "daggerheart" {
		t.Fatalf("system id = %q, want %q", assetMap.GetSystemId(), "daggerheart")
	}
	if assetMap.GetLocale() != commonv1.Locale_LOCALE_EN_US {
		t.Fatalf("asset map locale = %v, want %v", assetMap.GetLocale(), commonv1.Locale_LOCALE_EN_US)
	}
	if len(assetMap.GetAssets()) == 0 {
		t.Fatal("expected non-empty content asset refs")
	}

	classIllustration := findAssetRef(
		assetMap.GetAssets(),
		pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_CLASS_ILLUSTRATION,
		"class",
		"class-1",
	)
	if classIllustration == nil {
		t.Fatal("expected class illustration asset ref for class-1")
	}
	if classIllustration.GetStatus() == pb.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_UNSPECIFIED {
		t.Fatal("expected class illustration asset status to be populated")
	}

	domainIcon := findAssetRef(
		assetMap.GetAssets(),
		pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_DOMAIN_ICON,
		"domain",
		"dom-1",
	)
	if domainIcon == nil {
		t.Fatal("expected domain icon asset ref for dom-1")
	}
	if domainIcon.GetStatus() == pb.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_UNSPECIFIED {
		t.Fatal("expected domain icon asset status to be populated")
	}
}

func TestGetAssetMap_UsesRequestedLocale(t *testing.T) {
	svc := newAssetTestService()
	resp, err := svc.GetAssetMap(context.Background(), &pb.GetDaggerheartAssetMapRequest{
		Locale: commonv1.Locale_LOCALE_PT_BR,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := resp.GetAssetMap().GetLocale(); got != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("asset map locale = %v, want %v", got, commonv1.Locale_LOCALE_PT_BR)
	}
}

func findAssetRef(
	assets []*pb.DaggerheartAssetRef,
	assetType pb.DaggerheartAssetType,
	entityType string,
	entityID string,
) *pb.DaggerheartAssetRef {
	for _, asset := range assets {
		if asset == nil {
			continue
		}
		if asset.GetType() != assetType {
			continue
		}
		if asset.GetEntityType() != entityType || asset.GetEntityId() != entityID {
			continue
		}
		return asset
	}
	return nil
}

func newAssetTestService() *DaggerheartAssetService {
	contentService := newContentTestService()
	svc, err := NewDaggerheartAssetService(contentService.store)
	if err != nil {
		panic(err)
	}
	return svc
}
