package testing

import (
	"context"
	"net"
	gotesting "testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// StartServer serves the given server using an in-memory listener and returns
// a client connection to it. The connection is closed and the server stopped
// when the test completes.
func StartServer(tb gotesting.TB, svr *grpc.Server) *grpc.ClientConn {
	tb.Helper()
	listener := bufconn.Listen(1024 * 1024)
	go func() {
		_ = svr.Serve(listener)
	}()
	tb.Cleanup(svr.Stop)
	clientConn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		tb.Fatalf("failed to create client: %v", err)
	}
	tb.Cleanup(func() {
		_ = clientConn.Close()
	})
	return clientConn
}
