package ingestion

import (
	"context"
	"errors"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/0xfandom/chainpulse/internal/types"
)

// fakeSubscription implements ethereum.Subscription for tests.
type fakeSubscription struct {
	errCh chan error
}

func (f *fakeSubscription) Err() <-chan error { return f.errCh }
func (f *fakeSubscription) Unsubscribe()      {}

// fakeClient implements EthClient.
type fakeClient struct {
	headers     chan *ethtypes.Header
	subErr      chan error
	subscribeFn func(ctx context.Context, ch chan<- *ethtypes.Header) (ethereum.Subscription, error)
	filterFn    func(ctx context.Context, q ethereum.FilterQuery) ([]ethtypes.Log, error)
	headerFn    func(ctx context.Context, n *big.Int) (*ethtypes.Header, error)
	closed      bool
}

func (f *fakeClient) SubscribeNewHead(ctx context.Context, ch chan<- *ethtypes.Header) (ethereum.Subscription, error) {
	if f.subscribeFn != nil {
		return f.subscribeFn(ctx, ch)
	}
	go func() {
		for h := range f.headers {
			select {
			case ch <- h:
			case <-ctx.Done():
				return
			}
		}
	}()
	return &fakeSubscription{errCh: f.subErr}, nil
}

func (f *fakeClient) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]ethtypes.Log, error) {
	if f.filterFn != nil {
		return f.filterFn(ctx, q)
	}
	return nil, nil
}

func (f *fakeClient) HeaderByNumber(ctx context.Context, n *big.Int) (*ethtypes.Header, error) {
	if f.headerFn != nil {
		return f.headerFn(ctx, n)
	}
	return &ethtypes.Header{Number: new(big.Int).Set(n), Time: 1_700_000_000}, nil
}

func (f *fakeClient) Close() { f.closed = true }

func TestRun_DialFailureReconnects(t *testing.T) {
	cfg := types.ChainConfig{ChainID: 8453, Name: "base", RPCWSS: "wss://example/ws"}
	cl := NewChainListener(cfg, NewABIDecoder(), &Producer{chainNames: map[uint64]string{8453: "base"}})

	dialCalls := 0
	cl.WithDialer(func(ctx context.Context, url string) (EthClient, error) {
		dialCalls++
		return nil, errors.New("dial failed")
	})

	// Override backoff for fast test by cancelling after one short attempt.
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	_ = cl.Run(ctx)
	if dialCalls < 1 {
		t.Errorf("dialer should have been called at least once, got %d", dialCalls)
	}
}

func TestRun_ProcessesBlocksThenContextCancel(t *testing.T) {
	cfg := types.ChainConfig{ChainID: 8453, Name: "base", RPCWSS: "wss://example/ws"}
	dec := NewABIDecoder()

	// Producer with no real broker; we won't call Publish because the test
	// fake returns no logs. (Empty FilterLogs => no Publish.)
	prod, err := NewProducer(ProducerConfig{Brokers: []string{"localhost:9092"}, Topic: "raw_events"})
	if err != nil {
		t.Fatal(err)
	}
	defer prod.Close()

	headers := make(chan *ethtypes.Header, 1)
	subErr := make(chan error, 1)

	var filterCalls atomic.Int32
	fc := &fakeClient{
		headers: headers,
		subErr:  subErr,
		filterFn: func(ctx context.Context, q ethereum.FilterQuery) ([]ethtypes.Log, error) {
			filterCalls.Add(1)
			if q.FromBlock.Cmp(big.NewInt(123)) != 0 {
				t.Errorf("FromBlock = %v, want 123", q.FromBlock)
			}
			return []ethtypes.Log{}, nil
		},
	}

	cl := NewChainListener(cfg, dec, prod).WithDialer(func(ctx context.Context, url string) (EthClient, error) {
		return fc, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- cl.Run(ctx) }()

	headers <- &ethtypes.Header{Number: big.NewInt(123), Time: 1_700_000_000}

	// Allow processBlock to run, then cancel.
	deadline := time.Now().Add(2 * time.Second)
	for filterCalls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not exit on cancel")
	}
	if got := filterCalls.Load(); got != 1 {
		t.Errorf("filterCalls = %d, want 1", got)
	}
}

// TestRun_ConfirmationsLag asserts that with Confirmations=2 a head at
// block 100 triggers a FilterLogs scoped to block 98 (the safe head),
// not 100. Demonstrates the reorg-safety lag.
func TestRun_ConfirmationsLag(t *testing.T) {
	cfg := types.ChainConfig{ChainID: 8453, Name: "base", RPCWSS: "wss://example/ws", Confirmations: 2}
	dec := NewABIDecoder()

	prod, err := NewProducer(ProducerConfig{Brokers: []string{"localhost:9092"}, Topic: "raw_events"})
	if err != nil {
		t.Fatal(err)
	}
	defer prod.Close()

	headers := make(chan *ethtypes.Header, 1)
	subErr := make(chan error, 1)

	var filterCalls atomic.Int32
	var filterFromBlock atomic.Int64
	fc := &fakeClient{
		headers: headers,
		subErr:  subErr,
		filterFn: func(ctx context.Context, q ethereum.FilterQuery) ([]ethtypes.Log, error) {
			filterCalls.Add(1)
			filterFromBlock.Store(q.FromBlock.Int64())
			return []ethtypes.Log{}, nil
		},
		headerFn: func(ctx context.Context, n *big.Int) (*ethtypes.Header, error) {
			return &ethtypes.Header{Number: new(big.Int).Set(n), Time: 1_700_000_000}, nil
		},
	}

	cl := NewChainListener(cfg, dec, prod).WithDialer(func(ctx context.Context, url string) (EthClient, error) {
		return fc, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- cl.Run(ctx) }()

	headers <- &ethtypes.Header{Number: big.NewInt(100), Time: 1_700_000_000}

	deadline := time.Now().Add(2 * time.Second)
	for filterCalls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not exit on cancel")
	}
	if got := filterFromBlock.Load(); got != 98 {
		t.Errorf("FromBlock = %d, want 98 (head 100 - confirmations 2)", got)
	}
}

func TestParseContracts(t *testing.T) {
	in := []string{"0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", "", "0x0000000000000000000000000000000000000001"}
	out := parseContracts(in)
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2", len(out))
	}
	if out[0] != common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913") {
		t.Errorf("addr 0 = %s", out[0].Hex())
	}
}
