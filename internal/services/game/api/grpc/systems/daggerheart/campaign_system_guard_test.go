package daggerheart

import (
	"testing"

	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCampaignSupportsDaggerheart(t *testing.T) {
	t.Run("daggerheart", func(t *testing.T) {
		record := storage.CampaignRecord{System: bridge.SystemIDDaggerheart}
		if !CampaignSupportsDaggerheart(record) {
			t.Fatal("expected daggerheart campaign to be supported")
		}
	})

	t.Run("unspecified", func(t *testing.T) {
		record := storage.CampaignRecord{System: bridge.SystemIDUnspecified}
		if CampaignSupportsDaggerheart(record) {
			t.Fatal("expected unspecified campaign system to be unsupported")
		}
	})
}

func TestRequireDaggerheartSystem(t *testing.T) {
	record := storage.CampaignRecord{System: bridge.SystemIDUnspecified}
	err := RequireDaggerheartSystem(record, "unsupported system")
	if err == nil {
		t.Fatal("expected failed precondition error")
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
	if got := status.Convert(err).Message(); got != "unsupported system" {
		t.Fatalf("message = %q, want %q", got, "unsupported system")
	}
}

func TestRequireDaggerheartSystemf(t *testing.T) {
	record := storage.CampaignRecord{System: bridge.SystemIDUnspecified}
	err := RequireDaggerheartSystemf(record, "unsupported %s", "operation")
	if err == nil {
		t.Fatal("expected failed precondition error")
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
	if got := status.Convert(err).Message(); got != "unsupported operation" {
		t.Fatalf("message = %q, want %q", got, "unsupported operation")
	}
}
