package engine

import "testing"

func TestValidateCoreRejectionCodeUniqueness_NoCollisions(t *testing.T) {
	if err := ValidateCoreRejectionCodeUniqueness(); err != nil {
		t.Fatalf("rejection code collision detected: %v", err)
	}
}

func TestValidateCoreRejectionCodeUniqueness_AllDomainsExportCodes(t *testing.T) {
	for _, domain := range CoreDomains() {
		if domain.RejectionCodes == nil {
			t.Errorf("core domain %s does not export RejectionCodes()", domain.Name())
			continue
		}
		codes := domain.RejectionCodes()
		if len(codes) == 0 {
			t.Errorf("core domain %s exports empty RejectionCodes()", domain.Name())
		}
	}
}
