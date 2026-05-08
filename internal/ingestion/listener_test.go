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
	"github.com/segmentio/kafka-go"

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
	logs        chan ethtypes.Log
	subErr      chan error
	logSubErr   chan error
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

func (f *fakeClient) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- ethtypes.Log) (ethereum.Subscription, error) {
	if f.logs != nil {
		go func() {
			for l := range f.logs {
				select {
				case ch <- l:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	errCh := f.logSubErr
	if errCh == nil {
		errCh = make(chan error, 1)
	}
	return &fakeSubscription{errCh: errCh}, nil
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
	cfg := types.ChainConfig{ChainID: 8453, Name: "base", RPCWSS: "wss://example/ws", SubscribeMode: "blocks"}
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
	cfg := types.ChainConfig{ChainID: 8453, Name: "base", RPCWSS: "wss://example/ws", SubscribeMode: "blocks"}
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
	cfg := types.ChainConfig{ChainID: 8453, Name: "base", RPCWSS: "wss://example/ws", Confirmations: 2, SubscribeMode: "blocks"}
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

// TestRun_HeadWatchdogFires asserts that with a short HeadTimeout and no
// headers ever delivered, the listener exits its current session with
// the watchdog error and the outer Run loop tears down on ctx cancel.
// This guards against silent WSS death where sub.Err() never fires.
func TestRun_HeadWatchdogFires(t *testing.T) {
	cfg := types.ChainConfig{
		ChainID:     8453,
		Name:        "base",
		RPCWSS:      "wss://example/ws",
		HeadTimeout:   types.Duration(80 * time.Millisecond),
		SubscribeMode: "blocks",
	}
	dec := NewABIDecoder()

	prod, err := NewProducer(ProducerConfig{Brokers: []string{"localhost:9092"}, Topic: "raw_events"})
	if err != nil {
		t.Fatal(err)
	}
	defer prod.Close()

	headers := make(chan *ethtypes.Header)
	subErr := make(chan error, 1)
	fc := &fakeClient{headers: headers, subErr: subErr}

	dialCalls := atomic.Int32{}
	cl := NewChainListener(cfg, dec, prod).WithDialer(func(ctx context.Context, url string) (EthClient, error) {
		dialCalls.Add(1)
		return fc, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- cl.Run(ctx) }()

	// Watchdog should fire (~80ms), trigger a reconnect attempt; with no
	// headers ever arriving, the outer loop keeps retrying until the
	// context expires. Reconnect backoff starts at 1s, so dial #2 lands
	// near t=1.1s. Wait up to 3s to absorb scheduler jitter on CI.
	deadline := time.Now().Add(3 * time.Second)
	for dialCalls.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not exit after cancel")
	}
	if got := dialCalls.Load(); got < 2 {
		t.Errorf("dialCalls = %d, want >= 2 (watchdog should have triggered a reconnect)", got)
	}
}

// TestRun_HeadWatchdogResetsOnHeader asserts that a steady stream of
// headers spaced shorter than HeadTimeout does NOT trigger the
// watchdog. Drives 5 headers spaced at 30ms with a 100ms timeout, then
// cancels.
func TestRun_HeadWatchdogResetsOnHeader(t *testing.T) {
	cfg := types.ChainConfig{
		ChainID:     8453,
		Name:        "base",
		RPCWSS:      "wss://example/ws",
		HeadTimeout:   types.Duration(100 * time.Millisecond),
		SubscribeMode: "blocks",
	}
	dec := NewABIDecoder()

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
			return nil, nil
		},
	}

	dialCalls := atomic.Int32{}
	cl := NewChainListener(cfg, dec, prod).WithDialer(func(ctx context.Context, url string) (EthClient, error) {
		dialCalls.Add(1)
		return fc, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- cl.Run(ctx) }()

	for n := int64(100); n < 105; n++ {
		select {
		case headers <- &ethtypes.Header{Number: big.NewInt(n), Time: 1_700_000_000}:
		case <-time.After(time.Second):
			t.Fatalf("send header %d timed out", n)
		}
		time.Sleep(30 * time.Millisecond)
	}
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not exit after cancel")
	}
	if got := dialCalls.Load(); got != 1 {
		t.Errorf("dialCalls = %d, want 1 (no reconnect — watchdog must reset)", got)
	}
	if got := filterCalls.Load(); got < 5 {
		t.Errorf("filterCalls = %d, want >= 5 (each header drives one)", got)
	}
}

// erroringWriter is a messageWriter test double that fails every
// WriteMessages call with the supplied error.
type erroringWriter struct {
	err   error
	calls atomic.Int32
}

func (w *erroringWriter) WriteMessages(_ context.Context, _ ...kafka.Message) error {
	w.calls.Add(1)
	return w.err
}

func (w *erroringWriter) Close() error { return nil }

// flakyWriter fails the first failsLeft calls and succeeds afterwards.
type flakyWriter struct {
	failsLeft atomic.Int32
	calls     atomic.Int32
}

func (w *flakyWriter) WriteMessages(_ context.Context, _ ...kafka.Message) error {
	w.calls.Add(1)
	if w.failsLeft.Add(-1) >= 0 {
		return errors.New("transient kafka write error")
	}
	return nil
}

func (w *flakyWriter) Close() error { return nil }

// TestRun_KafkaPublishFailFastTriggersErrKafkaUnhealthy asserts that
// repeated kafka publish failures across consecutive blocks bubble
// ErrKafkaUnhealthy up out of Run instead of silently dropping events
// or reconnecting WSS forever.
func TestRun_KafkaPublishFailFastTriggersErrKafkaUnhealthy(t *testing.T) {
	cfg := types.ChainConfig{
		ChainID:       8453,
		Name:          "base",
		RPCWSS:        "wss://example/ws",
		SubscribeMode: "blocks",
		// Confirmations 0 so each head emits immediately; keeps the
		// per-head -> per-publish mapping easy to reason about.
	}

	w := &erroringWriter{err: errors.New("connection refused")}
	prod := newProducerWithWriter(w, "raw_events", map[uint64]string{8453: "base"})

	headers := make(chan *ethtypes.Header, 4)
	subErr := make(chan error, 1)
	fc := &fakeClient{
		headers: headers,
		subErr:  subErr,
		filterFn: func(_ context.Context, _ ethereum.FilterQuery) ([]ethtypes.Log, error) {
			return []ethtypes.Log{fixtureBaseUSDCTransfer}, nil
		},
	}

	cl := NewChainListener(cfg, NewABIDecoder(), prod).
		WithDialer(func(_ context.Context, _ string) (EthClient, error) { return fc, nil }).
		WithKafkaFailThreshold(2)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- cl.Run(ctx) }()

	// Two heads => two publish attempts => streak 2 => ErrKafkaUnhealthy.
	headers <- &ethtypes.Header{Number: big.NewInt(100), Time: 1_700_000_000}
	headers <- &ethtypes.Header{Number: big.NewInt(101), Time: 1_700_000_001}

	select {
	case err := <-done:
		if !errors.Is(err, ErrKafkaUnhealthy) {
			t.Fatalf("Run returned %v, want ErrKafkaUnhealthy", err)
		}
	case <-time.After(2 * time.Second):
		cancel()
		<-done
		t.Fatal("Run did not exit after streak reached threshold")
	}

	if got := w.calls.Load(); got < 2 {
		t.Errorf("writer.calls = %d, want >= 2 (each block should attempt publish before fail-fast)", got)
	}
}

// TestRun_KafkaPublishFailureDoesNotAdvance asserts that a transient
// publish failure leaves lastEmitted unchanged so the next head re-tries
// the missing block in order. After the broker recovers the listener
// catches up without dropping the originally-failing block.
func TestRun_KafkaPublishFailureDoesNotAdvance(t *testing.T) {
	cfg := types.ChainConfig{
		ChainID:       8453,
		Name:          "base",
		RPCWSS:        "wss://example/ws",
		SubscribeMode: "blocks",
	}

	w := &flakyWriter{}
	w.failsLeft.Store(1) // first publish fails, subsequent succeed
	prod := newProducerWithWriter(w, "raw_events", map[uint64]string{8453: "base"})

	headers := make(chan *ethtypes.Header, 4)
	subErr := make(chan error, 1)

	var filterCalls atomic.Int32
	fc := &fakeClient{
		headers: headers,
		subErr:  subErr,
		filterFn: func(_ context.Context, _ ethereum.FilterQuery) ([]ethtypes.Log, error) {
			filterCalls.Add(1)
			return []ethtypes.Log{fixtureBaseUSDCTransfer}, nil
		},
	}

	cl := NewChainListener(cfg, NewABIDecoder(), prod).
		WithDialer(func(_ context.Context, _ string) (EthClient, error) { return fc, nil }).
		WithKafkaFailThreshold(5)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- cl.Run(ctx) }()

	// Head 100 will trigger publish for block 100, which fails. Head
	// 101 should retry block 100 first (no-advance), then publish 101.
	headers <- &ethtypes.Header{Number: big.NewInt(100), Time: 1_700_000_000}

	deadline := time.Now().Add(2 * time.Second)
	for filterCalls.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	headers <- &ethtypes.Header{Number: big.NewInt(101), Time: 1_700_000_001}

	deadline = time.Now().Add(2 * time.Second)
	for w.calls.Load() < 3 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not exit on cancel")
	}

	// Expected sequence: publish(100) fails -> publish(100) retries on
	// next head, succeeds -> publish(101) succeeds. Three writer calls.
	if got := w.calls.Load(); got != 3 {
		t.Errorf("writer.calls = %d, want 3 (fail at 100 + retry 100 + publish 101)", got)
	}
	if got := cl.lastEmitted; got != 101 {
		t.Errorf("lastEmitted = %d, want 101", got)
	}
	if got := cl.kafkaFailStreak; got != 0 {
		t.Errorf("kafkaFailStreak = %d, want 0 after recovery", got)
	}
}

