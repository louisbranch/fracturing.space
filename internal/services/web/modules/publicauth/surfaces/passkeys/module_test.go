package passkeys

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func TestModuleIDAndPrefix(t *testing.T) {
	t.Parallel()

	m := NewWithGatewayAndPolicy(nil, requestmeta.SchemePolicy{})
	if got := m.ID(); got != "public-passkeys" {
		t.Fatalf("ID() = %q, want %q", got, "public-passkeys")
	}
	mount, err := m.Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	if got := mount.Prefix; got != routepath.PasskeysPrefix {
		t.Fatalf("prefix = %q, want %q", got, routepath.PasskeysPrefix)
	}
}
