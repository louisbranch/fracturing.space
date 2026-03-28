package testkit

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	gameapp "github.com/louisbranch/fracturing.space/internal/services/game/app"
	inviteapp "github.com/louisbranch/fracturing.space/internal/services/invite/app"
	userhubapp "github.com/louisbranch/fracturing.space/internal/services/userhub/app"
	workerapp "github.com/louisbranch/fracturing.space/internal/services/worker/app"
)

// MeshConfig declares the shared runtime-test fixture shape for one suite.
type MeshConfig struct {
	ContentSeedProfile ContentSeedProfile
}

// Mesh owns one per-suite runtime-test service graph and temp state.
type Mesh struct {
	t    *testing.T
	base string

	authAddr          string
	gameAddr          string
	socialAddr        string
	notificationsAddr string
	inviteAddr        string
	workerAddr        string
	userhubAddr       string
}

// NewMesh prepares per-suite storage/env state for runtime-backed tests.
func NewMesh(t *testing.T, cfg MeshConfig) *Mesh {
	t.Helper()

	base := t.TempDir()
	setenv := func(key, value string) error {
		t.Setenv(key, value)
		return nil
	}
	SetGameDBPaths(t, base, setenv)
	SetAuthDBPath(t, base, setenv)
	SetSocialDBPath(t, base, setenv)
	SetNotificationsDBPath(t, base, setenv)
	SetInviteDBPath(t, base, setenv)

	if cfg.ContentSeedProfile != "" {
		SeedDaggerheartContent(t, cfg.ContentSeedProfile)
	}

	return &Mesh{t: t, base: base}
}

// StartAuthServer boots the auth runtime once for this suite and returns its address.
func (m *Mesh) StartAuthServer() string {
	m.t.Helper()

	if m.authAddr != "" {
		return m.authAddr
	}
	addr, _ := StartAuthServer(m.t)
	m.authAddr = addr
	return m.authAddr
}

