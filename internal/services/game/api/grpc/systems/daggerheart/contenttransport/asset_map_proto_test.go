package contenttransport

import (
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
)

func TestAssetTypeToProto(t *testing.T) {
	tests := []struct {
		name      string
		assetType string
		want      pb.DaggerheartAssetType
	}{
		{name: "class icon", assetType: catalog.DaggerheartAssetTypeClassIcon, want: pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_CLASS_ICON},
		{name: "domain illustration", assetType: catalog.DaggerheartAssetTypeDomainIllustration, want: pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_DOMAIN_ILLUSTRATION},
		{name: "unknown", assetType: "unknown", want: pb.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_UNSPECIFIED},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := assetTypeToProto(tc.assetType); got != tc.want {
				t.Fatalf("assetTypeToProto(%q) = %v, want %v", tc.assetType, got, tc.want)
			}
		})
	}
}

func TestAssetStatusToProto(t *testing.T) {
	tests := []struct {
		name   string
		status catalog.DaggerheartAssetResolutionStatus
		want   pb.DaggerheartAssetStatus
	}{
		{name: "mapped", status: catalog.DaggerheartAssetResolutionStatusMapped, want: pb.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_MAPPED},
		{name: "set default", status: catalog.DaggerheartAssetResolutionStatusSetDefault, want: pb.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_SET_DEFAULT},
		{name: "unknown", status: catalog.DaggerheartAssetResolutionStatus("bogus"), want: pb.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_UNSPECIFIED},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := assetStatusToProto(tc.status); got != tc.want {
				t.Fatalf("assetStatusToProto(%q) = %v, want %v", tc.status, got, tc.want)
			}
		})
	}
}

func TestFallbackString(t *testing.T) {
	if got := fallbackString(" value ", "fallback"); got != " value " {
		t.Fatalf("fallbackString(nonblank) = %q, want original value", got)
	}
	if got := fallbackString("   ", "fallback"); got != "fallback" {
		t.Fatalf("fallbackString(blank) = %q, want fallback", got)
	}
}

func TestResolveDaggerheartAssetMapLocale(t *testing.T) {
	if got := resolveDaggerheartAssetMapLocale(commonv1.Locale_LOCALE_UNSPECIFIED); got != defaultDaggerheartAssetMapLocale {
		t.Fatalf("resolveDaggerheartAssetMapLocale(unspecified) = %v, want %v", got, defaultDaggerheartAssetMapLocale)
	}
	if got := resolveDaggerheartAssetMapLocale(commonv1.Locale_LOCALE_PT_BR); got != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("resolveDaggerheartAssetMapLocale(pt_br) = %v, want %v", got, commonv1.Locale_LOCALE_PT_BR)
	}
	if got := resolveDaggerheartAssetMapLocale(commonv1.Locale(99)); got != defaultDaggerheartAssetMapLocale {
		t.Fatalf("resolveDaggerheartAssetMapLocale(unknown) = %v, want %v", got, defaultDaggerheartAssetMapLocale)
	}
}
