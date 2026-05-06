// Package processor implements the Day-2 read side of the pipeline:
// Kafka consumer, protocol decoders, position aggregator, and storage
// writers. Adding a new protocol means implementing ProtocolDecoder and
// registering it via Registry.Register — nothing else changes.
package processor

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xfandom/chainpulse/internal/types"
)

// ProtocolDecoder is the interface every protocol-specific decoder
// implements. Implementations live under internal/processor/protocols.
type ProtocolDecoder interface {
	// Name returns the protocol identifier, e.g. "uniswap_v3", "aave_v3",
	// "erc20". Used as the Protocol field on DecodedEvent and as a label
	// value on processor metrics.
	Name() string

	// CanDecode reports whether the decoder handles the given event
	// signature (Topics[0] of a log).
	CanDecode(eventSig common.Hash) bool

	// Decode transforms a raw ChainEvent into a protocol-specific
	// DecodedEvent. Returning an error signals the log was claimed by
	// CanDecode but could not be decoded (malformed log) — callers should
	// log and skip rather than retry.
	Decode(event *types.ChainEvent) (*types.DecodedEvent, error)

	// SupportedEvents returns every signature the decoder handles. Used
	// for documentation / introspection; the routing path uses CanDecode.
	SupportedEvents() []common.Hash
}

// Sentinel errors surfaced by the registry.
var (
	// ErrNoDecoder means no registered decoder handles the event's Topics[0].
	ErrNoDecoder = errors.New("no decoder for event signature")

	// ErrEmptyTopics is returned when a ChainEvent has no Topics — usually
	// indicates an unfiltered anonymous event.
	ErrEmptyTopics = errors.New("event has no topics")
)
