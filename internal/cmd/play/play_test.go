package play

import (
	"context"
	"errors"
	"flag"
	"io"
	"reflect"
	"testing"

	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
	gogrpc "google.golang.org/grpc"
	gogrpcmetadata "google.golang.org/grpc/metadata"
)

func TestParseConfigUsesDefaultsAndAppliesFlags(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_PLAY_HTTP_ADDR", "")
	t.Setenv("FRACTURING_SPACE_WEB_HTTP_ADDR", "")
	t.Setenv("FRACTURING_SPACE_AUTH_ADDR", "")
	t.Setenv("FRACTURING_SPACE_GAME_ADDR", "")
	t.Setenv("FRACTURING_SPACE_STATUS_ADDR", "")
	t.Setenv("FRACTURING_SPACE_PLAY_DB_PATH", "")
	t.Setenv("FRACTURING_SPACE_PLAY_UI_DEV_SERVER_URL", "")
	t.Setenv("FRACTURING_SPACE_PLAY_TRUST_FORWARDED_PROTO", "")

	fs := flag.NewFlagSet("play", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{
		"--http-addr=:9100",
		"--db-path=/tmp/play.sqlite",
		"--ui-dev-server-url=http://localhost:5173",
		"--trust-forwarded-proto",
	})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if cfg.HTTPAddr != ":9100" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":9100")
	}
	if cfg.DBPath != "/tmp/play.sqlite" {
		t.Fatalf("DBPath = %q, want %q", cfg.DBPath, "/tmp/play.sqlite")
	}
	if cfg.PlayUIDevServerURL != "http://localhost:5173" {
		t.Fatalf("PlayUIDevServerURL = %q, want %q", cfg.PlayUIDevServerURL, "http://localhost:5173")
	}
	if !cfg.TrustForwardedProto {
		t.Fatal("TrustForwardedProto = false, want true")
	}
	if cfg.WebHTTPAddr != serviceaddr.OrDefaultHTTPAddr("", serviceaddr.ServiceWeb) {
		t.Fatalf("WebHTTPAddr = %q", cfg.WebHTTPAddr)
	}
	if cfg.AuthAddr != serviceaddr.OrDefaultGRPCAddr("", serviceaddr.ServiceAuth) {
		t.Fatalf("AuthAddr = %q", cfg.AuthAddr)
	}
	if cfg.GameAddr != serviceaddr.OrDefaultGRPCAddr("", serviceaddr.ServiceGame) {
		t.Fatalf("GameAddr = %q", cfg.GameAddr)
	}
	if cfg.StatusAddr != serviceaddr.OrDefaultGRPCAddr("", serviceaddr.ServiceStatus) {
		t.Fatalf("StatusAddr = %q", cfg.StatusAddr)
	}
}

func TestOpenRuntimeDependenciesWithClosesPartiallyOpenedResources(t *testing.T) {
	t.Parallel()

	t.Run("game dial failure closes auth", func(t *testing.T) {
		t.Parallel()

		auth := &fakeManagedConn{name: "auth", conn: fakeClientConn{}}
		var gotConfigs []platformgrpc.ManagedConnConfig
		_, _, err := openRuntimeDependenciesWith(context.Background(), Config{
			AuthAddr: "auth:1",
			GameAddr: "game:2",
			DBPath:   "play.sqlite",
		}, runtimeDependencyOpeners{
			openManagedConn: func(_ context.Context, cfg platformgrpc.ManagedConnConfig) (managedConnResource, error) {
				gotConfigs = append(gotConfigs, cfg)
				if cfg.Name == "auth" {
					return auth, nil
				}
				return nil, errors.New("game down")
			},
			openStore: func(string) (transcriptStoreResource, error) {
				t.Fatal("openStore should not be called on game dial failure")
				return nil, nil
			},
		})
		if err == nil || err.Error() != "connect game: game down" {
			t.Fatalf("error = %v", err)
		}
		if auth.closeCalls != 1 {
			t.Fatalf("auth close calls = %d, want 1", auth.closeCalls)
		}
		if len(gotConfigs) != 2 || gotConfigs[0].Name != "auth" || gotConfigs[1].Name != "game" {
			t.Fatalf("managed conn configs = %#v", gotConfigs)
		}
	})

	t.Run("store open failure closes auth and game", func(t *testing.T) {
		t.Parallel()

		auth := &fakeManagedConn{name: "auth", conn: fakeClientConn{}}
		game := &fakeManagedConn{name: "game", conn: fakeClientConn{}}
		_, _, err := openRuntimeDependenciesWith(context.Background(), Config{
			AuthAddr: "auth:1",
			GameAddr: "game:2",
			DBPath:   "play.sqlite",
		}, runtimeDependencyOpeners{
			openManagedConn: func(_ context.Context, cfg platformgrpc.ManagedConnConfig) (managedConnResource, error) {
				if cfg.Name == "auth" {
					return auth, nil
				}
				return game, nil
			},
			openStore: func(string) (transcriptStoreResource, error) {
				return nil, errors.New("disk full")
			},
		})
		if err == nil || err.Error() != "open play transcript store: disk full" {
			t.Fatalf("error = %v", err)
		}
		if auth.closeCalls != 1 || game.closeCalls != 1 {
			t.Fatalf("close calls = auth:%d game:%d, want 1 each", auth.closeCalls, game.closeCalls)
		}
	})
}

