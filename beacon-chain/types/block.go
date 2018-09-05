// Package types defines the essential types used throughout the beacon-chain.
package types

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"golang.org/x/crypto/blake2b"
)

// Block defines a beacon chain core primitive.
type Block struct {
	block *pb.BeaconBlock
	hash  [32]byte
	enc   []byte
}

// NewBlockFromEncoding accepts the encoding pb.BeaconBlock and returns a new Block.
func NewBlockFromEncoding(enc []byte) (*Block, error) {
	b := &pb.BeaconBlock{}
	if err := proto.Unmarshal(enc, b); err != nil {
		return nil, fmt.Errorf("proto.Unmarhsal failed for encoding %x", enc)
	}

	h := blake2b.Sum256(enc)

	return &Block{
		block: b,
		hash:  h,
		enc:   enc,
	}, nil
}

// NewBlock explicitly sets the data field of a block.
// Return block with default fields if data is nil.
func NewBlock(block *pb.BeaconBlock) (*Block, error) {
	if block == nil {
		block = &pb.BeaconBlock{
			ParentHash:            []byte{0},
			SlotNumber:            0,
			RandaoReveal:          []byte{0},
			Attestations:          []*pb.AttestationRecord{},
			PowChainRef:           []byte{0},
			ActiveStateHash:       []byte{0},
			CrystallizedStateHash: []byte{0},
			Timestamp:             ptypes.TimestampNow(),
		}
	}

	enc, err := proto.Marshal(block)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal block: %v", err)
	}

	return &Block{
		block: block,
		hash:  blake2b.Sum256(enc),
		enc:   enc,
	}, nil
}

// NewGenesisBlock returns the canonical, genesis block for the beacon chain protocol.
//
// TODO: Add more default fields.
func NewGenesisBlock() (*Block, error) {
	protoGenesis, err := ptypes.TimestampProto(time.Unix(0, 0))
	if err != nil {
		return nil, err
	}
	return NewBlock(&pb.BeaconBlock{
		Timestamp:  protoGenesis,
		ParentHash: []byte{},
	})
}

// Proto returns the underlying protobuf data within a block primitive.
func (b *Block) Proto() *pb.BeaconBlock {
	return b.block
}

// Marshal encodes block object into the wire format.
func (b *Block) Marshal() []byte {
	return b.enc
}

// Hash generates the blake2b hash of the block
func (b *Block) Hash() [32]byte {
	return b.hash
}

// ParentHash corresponding to parent beacon block.
func (b *Block) ParentHash() [32]byte {
	var h [32]byte
	copy(h[:], b.block.ParentHash)
	return h
}

// SlotNumber of the beacon block.
func (b *Block) SlotNumber() uint64 {
	return b.block.SlotNumber
}

// PowChainRef returns a keccak256 hash corresponding to a PoW chain block.
func (b *Block) PowChainRef() common.Hash {
	return common.BytesToHash(b.block.PowChainRef)
}

// RandaoReveal returns the blake2b randao hash.
func (b *Block) RandaoReveal() [32]byte {
	var h [32]byte
	copy(h[:], b.block.RandaoReveal)
	return h
}

// ActiveStateHash returns the active state hash.
func (b *Block) ActiveStateHash() [32]byte {
	var h [32]byte
	copy(h[:], b.block.ActiveStateHash)
	return h
}

// CrystallizedStateHash returns the crystallized state hash.
func (b *Block) CrystallizedStateHash() [32]byte {
	var h [32]byte
	copy(h[:], b.block.CrystallizedStateHash)
	return h
}

// AttestationCount returns the number of attestations.
func (b *Block) AttestationCount() int {
	return len(b.block.Attestations)
}

// Attestations returns an array of attestations in the block.
func (b *Block) Attestations() []*pb.AttestationRecord {
	return b.block.Attestations
}

// Timestamp returns the Go type time.Time from the protobuf type contained in the block.
func (b *Block) Timestamp() (time.Time, error) {
	return ptypes.Timestamp(b.block.Timestamp)
}
