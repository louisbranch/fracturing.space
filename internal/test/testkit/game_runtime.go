package testkit

import "testing"

// GameRuntimeConfig controls common auth+game runtime bootstrap for tests.
type GameRuntimeConfig struct {
	ContentSeedProfile ContentSeedProfile
	JoinGrantIssuer    string
	JoinGrantAudience  string
}

// GameRuntime exposes the shared auth/game runtime graph for tests.
type GameRuntime struct {
	Mesh     *Mesh
	AuthAddr string
	GameAddr string
}

// StartGameRuntime boots the shared auth+game test graph with standard grants.
func StartGameRuntime(t *testing.T, cfg GameRuntimeConfig) *GameRuntime {
	t.Helper()

	mesh := NewMesh(t, MeshConfig{
		ContentSeedProfile: cfg.ContentSeedProfile,
	})
	SetJoinGrantEnv(t, cfg.JoinGrantIssuer, cfg.JoinGrantAudience)
	SetAISessionGrantEnv(t)

	authAddr := mesh.StartAuthServer()
	gameAddr := mesh.StartGameServer()

	return &GameRuntime{
		Mesh:     mesh,
		AuthAddr: authAddr,
		GameAddr: gameAddr,
	}
}
