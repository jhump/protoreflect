package testing

import (
	"context"
	"net"
	gotesting "testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// ServerOption customizes the behavior of StartServer.
type ServerOption func(*serverConfig)

// WithListenerWrapper returns an option that wraps the server's listener, such
// as to observe or tamper with accepted connections.
func WithListenerWrapper(wrap func(net.Listener) net.Listener) ServerOption {
	return func(cfg *serverConfig) {
		cfg.wrapListener = wrap
	}
}

// WithDialOptions returns an option that adds the given options to those used
// to create the client connection.
func WithDialOptions(opts ...grpc.DialOption) ServerOption {
	return func(cfg *serverConfig) {
		cfg.dialOpts = append(cfg.dialOpts, opts...)
	}
}

type serverConfig struct {
	wrapListener func(net.Listener) net.Listener
	dialOpts     []grpc.DialOption
}

// StartServer serves the given server using an in-memory listener and returns
// a client connection to it. The connection is closed and the server stopped
// when the test completes.
func StartServer(tb gotesting.TB, svr *grpc.Server, opts ...ServerOption) *grpc.ClientConn {
	tb.Helper()
	var cfg serverConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	bufListener := bufconn.Listen(1024 * 1024)
	var listener net.Listener = bufListener
	if cfg.wrapListener != nil {
		listener = cfg.wrapListener(bufListener)
	}
	go func() {
		_ = svr.Serve(listener)
	}()
	tb.Cleanup(svr.Stop)
	dialOpts := append([]grpc.DialOption{
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return bufListener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}, cfg.dialOpts...)
	clientConn, err := grpc.NewClient("passthrough:///bufconn", dialOpts...)
	if err != nil {
		tb.Fatalf("failed to create client: %v", err)
	}
	tb.Cleanup(func() {
		_ = clientConn.Close()
	})
	return clientConn
}
