package db

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// InitializeState ...
func (db *BeaconDB) InitializeState(genesisValidators []*pb.ValidatorRecord) error {
	aState := types.NewGenesisActiveState()
	cState, err := types.NewGenesisCrystallizedState(genesisValidators)
	if err != nil {
		return err
	}

	// #nosec G104
	aStateHash, _ := aState.Hash()
	// #nosec G104
	cStateHash, _ := cState.Hash()

	genesisBlock := types.NewGenesisBlock(aStateHash, cStateHash)
	// #nosec G104
	blockhash, _ := genesisBlock.Hash()

	// #nosec G104
	blockEnc, _ := genesisBlock.Marshal()
	// #nosec G104
	aStateEnc, _ := aState.Marshal()
	// #nosec G104
	cStateEnc, _ := cState.Marshal()

	zeroBinary := encodeSlotNumber(0)

	return db.update(func(tx *bolt.Tx) error {
		b := tx.Bucket(mainBucket)

		if err := b.Put(chainHeightKey, zeroBinary); err != nil {
			return fmt.Errorf("failed to record block height: %v", err)
		}

		if err := b.Put(blockByHeightKey(zeroBinary), blockhash[:]); err != nil {
			return fmt.Errorf("failed to record block hash: %v", err)
		}

		if err := b.Put(blockKey(blockhash[:]), blockEnc); err != nil {
			return err
		}

		if err := b.Put(aStateKey(aStateHash[:]), aStateEnc); err != nil {
			return err
		}

		if err := b.Put(cStateKey(cStateHash[:]), cStateEnc); err != nil {
			return err
		}

		return nil
	})
}

// GetActiveState fetches ActiveState for the given hash.
// Returns nil if the state does not exist.
func (db *BeaconDB) GetActiveState(hash []byte) (*types.ActiveState, error) {
	var state *types.ActiveState
	err := db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(mainBucket)

		stateEnc := b.Get(aStateKey(hash))
		if stateEnc == nil {
			return nil
		}

		var err error
		state, err = createActiveState(stateEnc)
		return err
	})

	return state, err
}

// GetCrystallizedState fetches CrystallizedState for the given hash.
// Returns nil if the state does not exist.
func (db *BeaconDB) GetCrystallizedState(hash []byte) (*types.CrystallizedState, error) {
	var state *types.CrystallizedState
	err := db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(mainBucket)

		stateEnc := b.Get(cStateKey(hash))
		if stateEnc == nil {
			return nil
		}

		var err error
		state, err = createCrystallizedState(stateEnc)
		return err
	})

	return state, err
}

func createActiveState(enc []byte) (*types.ActiveState, error) {
	protoState := &pb.ActiveState{}
	err := proto.Unmarshal(enc, protoState)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal encoding: %v", err)
	}

	state := types.NewActiveState(protoState, map[[32]byte]*utils.VoteCache{})

	return state, nil
}

func createCrystallizedState(enc []byte) (*types.CrystallizedState, error) {
	protoState := &pb.CrystallizedState{}
	err := proto.Unmarshal(enc, protoState)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal encoding: %v", err)
	}

	state := types.NewCrystallizedState(protoState)

	return state, nil
}
