package processor

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xfandom/chainpulse/internal/types"
)

// Registry holds an ordered set of ProtocolDecoders. Match returns the
// first decoder whose CanDecode returns true, so insertion order is the
// dispatch priority.
type Registry struct {
	decoders []ProtocolDecoder
	byName   map[string]struct{}
}

// NewRegistry constructs an empty Registry.
func NewRegistry() *Registry {
	return &Registry{byName: map[string]struct{}{}}
}

// Register adds a decoder to the registry. Returns an error if the
// decoder's Name has already been registered.
func (r *Registry) Register(d ProtocolDecoder) error {
	if d == nil {
		return fmt.Errorf("registry: nil decoder")
	}
	name := d.Name()
	if name == "" {
		return fmt.Errorf("registry: decoder has empty Name")
	}
	if _, exists := r.byName[name]; exists {
		return fmt.Errorf("registry: decoder %q already registered", name)
	}
	r.byName[name] = struct{}{}
	r.decoders = append(r.decoders, d)
	return nil
}

// Match returns the first decoder whose CanDecode reports true for sig,
// or nil if none match.
func (r *Registry) Match(sig common.Hash) ProtocolDecoder {
	for _, d := range r.decoders {
		if d.CanDecode(sig) {
			return d
		}
	}
	return nil
}

// Decode looks up the right decoder by event.RawTopics[0] and dispatches.
// Returns ErrEmptyTopics if the event has no topics, or ErrNoDecoder
// (wrapped with the signature) if no decoder claims it.
func (r *Registry) Decode(event *types.ChainEvent) (*types.DecodedEvent, error) {
	if event == nil {
		return nil, fmt.Errorf("registry: nil event")
	}
	if len(event.RawTopics) == 0 {
		return nil, ErrEmptyTopics
	}
	sig := event.RawTopics[0]
	d := r.Match(sig)
	if d == nil {
		return nil, fmt.Errorf("%w: %s", ErrNoDecoder, sig.Hex())
	}
	return d.Decode(event)
}

// Names returns the registered decoder names in registration order. For
// observability and tests.
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.decoders))
	for _, d := range r.decoders {
		out = append(out, d.Name())
	}
	return out
}
