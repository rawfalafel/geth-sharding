package db

import (
	"bytes"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestNilDB(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	b := types.NewBlock(nil)
	h, _ := b.Hash()

	hasBlock := db.HasBlock(h)
	if hasBlock {
		t.Fatal("HashBlock should return false")
	}

	bPrime, err := db.GetBlock(h)
	if err != nil {
		t.Fatalf("failed to get block: %v", err)
	}
	if bPrime != nil {
		t.Fatalf("get should return nil for a non existent key")
	}
}

func TestGetChainHead(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	if err := db.InitializeState(nil); err != nil {
		t.Fatalf("InitializeState failed: %v", err)
	}

	block, err := db.GetChainHead()
	if err != nil {
		t.Fatalf("GetChainHead failed: %v", err)
	}
	if block == nil {
		t.Fatal("Expected GetChainHead to return a block")
	}
}

func TestRecordBlock(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	aState := types.NewActiveState(&pb.ActiveState{}, nil)
	aStateHash, err := aState.Hash()
	if err != nil {
		t.Fatalf("Failed to hash active state: %v", err)
	}
	cState := types.NewCrystallizedState(&pb.CrystallizedState{})
	cStateHash, err := cState.Hash()
	if err != nil {
		t.Fatalf("Failed to hash crystallized state: %v", err)
	}
	b := types.NewBlock(&pb.BeaconBlock{
		ActiveStateRoot:       aStateHash[:],
		CrystallizedStateRoot: cStateHash[:],
	})
	bHash, err := b.Hash()
	if err != nil {
		t.Fatalf("Failed to hash block: %v", err)
	}

	err = db.RecordBlock(b, aState, nil)
	if err != nil {
		t.Fatalf("Failed to record block: %v", err)
	}

	bPrime, err := db.GetBlock(bHash)
	if err != nil || bPrime == nil {
		t.Fatalf("Failed to get block: %v", err)
	}
	bEnc, _ := b.Marshal()
	bPrimeEnc, _ := bPrime.Marshal()
	if !bytes.Equal(bEnc, bPrimeEnc) {
		t.Fatalf("Expected %#x and %#x to be qual", bEnc, bPrimeEnc)
	}

	aStatePrime, err := db.GetActiveState(aStateHash)
	if err != nil || aStatePrime == nil {
		t.Fatalf("Failed to retrieve active state: %v", err)
	}
	aStateEnc, _ := aState.Marshal()
	aStatePrimeEnc, _ := aStatePrime.Marshal()
	if !bytes.Equal(aStateEnc, aStatePrimeEnc) {
		t.Fatalf("Expected %v and %v to be equal", aState, aStatePrime)
	}
}

func TestGetBlockBySlotEmptyChain(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	b, err := db.GetBlockBySlot(0)
	if err != nil {
		t.Errorf("failure when fetching block by slot: %v", err)
	}
	if b != nil {
		t.Error("GetBlockBySlot should return nil for an empty chain")
	}
}

func TestUpdateChainHead(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	err := db.InitializeState(nil)
	if err != nil {
		t.Fatalf("failed to initialize state: %v", err)
	}

	genesis, err := db.GetBlockBySlot(0)
	if err != nil || genesis == nil {
		t.Fatalf("Failed to retrieve genesis block: %v", err)
	}
	genesisHash, _ := genesis.Hash()

	aState := types.NewActiveState(&pb.ActiveState{}, nil)
	aStateHash, _ := aState.Hash()
	b1 := types.NewBlock(&pb.BeaconBlock{
		Slot:            1,
		ActiveStateRoot: aStateHash[:],
		AncestorHashes:  [][]byte{genesisHash[:]},
	})
	if err := db.RecordBlock(b1, aState, nil); err != nil {
		t.Fatalf("Failed to record block: %v", err)
	}

	if err = db.UpdateChainHead(b1); err != nil {
		t.Fatalf("Failed to update head: %v", err)
	}
	b1Prime, err := db.GetChainHead()
	if err != nil {
		t.Fatalf("Failed to get chain head: %v", err)
	}
	b1Enc, _ := b1.Marshal()
	b1PrimeEnc, _ := b1Prime.Marshal()
	if !bytes.Equal(b1Enc, b1PrimeEnc) {
		t.Fatalf("Expected %#x and %#x to be equal", b1Enc, b1PrimeEnc)
	}

	b1Omega, err := db.GetBlockBySlot(1)
	b1OmegaEnc, _ := b1Omega.Marshal()
	if !bytes.Equal(b1Enc, b1OmegaEnc) {
		t.Fatalf("Expected %#x and %#x to be equal", b1Enc, b1OmegaEnc)
	}
}

func TestGetGenesisTime(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	time, err := db.GetGenesisTime()
	if err == nil {
		t.Fatal("expected GetGenesisTime to fail")
	}

	err = db.InitializeState(nil)
	if err != nil {
		t.Fatalf("failed to initialize state: %v", err)
	}

	time, err = db.GetGenesisTime()
	if err != nil {
		t.Fatalf("GetGenesisTime failed on second attempt: %v", err)
	}
	time2, err := db.GetGenesisTime()
	if err != nil {
		t.Fatalf("GetGenesisTime failed on second attempt: %v", err)
	}

	if time != time2 {
		t.Fatalf("Expected %v and %v to be equal", time, time2)
	}
}
