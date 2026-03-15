package httptransport

import (
	"context"
	"crypto/tls"
	"net"
	"testing"
	"time"
)

func TestStartUsesTLSListenerWhenConfigured(t *testing.T) {
	transport := NewHTTPTransport("127.0.0.1:0")
	transport.SetTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12})

	origListenTCP := listenTCP
	origNewTLSListener := newTLSListener
	defer func() {
		listenTCP = origListenTCP
		newTLSListener = origNewTLSListener
	}()

	tcpCalled := false
	tlsCalled := false
	listenDone := make(chan struct{}, 1)
	tlsDone := make(chan struct{}, 1)

	listenTCP = func(network, address string) (net.Listener, error) {
		tcpCalled = true
		listener, err := origListenTCP(network, address)
		if err == nil {
			listenDone <- struct{}{}
		}
		return listener, err
	}
	newTLSListener = func(l net.Listener, cfg *tls.Config) net.Listener {
		tlsCalled = true
		tlsDone <- struct{}{}
		return origNewTLSListener(l, cfg)
	}

	ctx, cancel := context.WithCancel(context.Background())
	startErr := make(chan error, 1)

	go func() {
		startErr <- transport.Start(ctx)
	}()

	select {
	case <-listenDone:
		// listener created and wrapped as expected
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected transport start to open listener")
	}

	select {
	case <-tlsDone:
		// TLS wrapper created as expected
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected TLS listener to be used when TLS config is configured")
	}

	cancel()
	select {
	case err := <-startErr:
		if err != nil {
			t.Fatalf("expected Start to return nil on context cancel, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("expected Start to exit after cancel")
	}

	if !tcpCalled {
		t.Fatal("expected net listener to be used for HTTP transport start")
	}
	if !tlsCalled {
		t.Fatal("expected TLS listener to be used when TLS config is configured")
	}
}
