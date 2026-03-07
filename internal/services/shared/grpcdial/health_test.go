package grpcdial

import (
	"errors"
	"strings"
	"testing"

	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
)

func TestNormalizeDialError_MapsHealthStage(t *testing.T) {
	root := errors.New("health timeout")
	err := NormalizeDialError("status", "status:8093", &platformgrpc.DialError{
		Stage: platformgrpc.DialStageHealth,
		Err:   root,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "status gRPC health check failed for status:8093") {
		t.Fatalf("unexpected message: %q", err.Error())
	}
	if !errors.Is(err, root) {
		t.Fatal("expected wrapped root error")
	}
}

func TestNormalizeDialError_MapsConnectStage(t *testing.T) {
	root := errors.New("connect refused")
	err := NormalizeDialError("social", "social:8090", &platformgrpc.DialError{
		Stage: platformgrpc.DialStageConnect,
		Err:   root,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "dial social gRPC social:8090") {
		t.Fatalf("unexpected message: %q", err.Error())
	}
	if !errors.Is(err, root) {
		t.Fatal("expected wrapped root error")
	}
}

func TestNormalizeDialError_MapsGenericError(t *testing.T) {
	root := errors.New("unexpected")
	err := NormalizeDialError("auth", "auth:8083", root)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "dial auth gRPC auth:8083") {
		t.Fatalf("unexpected message: %q", err.Error())
	}
	if !errors.Is(err, root) {
		t.Fatal("expected wrapped root error")
	}
}
