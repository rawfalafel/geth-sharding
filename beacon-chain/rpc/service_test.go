package rpc

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ethereum/go-ethereum/event"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

type mockChainService struct{}

func (m *mockChainService) IncomingBlockFeed() *event.Feed {
	return new(event.Feed)
}

type mockAnnouncer struct {
	blockFeed *event.Feed
	stateFeed *event.Feed
}

func newMockAnnouncer() *mockAnnouncer {
	return &mockAnnouncer{
		blockFeed: new(event.Feed),
		stateFeed: new(event.Feed),
	}
}

func (m *mockAnnouncer) CanonicalBlockFeed() *event.Feed {
	return m.blockFeed
}

func (m *mockAnnouncer) CanonicalCrystallizedStateFeed() *event.Feed {
	return m.stateFeed
}

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	announcer := newMockAnnouncer()
	rpcService := NewRPCService(context.Background(), &Config{
		Port:      "7348",
		CertFlag:  "alice.crt",
		KeyFlag:   "alice.key",
		Announcer: announcer,
	})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("RPC server listening on port :%s", rpcService.port))

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestBadEndpoint(t *testing.T) {
	hook := logTest.NewGlobal()
	announcer := newMockAnnouncer()
	rpcService := NewRPCService(context.Background(), &Config{Port: "ralph merkle!!!", Announcer: announcer})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Could not listen to port :%s", rpcService.port))

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestInsecureEndpoint(t *testing.T) {
	hook := logTest.NewGlobal()
	announcer := newMockAnnouncer()
	rpcService := NewRPCService(context.Background(), &Config{Port: "7777", Announcer: announcer})

	rpcService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("RPC server listening on port :%s", rpcService.port))
	testutil.AssertLogsContain(t, hook, "You are using an insecure gRPC connection")

	rpcService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
}

func TestRPCMethods(t *testing.T) {
	announcer := newMockAnnouncer()
	rpcService := NewRPCService(context.Background(), &Config{Port: "7362", Announcer: announcer})
	if _, err := rpcService.SignBlock(context.Background(), nil); err == nil {
		t.Error("Wanted error: unimplemented, received nil")
	}
}

func TestFetchShuffledValidatorIndices(t *testing.T) {
	announcer := newMockAnnouncer()
	rpcService := NewRPCService(context.Background(), &Config{Port: "6372", Announcer: announcer})
	res, err := rpcService.FetchShuffledValidatorIndices(context.Background(), &pb.ShuffleRequest{})
	if err != nil {
		t.Fatalf("Could not call RPC method: %v", err)
	}
	if len(res.ShuffledValidatorIndices) != 100 {
		t.Errorf("Expected 100 validators in the shuffled indices, received %d", len(res.ShuffledValidatorIndices))
	}
}

func TestProposeBlock(t *testing.T) {
	announcer := newMockAnnouncer()
	mockChain := &mockChainService{}
	rpcService := NewRPCService(context.Background(), &Config{
		Port:         "6372",
		Announcer:    announcer,
		ChainService: mockChain,
	})
	req := &pb.ProposeRequest{
		SlotNumber: 5,
		ParentHash: []byte("parent-hash"),
		Timestamp:  ptypes.TimestampNow(),
	}
	if _, err := rpcService.ProposeBlock(context.Background(), req); err != nil {
		t.Errorf("Could not propose block correctly: %v", err)
	}
}

