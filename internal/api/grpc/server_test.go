package grpcsrv

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/0xfandom/chainpulse/internal/api/grpc/pb"
	"github.com/0xfandom/chainpulse/internal/api/store"
)

type fakeRows struct{}

func (f *fakeRows) Next() bool                       { return false }
func (f *fakeRows) Scan(...any) error                { return nil }
func (f *fakeRows) ScanStruct(any) error             { return nil }
func (f *fakeRows) ColumnTypes() []driver.ColumnType { return nil }
func (f *fakeRows) Totals(...any) error              { return nil }
func (f *fakeRows) Columns() []string                { return nil }
func (f *fakeRows) Close() error                     { return nil }
func (f *fakeRows) Err() error                       { return nil }
func (f *fakeRows) HasData() bool                    { return false }

type fakeQueryConn struct{}

func (f *fakeQueryConn) Query(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
	return &fakeRows{}, nil
}
func (f *fakeQueryConn) Ping(context.Context) error { return nil }
func (f *fakeQueryConn) Close() error               { return nil }

func newServer(t *testing.T) *Server {
	t.Helper()
	rs := store.NewReadStore(&fakeQueryConn{})
	return New(rs, nil)
}

func bufDialer(srv *Server) (*grpc.ClientConn, func()) {
	lis := bufconn.Listen(1 << 20)
	go func() { _ = srv.ServeListener(lis) }()
	conn, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}
	return conn, func() {
		_ = conn.Close()
		srv.GracefulStop()
		_ = lis.Close()
	}
}

func TestGRPC_GetWalletPositionsRoundtrip(t *testing.T) {
	srv := newServer(t)
	conn, cleanup := bufDialer(srv)
	defer cleanup()

	c := pb.NewWalletClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := c.GetWalletPositions(ctx, &pb.GetWalletPositionsRequest{
		Wallet: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	if err != nil {
		t.Fatalf("rpc: %v", err)
	}
	if resp == nil {
		t.Fatal("nil response")
	}
	if len(resp.Positions) != 0 {
		t.Errorf("positions = %d, want 0", len(resp.Positions))
	}
}

func TestGRPC_GetProtocolStatsRejectsUnknown(t *testing.T) {
	srv := newServer(t)
	conn, cleanup := bufDialer(srv)
	defer cleanup()

	c := pb.NewProtocolClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := c.GetProtocolStats(ctx, &pb.GetProtocolStatsRequest{Protocol: "morpho"}); err == nil {
		t.Error("expected error for unknown protocol")
	}
}

func TestGRPC_GetWalletBalancesRequiresChain(t *testing.T) {
	srv := newServer(t)
	conn, cleanup := bufDialer(srv)
	defer cleanup()

	c := pb.NewWalletClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := c.GetWalletBalances(ctx, &pb.GetWalletBalancesRequest{
		Wallet: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	if err == nil {
		t.Error("expected error for missing chain_id")
	}
}
