package simulator

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/testutils"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

type mockP2P struct{}

func (mp *mockP2P) Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription {
	return new(event.Feed).Subscribe(channel)
}

func (mp *mockP2P) Broadcast(msg proto.Message) {}

func (mp *mockP2P) Send(msg proto.Message, peer p2p.Peer) {}

type mockPOWChainService struct{}

func (mpow *mockPOWChainService) LatestBlockHash() common.Hash {
	return common.BytesToHash([]byte{})
}

type mockChainService struct{}

func (mc *mockChainService) CurrentActiveState() *types.ActiveState {
	return types.NewActiveState(&pb.ActiveState{}, make(map[[32]byte]*types.VoteCache))
}

func (mc *mockChainService) CurrentCrystallizedState() *types.CrystallizedState {
	return types.NewCrystallizedState(&pb.CrystallizedState{})
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	bolt := testutils.SetupDB(t)
	cfg := &Config{
		Delay:           time.Second,
		BlockRequestBuf: 0,
		P2P:             &mockP2P{},
		Web3Service:     &mockPOWChainService{},
		ChainService:    &mockChainService{},
		BeaconDB:        db.NewDB(bolt),
		Validator:       false,
	}
	sim := NewSimulator(context.Background(), cfg)

	sim.Start()
	testutil.AssertLogsContain(t, hook, "Starting service")
	sim.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")

	// The context should have been canceled.
	if sim.ctx.Err() == nil {
		t.Error("context was not canceled")
	}
}

func TestBroadcastBlockHash(t *testing.T) {
	hook := logTest.NewGlobal()
	bolt := testutils.SetupDB(t)
	cfg := &Config{
		Delay:           time.Second,
		BlockRequestBuf: 0,
		P2P:             &mockP2P{},
		Web3Service:     &mockPOWChainService{},
		ChainService:    &mockChainService{},
		BeaconDB:        db.NewDB(bolt),
		Validator:       false,
	}
	sim := NewSimulator(context.Background(), cfg)

	delayChan := make(chan time.Time)
	doneChan := make(chan struct{})
	exitRoutine := make(chan bool)

	go func() {
		sim.run(delayChan, doneChan)
		<-exitRoutine
	}()

	delayChan <- time.Time{}
	doneChan <- struct{}{}

	testutil.AssertLogsContain(t, hook, "Announcing block hash")

	exitRoutine <- true

	if len(sim.broadcastedBlockHashes) != 1 {
		t.Error("Did not store the broadcasted block hash")
	}
	hook.Reset()
}

func TestBlockRequest(t *testing.T) {
	hook := logTest.NewGlobal()
	bolt := testutils.SetupDB(t)
	cfg := &Config{
		Delay:           time.Second,
		BlockRequestBuf: 0,
		P2P:             &mockP2P{},
		Web3Service:     &mockPOWChainService{},
		ChainService:    &mockChainService{},
		BeaconDB:        db.NewDB(bolt),
		Validator:       true,
	}
	sim := NewSimulator(context.Background(), cfg)

	delayChan := make(chan time.Time)
	doneChan := make(chan struct{})
	exitRoutine := make(chan bool)

	go func() {
		sim.run(delayChan, doneChan)
		<-exitRoutine
	}()

	block := types.NewBlock(&pb.BeaconBlock{ParentHash: make([]byte, 32)})
	h, err := block.Hash()
	if err != nil {
		t.Fatal(err)
	}

	data := &pb.BeaconBlockRequest{
		Hash: h[:],
	}

	msg := p2p.Message{
		Peer: p2p.Peer{},
		Data: data,
	}

	sim.broadcastedBlocks[h] = block

	sim.blockRequestChan <- msg
	doneChan <- struct{}{}
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Responding to full block request for hash: 0x%x", h))
}

func TestDefaultConfig(t *testing.T) {
	if DefaultConfig().BlockRequestBuf != 100 {
		t.Errorf("incorrect default config for block request buffer")
	}
	if DefaultConfig().Delay != time.Second*5 {
		t.Errorf("incorrect default config for delay")
	}
}
