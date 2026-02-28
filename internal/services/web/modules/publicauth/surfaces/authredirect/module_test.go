package authredirect

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestModuleIDAndPrefix(t *testing.T) {
	t.Parallel()

	m := NewWithGatewayAndPolicy(nil, requestmeta.SchemePolicy{})
	if got := m.ID(); got != "public-auth-redirect" {
		t.Fatalf("ID() = %q, want %q", got, "public-auth-redirect")
	}
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	if got := mount.Prefix; got != routepath.AuthPrefix {
		t.Fatalf("prefix = %q, want %q", got, routepath.AuthPrefix)
	}
}
