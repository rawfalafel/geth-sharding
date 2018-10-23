package db

import (
	"errors"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func createBlock(enc []byte) (*types.Block, error) {
	protoBlock := &pb.BeaconBlock{}
	err := proto.Unmarshal(enc, protoBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal encoding: %v", err)
	}

	block := types.NewBlock(protoBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate a block from the encoding: %v", err)
	}

	return block, nil
}

// GetBlock accepts a block hash and returns the corresponding block.
// Returns nil if the block does not exist.
func (db *BeaconDB) GetBlock(hash [32]byte) (*types.Block, error) {
	var block *types.Block
	err := db.view(func(b *bolt.Bucket) error {
		enc := b.Get(blockKey(hash[:]))
		if enc == nil {
			return nil
		}

		var err error
		block, err = createBlock(enc)
		return err
	})

	return block, err
}

// HasBlock accepts a block hash and returns true if the block does not exist.
func (db *BeaconDB) HasBlock(hash [32]byte) bool {
	hasBlock := false
	_ = db.view(func(b *bolt.Bucket) error {
		hasBlock = b.Get(hash[:]) != nil

		return nil
	})

	return hasBlock
}

// GetChainHead returns the head of the main chain.
func (db *BeaconDB) GetChainHead() (*types.Block, error) {
	var block *types.Block

	err := db.view(func(b *bolt.Bucket) error {
		height := b.Get(chainHeightKey)
		if height == nil {
			return errors.New("unable to determinechain height")
		}

		blockhash := b.Get(blockByHeightKey(height))
		if blockhash == nil {
			return fmt.Errorf("hash at the current height not found: %d", height)
		}

		enc := b.Get(blockKey(blockhash))
		if enc == nil {
			return fmt.Errorf("block not found: %x", blockhash)
		}

		var err error
		block, err = createBlock(enc)

		return err
	})

	return block, err
}

// UpdateChainHead updates the given block as the head of the main chain.
// Assumes that the block has already been saved to the database.
func (db *BeaconDB) UpdateChainHead(block *types.Block) error {
	blockhash, err := block.Hash()
	if err != nil {
		return fmt.Errorf("unable to get the block hash: %v", err)
	}

	slotBinary := encodeSlotNumber(block.SlotNumber())
	return db.update(func(b *bolt.Bucket) error {
		if b.Get(blockKey(blockhash[:])) == nil {
			return fmt.Errorf("expected block %#x to have already been saved before updating head: %v", blockhash, err)
		}

		if err := b.Put(blockByHeightKey(slotBinary), blockhash[:]); err != nil {
			return fmt.Errorf("failed to include the block in the main chain bucket: %v", err)
		}

		if err := b.Put(chainHeightKey, slotBinary); err != nil {
			return fmt.Errorf("failed to record the block as the head of the main chain: %v", err)
		}

		return nil
	})
}

// RecordBlock atomically updates the head of the chain as well as the corresponding state changes
// Including a new crystallized state is optional.
func (db *BeaconDB) RecordBlock(block *types.Block, aState *types.ActiveState, cState *types.CrystallizedState) error {
	blockhash, err := block.Hash()
	if err != nil {
		return fmt.Errorf("unable to get the block hash: %v", err)
	}
	blockEnc, err := block.Marshal()
	if err != nil {
		return fmt.Errorf("unable to encode the block: %v", err)
	}

	aStateHash, err := aState.Hash()
	if err != nil {
		return fmt.Errorf("unable to hash the active state: %v", err)
	}
	aStateEnc, err := aState.Marshal()
	if err != nil {
		return fmt.Errorf("unable to encode the active state: %v", err)
	}

	var cStateHash [32]byte
	var cStateEnc []byte
	if cState != nil {
		cStateHash, err = cState.Hash()
		if err != nil {
			return fmt.Errorf("unable to hash the crystallized state: %v", err)
		}
		cStateEnc, err = cState.Marshal()
		if err != nil {
			return fmt.Errorf("unable to encode the crystallized state: %v", err)
		}
	}

	return db.update(func(b *bolt.Bucket) error {
		if err := b.Put(blockKey(blockhash[:]), blockEnc); err != nil {
			return fmt.Errorf("failed to save block: %v", err)
		}

		if err := b.Put(aStateKey(aStateHash[:]), aStateEnc); err != nil {
			return fmt.Errorf("failed to record the new active state: %v", err)
		}

		if cState != nil {
			if err := b.Put(cStateKey(cStateHash[:]), cStateEnc); err != nil {
				return fmt.Errorf("failed to record the new crystallized state: %v", err)
			}
		}

		return nil
	})
}

// GetBlockBySlot accepts a slot number and returns the corresponding block in the main chain.
// Returns nil if a block was not recorded for the given slot.
func (db *BeaconDB) GetBlockBySlot(slot uint64) (*types.Block, error) {
	var block *types.Block
	slotEnc := encodeSlotNumber(slot)

	err := db.view(func(b *bolt.Bucket) error {
		blockhash := b.Get(blockByHeightKey(slotEnc))
		if blockhash == nil {
			return nil
		}

		enc := b.Get(blockKey(blockhash))
		if enc == nil {
			return fmt.Errorf("block not found: %x", blockhash)
		}

		var err error
		block, err = createBlock(enc)
		return err
	})

	return block, err
}

// GetGenesisTime returns the timestamp for the genesis block
func (db *BeaconDB) GetGenesisTime() (time.Time, error) {
	genesis, err := db.GetBlockBySlot(0)
	if err != nil {
		return time.Time{}, fmt.Errorf("Could not get genesis block: %v", err)
	}
	if genesis == nil {
		return time.Time{}, fmt.Errorf("Genesis block not found: %v", err)
	}

	genesisTime, err := genesis.Timestamp()
	if err != nil {
		return time.Time{}, fmt.Errorf("Could not get genesis timestamp: %v", err)
	}

	return genesisTime, nil
}
