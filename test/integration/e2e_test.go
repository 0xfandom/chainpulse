//go:build integration

// Package integration runs an end-to-end test against a live ChainPulse
// stack started by docker compose. It publishes a synthetic ERC-20
// Transfer event to Kafka and asserts it propagates through the
// processor -> ClickHouse -> Redis -> API pipeline.
//
// Usage:
//
//	docker compose up -d --build
//	go test -tags integration ./test/integration/...
package integration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/segmentio/kafka-go"

	"github.com/0xfandom/chainpulse/internal/types"
)

// envOrDefault reads env var name or returns def.
func envOrDefault(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

// erc20TransferTopic0 is keccak256("Transfer(address,address,uint256)").
var erc20TransferTopic0 = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))

// addressTopic packs an address into a left-padded 32-byte topic.
func addressTopic(a common.Address) common.Hash {
	var h common.Hash
	copy(h[12:], a.Bytes())
	return h
}

// uint256Bytes encodes n into a 32-byte big-endian buffer.
func uint256Bytes(n uint64) []byte {
	out := make([]byte, 32)
	for i := 7; i >= 0; i-- {
		out[31-i] = byte(n >> (uint(i) * 8))
	}
	return out
}

// randomAddress generates a fresh address per test run so leftover state
// from previous runs doesn't taint results.
func randomAddress(t *testing.T) common.Address {
	t.Helper()
	var a common.Address
	if _, err := rand.Read(a[:]); err != nil {
		t.Fatal(err)
	}
	return a
}

// randomTxHash gives every event a unique (chain, block, log) coordinate.
func randomTxHash(t *testing.T) common.Hash {
	t.Helper()
	var h common.Hash
	if _, err := rand.Read(h[:]); err != nil {
		t.Fatal(err)
	}
	return h
}

func TestE2E_TransferFlowsThroughPipeline(t *testing.T) {
	brokers := strings.Split(envOrDefault("KAFKA_BROKERS", "localhost:9092"), ",")
	topic := envOrDefault("KAFKA_TOPIC_RAW_EVENTS", "raw_events")
	apiURL := envOrDefault("API_URL", "http://localhost:8080")

	from := randomAddress(t)
	to := randomAddress(t)
	token := randomAddress(t)
	txHash := randomTxHash(t)
	blockNumber := uint64(time.Now().Unix())
	const amount uint64 = 1_000_000

	event := types.ChainEvent{
		ChainID:     8453,
		BlockNumber: blockNumber,
		TxHash:      txHash,
		LogIndex:    0,
		Contract:    token,
		EventName:   "Transfer",
		RawTopics: []common.Hash{
			erc20TransferTopic0,
			addressTopic(from),
			addressTopic(to),
		},
		RawData:   uint256Bytes(amount),
		Timestamp: time.Now().UTC(),
	}

	body, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}

	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.Hash{},
	}
	defer writer.Close()

	produceCtx, produceCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer produceCancel()
	if err := writer.WriteMessages(produceCtx, kafka.Message{
		Key:   []byte("8453"),
		Value: body,
	}); err != nil {
		t.Fatalf("write to kafka: %v", err)
	}

	wallet := strings.ToLower(from.Hex())
	endpoint := fmt.Sprintf("%s/v1/tokens/%s/transfers?limit=10", apiURL, strings.ToLower(token.Hex()))
	t.Logf("polling %s for new transfer (token=%s)", endpoint, token.Hex())

	deadline := time.Now().Add(60 * time.Second)
	wantTx := strings.ToLower("0x" + hex.EncodeToString(txHash.Bytes()))
	for time.Now().Before(deadline) {
		if found := pollForTx(t, endpoint, wantTx); found {
			t.Logf("transfer for wallet=%s landed in api", wallet)
			return
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("transfer never appeared in api within 60s; wallet=%s tx=%s", wallet, wantTx)
}

// pollForTx hits the api and returns true if the response contains the
// expected tx hash anywhere in the body.
func pollForTx(t *testing.T, url, wantTx string) bool {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Logf("api unavailable yet: %v", err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	buf := make([]byte, 64*1024)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])
	return strings.Contains(strings.ToLower(body), wantTx)
}