func TestOpenRuntimeDependenciesWithSuccessBuildsDependenciesAndClosesIdempotently(t *testing.T) {
	t.Parallel()

	auth := &fakeManagedConn{name: "auth", conn: fakeClientConn{}}
	game := &fakeManagedConn{name: "game", conn: fakeClientConn{}}
	store := &fakeTranscriptStore{}
	var closeOrder []string
	auth.onClose = func() { closeOrder = append(closeOrder, "auth") }
	game.onClose = func() { closeOrder = append(closeOrder, "game") }
	store.onClose = func() { closeOrder = append(closeOrder, "store") }

	resources, deps, err := openRuntimeDependenciesWith(context.Background(), Config{
		AuthAddr: "auth:1",
		GameAddr: "game:2",
		DBPath:   "play.sqlite",
	}, runtimeDependencyOpeners{
		openManagedConn: func(_ context.Context, cfg platformgrpc.ManagedConnConfig) (managedConnResource, error) {
			if cfg.Name == "auth" {
				return auth, nil
			}
			return game, nil
		},
		openStore: func(path string) (transcriptStoreResource, error) {
			if path != "play.sqlite" {
				t.Fatalf("path = %q, want %q", path, "play.sqlite")
			}
			return store, nil
		},
	})
	if err != nil {
		t.Fatalf("openRuntimeDependenciesWith() error = %v", err)
	}
	if deps.Auth == nil || deps.Interaction == nil || deps.Campaign == nil || deps.System == nil || deps.Events == nil {
		t.Fatalf("dependencies = %#v", deps)
	}
	if deps.Transcripts != store {
		t.Fatalf("Transcripts = %#v, want %#v", deps.Transcripts, store)
	}

	if err := resources.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := resources.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if !reflect.DeepEqual(closeOrder, []string{"store", "game", "auth"}) {
		t.Fatalf("close order = %#v, want %#v", closeOrder, []string{"store", "game", "auth"})
	}
	if auth.closeCalls != 1 || game.closeCalls != 1 || store.closeCalls != 1 {
		t.Fatalf("close calls = auth:%d game:%d store:%d, want 1 each", auth.closeCalls, game.closeCalls, store.closeCalls)
	}
}

type fakeManagedConn struct {
	name       string
	conn       fakeClientConn
	closeCalls int
	onClose    func()
}

func (f *fakeManagedConn) ClientConn() gogrpc.ClientConnInterface {
	return f.conn
}

func (f *fakeManagedConn) Close() error {
	f.closeCalls++
	if f.onClose != nil {
		f.onClose()
	}
	return nil
}

type fakeClientConn struct{}

func (fakeClientConn) Invoke(context.Context, string, any, any, ...gogrpc.CallOption) error {
	return nil
}

func (fakeClientConn) NewStream(context.Context, *gogrpc.StreamDesc, string, ...gogrpc.CallOption) (gogrpc.ClientStream, error) {
	return fakeClientStream{}, nil
}

type fakeClientStream struct{}

func (fakeClientStream) Header() (gogrpcmetadata.MD, error) { return nil, nil }
func (fakeClientStream) Trailer() gogrpcmetadata.MD         { return nil }
func (fakeClientStream) CloseSend() error                   { return nil }
func (fakeClientStream) Context() context.Context           { return context.Background() }
func (fakeClientStream) SendMsg(any) error                  { return nil }
func (fakeClientStream) RecvMsg(any) error                  { return io.EOF }

type fakeTranscriptStore struct {
	closeCalls int
	onClose    func()
}

func (f *fakeTranscriptStore) LatestSequence(context.Context, transcript.Scope) (int64, error) {
	return 0, nil
}

func (f *fakeTranscriptStore) AppendMessage(context.Context, transcript.AppendRequest) (transcript.AppendResult, error) {
	return transcript.AppendResult{}, nil
}

func (f *fakeTranscriptStore) HistoryAfter(context.Context, transcript.HistoryAfterQuery) ([]transcript.Message, error) {
	return nil, nil
}

func (f *fakeTranscriptStore) HistoryBefore(context.Context, transcript.HistoryBeforeQuery) ([]transcript.Message, error) {
	return nil, nil
}

func (f *fakeTranscriptStore) Close() error {
	f.closeCalls++
	if f.onClose != nil {
		f.onClose()
	}
	return nil
}
