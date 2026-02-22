package server

import (
	"errors"
	"net"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

type trackedListener struct {
	closed bool
}

func (t *trackedListener) Accept() (net.Conn, error) {
	return nil, errors.New("accept not supported in test listener")
}

func (t *trackedListener) Close() error {
	t.closed = true
	return nil
}

func (t *trackedListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
}

func TestNormalizeServerBootstrapConfigDefaults(t *testing.T) {
	cfg := normalizeServerBootstrapConfig(serverBootstrapConfig{})
	if cfg.loadEnv == nil {
		t.Fatal("expected default loadEnv")
	}
	if cfg.listen == nil {
		t.Fatal("expected default listen")
	}
	if cfg.openStorageBundle == nil {
		t.Fatal("expected default openStorageBundle")
	}
	if cfg.configureDomain == nil {
		t.Fatal("expected default configureDomain")
	}
	if cfg.buildSystemRegistry == nil {
		t.Fatal("expected default buildSystemRegistry")
	}
	if cfg.validateSystemRegistration == nil {
		t.Fatal("expected default validateSystemRegistration")
	}
	if cfg.dialAuthGRPC == nil {
		t.Fatal("expected default dialAuthGRPC")
	}
	if cfg.newGRPCServer == nil {
		t.Fatal("expected default newGRPCServer")
	}
	if cfg.newHealthServer == nil {
		t.Fatal("expected default newHealthServer")
	}
	if cfg.resolveProjectionApplyModes == nil {
		t.Fatal("expected default resolveProjectionApplyModes")
	}
	if cfg.buildProjectionRegistries == nil {
		t.Fatal("expected default buildProjectionRegistries")
	}
	if cfg.buildProjectionApplyOutboxApply == nil {
		t.Fatal("expected default buildProjectionApplyOutboxApply")
	}
}

func TestServerBootstrapListensAndClosesOnOpenStorageFailure(t *testing.T) {
	rawListener := &trackedListener{}
	bootstrap := newServerBootstrapWithConfig(serverBootstrapConfig{
		loadEnv: func() serverEnv {
			return serverEnv{}
		},
		listen: func(_ string, _ string) (net.Listener, error) {
			return rawListener, nil
		},
		openStorageBundle: storageBundleOpenerFunc(func(_ serverEnv, _ *event.Registry) (*storageBundle, error) {
			return nil, errors.New("unable to open storage")
		}),
	})

	_, err := bootstrap.NewWithAddr(":0")
	if err == nil {
		t.Fatal("expected bootstrap to fail on storage")
	}
	if !rawListener.closed {
		t.Fatal("expected listener to be closed when storage bootstrap fails")
	}
}