// StartGameServer boots the game runtime once for this suite and returns its address.
func (m *Mesh) StartGameServer() string {
	m.t.Helper()

	if m.gameAddr != "" {
		return m.gameAddr
	}

	m.t.Setenv("FRACTURING_SPACE_AUTH_ADDR", m.StartAuthServer())

	ctx, cancel := context.WithCancel(context.Background())
	grpcServer, err := gameapp.NewWithAddr("127.0.0.1:0")
	if err != nil {
		cancel()
		m.t.Fatalf("new game server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(ctx)
	}()

	m.gameAddr = grpcServer.Addr()
	WaitForGRPCHealth(m.t, m.gameAddr)
	m.t.Cleanup(func() {
		cancel()
		select {
		case err := <-serveErr:
			if err != nil {
				m.t.Fatalf("game server error: %v", err)
			}
		case <-time.After(platformgrpc.DefaultGracefulStopTimeout + 5*time.Second):
			m.t.Fatalf("timed out waiting for game server to stop")
		}
	})

	return m.gameAddr
}

// StartSocialServer boots the social runtime once for this suite and returns its address.
func (m *Mesh) StartSocialServer() string {
	m.t.Helper()

	if m.socialAddr != "" {
		return m.socialAddr
	}
	addr, _ := StartSocialServer(m.t)
	m.socialAddr = addr
	return m.socialAddr
}

// StartNotificationsServer boots the notifications runtime once for this suite and returns its address.
func (m *Mesh) StartNotificationsServer() string {
	m.t.Helper()

	if m.notificationsAddr != "" {
		return m.notificationsAddr
	}
	addr, _ := StartNotificationsServer(m.t)
	m.notificationsAddr = addr
	return m.notificationsAddr
}

// StartInviteServer boots the invite runtime once for this suite and returns its address.
func (m *Mesh) StartInviteServer() string {
	m.t.Helper()

	if m.inviteAddr != "" {
		return m.inviteAddr
	}

	ctx, cancel := context.WithCancel(context.Background())
	server, err := inviteapp.NewWithAddr(ctx, "127.0.0.1:0", m.StartGameServer(), m.StartAuthServer())
	if err != nil {
		cancel()
		m.t.Fatalf("new invite server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(ctx)
	}()

	m.inviteAddr = server.Addr()
	WaitForGRPCHealth(m.t, m.inviteAddr)
	m.t.Cleanup(func() {
		cancel()
		select {
		case err := <-serveErr:
			if err != nil {
				m.t.Logf("invite server error: %v", err)
			}
		case <-time.After(5 * time.Second):
			m.t.Logf("timed out waiting for invite server to stop")
		}
	})

	return m.inviteAddr
}

// StartWorkerRuntime boots the worker runtime once for this suite and returns its health address.
func (m *Mesh) StartWorkerRuntime() string {
	m.t.Helper()

	if m.workerAddr != "" {
		return m.workerAddr
	}

	cfg := workerapp.RuntimeConfig{
		Port:              portFromAddress(m.t, pickUnusedAddress(m.t)),
		AuthAddr:          m.StartAuthServer(),
		AIAddr:            pickUnusedAddress(m.t),
		GameAddr:          m.StartGameServer(),
		InviteAddr:        m.StartInviteServer(),
		NotificationsAddr: m.StartNotificationsServer(),
		SocialAddr:        m.StartSocialServer(),
		DBPath:            filepath.Join(m.base, "worker.db"),
		Consumer:          "integration-worker",
		PollInterval:      10 * time.Millisecond,
		LeaseTTL:          250 * time.Millisecond,
		MaxAttempts:       3,
		RetryBackoff:      10 * time.Millisecond,
		RetryMaxDelay:     50 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	runtime, err := workerapp.NewRuntime(ctx, cfg)
	if err != nil {
		cancel()
		m.t.Fatalf("new worker runtime: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- runtime.Serve(ctx)
	}()

	m.workerAddr = runtime.Addr()
	WaitForGRPCHealth(m.t, m.workerAddr)
	m.t.Cleanup(func() {
		cancel()
		select {
		case err := <-serveErr:
			if err != nil {
				m.t.Logf("worker runtime error: %v", err)
			}
		case <-time.After(5 * time.Second):
			m.t.Logf("timed out waiting for worker runtime to stop")
		}
	})

	return m.workerAddr
}

// StartUserHubServer boots the userhub runtime once for this suite and returns its address.
func (m *Mesh) StartUserHubServer(cfg userhubapp.RuntimeConfig) string {
	m.t.Helper()

	if m.userhubAddr != "" {
		return m.userhubAddr
	}

	if cfg.AuthAddr == "" {
		cfg.AuthAddr = m.StartAuthServer()
	}
	if cfg.GameAddr == "" {
		cfg.GameAddr = m.StartGameServer()
	}
	if cfg.InviteAddr == "" {
		cfg.InviteAddr = m.StartInviteServer()
	}
	if cfg.SocialAddr == "" {
		cfg.SocialAddr = m.StartSocialServer()
	}
	if cfg.NotificationsAddr == "" {
		cfg.NotificationsAddr = m.StartNotificationsServer()
	}
	if cfg.StatusAddr == "" {
		cfg.StatusAddr = pickUnusedAddress(m.t)
	}
	if cfg.CacheFreshTTL == 0 {
		cfg.CacheFreshTTL = time.Minute
	}
	if cfg.CacheStaleTTL == 0 {
		cfg.CacheStaleTTL = 5 * time.Minute
	}

	addr, _ := StartUserHubServer(m.t, cfg)
	m.userhubAddr = addr
	return m.userhubAddr
}

func pickUnusedAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("pick unused address: %v", err)
	}
	defer listener.Close()
	return listener.Addr().String()
}

func portFromAddress(t *testing.T, addr string) int {
	t.Helper()

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		t.Fatalf("resolve tcp addr %q: %v", addr, err)
	}
	if tcpAddr.Port <= 0 {
		t.Fatalf("resolved tcp addr %q without port", addr)
	}
	return tcpAddr.Port
}