func TestLatestBeaconBlockContextClosed(t *testing.T) {
	hook := logTest.NewGlobal()
	announcer := newMockAnnouncer()
	rpcService := NewRPCService(context.Background(), &Config{Port: "6663", SubscriptionBuf: 0, Announcer: announcer})
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := internal.NewMockBeaconService_LatestBeaconBlockServer(ctrl)
	go func(tt *testing.T) {
		if err := rpcService.LatestBeaconBlock(&empty.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	rpcService.cancel()
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "RPC context closed, exiting goroutine")
}

func TestLatestBeaconBlockFailure(t *testing.T) {
	announcer := newMockAnnouncer()
	rpcService := NewRPCService(context.Background(), &Config{Port: "7771", SubscriptionBuf: 0, Announcer: announcer})
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	exitRoutine := make(chan bool)

	blk, _ := types.NewBlock(nil)

	mockStream := internal.NewMockBeaconService_LatestBeaconBlockServer(ctrl)
	mockStream.EXPECT().Send(blk.Proto()).Return(errors.New("something wrong"))
	// Tests a faulty stream.
	go func(tt *testing.T) {
		if err := rpcService.LatestBeaconBlock(&empty.Empty{}, mockStream); err.Error() != "something wrong" {
			tt.Errorf("Faulty stream should throw correct error, wanted 'something wrong', got %v", err)
		}
		<-exitRoutine
	}(t)

	rpcService.canonicalBlockChan <- blk
	rpcService.cancel()
	exitRoutine <- true
}

func TestLatestBeaconBlockSuccess(t *testing.T) {
	hook := logTest.NewGlobal()
	announcer := newMockAnnouncer()
	rpcService := NewRPCService(context.Background(), &Config{Port: "7771", SubscriptionBuf: 0, Announcer: announcer})
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	exitRoutine := make(chan bool)

	blk, _ := types.NewBlock(nil)

	mockStream := internal.NewMockBeaconService_LatestBeaconBlockServer(ctrl)
	mockStream.EXPECT().Send(blk.Proto()).Return(nil)
	// Tests a good stream.
	go func(tt *testing.T) {
		if err := rpcService.LatestBeaconBlock(&empty.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)

	rpcService.canonicalBlockChan <- blk
	rpcService.cancel()
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "Sending latest canonical block to RPC clients")
}

func TestLatestCrystallizedStateContextClosed(t *testing.T) {
	hook := logTest.NewGlobal()
	announcer := newMockAnnouncer()
	rpcService := NewRPCService(context.Background(), &Config{Port: "8777", SubscriptionBuf: 0, Announcer: announcer})
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := internal.NewMockBeaconService_LatestCrystallizedStateServer(ctrl)
	go func(tt *testing.T) {
		if err := rpcService.LatestCrystallizedState(&empty.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	rpcService.cancel()
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "RPC context closed, exiting goroutine")
}

func TestLatestCrystallizedStateFailure(t *testing.T) {
	announcer := newMockAnnouncer()
	rpcService := NewRPCService(context.Background(), &Config{Port: "8773", SubscriptionBuf: 0, Announcer: announcer})
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	exitRoutine := make(chan bool)

	mockStream := internal.NewMockBeaconService_LatestCrystallizedStateServer(ctrl)
	mockStream.EXPECT().Send(&pbp2p.CrystallizedState{}).Return(errors.New("something wrong"))
	// Tests a faulty stream.
	go func(tt *testing.T) {
		if err := rpcService.LatestCrystallizedState(&empty.Empty{}, mockStream); err.Error() != "something wrong" {
			tt.Errorf("Faulty stream should throw correct error, wanted 'something wrong', got %v", err)
		}
		<-exitRoutine
	}(t)
	rpcService.canonicalStateChan <- types.NewCrystallizedState(&pbp2p.CrystallizedState{})
	rpcService.cancel()
	exitRoutine <- true
}

func TestLatestCrystallizedStateSuccess(t *testing.T) {
	hook := logTest.NewGlobal()
	announcer := newMockAnnouncer()
	rpcService := NewRPCService(context.Background(), &Config{Port: "8773", SubscriptionBuf: 0, Announcer: announcer})
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	exitRoutine := make(chan bool)

	mockStream := internal.NewMockBeaconService_LatestCrystallizedStateServer(ctrl)
	mockStream.EXPECT().Send(&pbp2p.CrystallizedState{}).Return(nil)

	// Tests a good stream.
	go func(tt *testing.T) {
		if err := rpcService.LatestCrystallizedState(&empty.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	rpcService.canonicalStateChan <- types.NewCrystallizedState(&pbp2p.CrystallizedState{})
	testutil.AssertLogsContain(t, hook, "Sending crystallized state to RPC clients")
	rpcService.cancel()
	exitRoutine <- true
}