// TestRun_ReturnsErrChainListenerDeadOnExhaustion verifies the Run loop
// wraps its terminal error in ErrChainListenerDead so cmd/indexer can
// escalate to a process-level exit. Without that, a chain whose
// provider rejects every dial would silently drop out of the indexer
// for the lifetime of the process (issue #174).
func TestRun_ReturnsErrChainListenerDeadOnExhaustion(t *testing.T) {
	prevInit, prevMax, prevAttempts := reconnectInitialBackoff, reconnectMaxBackoff, reconnectMaxAttempts
	reconnectInitialBackoff = 5 * time.Millisecond
	reconnectMaxBackoff = 10 * time.Millisecond
	reconnectMaxAttempts = 3
	t.Cleanup(func() {
		reconnectInitialBackoff = prevInit
		reconnectMaxBackoff = prevMax
		reconnectMaxAttempts = prevAttempts
	})

	cfg := types.ChainConfig{ChainID: 8453, Name: "base", RPCWSS: "wss://example/ws", SubscribeMode: "blocks"}
	cl := NewChainListener(cfg, NewABIDecoder(), &Producer{chainNames: map[uint64]string{8453: "base"}}).
		WithDialer(func(_ context.Context, _ string) (EthClient, error) {
			return nil, errors.New("permanent dial failure")
		})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := cl.Run(ctx)
	if err == nil {
		t.Fatalf("Run returned nil, want ErrChainListenerDead")
	}
	if !errors.Is(err, ErrChainListenerDead) {
		t.Fatalf("Run returned %v, want ErrChainListenerDead", err)
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
