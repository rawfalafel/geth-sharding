package types

import (
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestEqualBlockHash(t *testing.T) {
	b1, err1 := NewGenesisBlock()
	b2, err2 := NewGenesisBlock()
	if err1 != nil || err2 != nil {
		t.Fatalf("failed to instantiate block: %v, %v", err1, err2)
	}

	b1Hash := b1.Hash()
	b2Hash := b2.Hash()
	if err1 != nil || err2 != nil {
		t.Fatalf("failed to hash: %v, %v", err1, err2)
	}

	if b1Hash != b2Hash {
		t.Fatalf("hash are not equal: %x", b1Hash)
	}
}

func TestDifferentBlockHash(t *testing.T) {
	b1, err1 := newTestBlock(&pb.BeaconBlock{
		SlotNumber: 1,
	})
	b2, err2 := newTestBlock(&pb.BeaconBlock{
		SlotNumber: 2,
	})
	if err1 != nil && err2 != nil {
		t.Fatalf("failed to instantiate new block: %v, %v", err1, err2)
	}

	if b1.Hash() == b2.Hash() {
		t.Fatalf("hashes are equal: %x, %x", b1.Hash(), b2.Hash())
	}
}

// NewTestBlock is a helper method to create blocks with valid defaults.
// For a generic block, use NewBlock(t, nil).
func NewTestBlock(t *testing.T, b *pb.BeaconBlock) *Block {
	if b == nil {
		b = &pb.BeaconBlock{}
	}
	if b.ActiveStateHash == nil {
		b.ActiveStateHash = make([]byte, 32)
	}
	if b.CrystallizedStateHash == nil {
		b.CrystallizedStateHash = make([]byte, 32)
	}
	if b.ParentHash == nil {
		b.ParentHash = make([]byte, 32)
	}

	blk, err := NewBlock(b)
	if err != nil {
		t.Fatalf("Failed to instantiate block: %v", err)
	}

	return blk
}
