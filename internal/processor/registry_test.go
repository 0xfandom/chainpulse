package processor

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xfandom/chainpulse/internal/types"
)

type fakeDecoder struct {
	name string
	sigs []common.Hash
}

func (f *fakeDecoder) Name() string { return f.name }
func (f *fakeDecoder) CanDecode(sig common.Hash) bool {
	for _, s := range f.sigs {
		if s == sig {
			return true
		}
	}
	return false
}
func (f *fakeDecoder) SupportedEvents() []common.Hash { return f.sigs }
func (f *fakeDecoder) Decode(e *types.ChainEvent) (*types.DecodedEvent, error) {
	return &types.DecodedEvent{
		ChainEvent: *e,
		Protocol:   f.name,
		EventType:  "fake",
		Params:     map[string]interface{}{"sig": e.RawTopics[0].Hex()},
	}, nil
}

var (
	sigA = common.HexToHash("0xaaaa000000000000000000000000000000000000000000000000000000000001")
	sigB = common.HexToHash("0xbbbb000000000000000000000000000000000000000000000000000000000002")
	sigC = common.HexToHash("0xcccc000000000000000000000000000000000000000000000000000000000003")
)

func TestRegistry_RegisterAndMatch(t *testing.T) {
	r := NewRegistry()

	dA := &fakeDecoder{name: "alpha", sigs: []common.Hash{sigA}}
	dB := &fakeDecoder{name: "beta", sigs: []common.Hash{sigB}}

	if err := r.Register(dA); err != nil {
		t.Fatalf("register alpha: %v", err)
	}
	if err := r.Register(dB); err != nil {
		t.Fatalf("register beta: %v", err)
	}

	if got := r.Match(sigA); got != dA {
		t.Errorf("Match(sigA) = %v, want alpha", got)
	}
	if got := r.Match(sigB); got != dB {
		t.Errorf("Match(sigB) = %v, want beta", got)
	}
	if got := r.Match(sigC); got != nil {
		t.Errorf("Match(sigC) = %v, want nil", got)
	}

	names := r.Names()
	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("Names = %v", names)
	}
}

func TestRegistry_DuplicateName(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(&fakeDecoder{name: "x", sigs: []common.Hash{sigA}}); err != nil {
		t.Fatal(err)
	}
	err := r.Register(&fakeDecoder{name: "x", sigs: []common.Hash{sigB}})
	if err == nil {
		t.Fatal("expected duplicate-name error")
	}
}

func TestRegistry_RegisterRejectsNilAndEmptyName(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(nil); err == nil {
		t.Error("expected error for nil decoder")
	}
	if err := r.Register(&fakeDecoder{name: "", sigs: []common.Hash{sigA}}); err == nil {
		t.Error("expected error for empty Name")
	}
}

func TestRegistry_Decode_Success(t *testing.T) {
	r := NewRegistry()
	d := &fakeDecoder{name: "alpha", sigs: []common.Hash{sigA}}
	if err := r.Register(d); err != nil {
		t.Fatal(err)
	}
	out, err := r.Decode(&types.ChainEvent{RawTopics: []common.Hash{sigA}})
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if out.Protocol != "alpha" {
		t.Errorf("Protocol = %q", out.Protocol)
	}
}

func TestRegistry_Decode_EmptyTopics(t *testing.T) {
	r := NewRegistry()
	_, err := r.Decode(&types.ChainEvent{})
	if !errors.Is(err, ErrEmptyTopics) {
		t.Errorf("err = %v, want ErrEmptyTopics", err)
	}
}

func TestRegistry_Decode_NoDecoder(t *testing.T) {
	r := NewRegistry()
	_, err := r.Decode(&types.ChainEvent{RawTopics: []common.Hash{sigC}})
	if !errors.Is(err, ErrNoDecoder) {
		t.Errorf("err = %v, want ErrNoDecoder", err)
	}
	want := fmt.Sprintf("%s", sigC.Hex())
	if err.Error() == "" || !contains(err.Error(), want) {
		t.Errorf("err message %q should contain %q", err.Error(), want)
	}
}

func TestRegistry_Decode_NilEvent(t *testing.T) {
	r := NewRegistry()
	_, err := r.Decode(nil)
	if err == nil {
		t.Error("expected error for nil event")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
