package dashboard

import "testing"

func TestNewService(t *testing.T) {
	t.Parallel()

	_ = newService()
}
