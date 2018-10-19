// Package simulator defines the simulation utility to test the beacon-chain.
package simulator

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/casper"
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/slotticker"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "simulator")

type p2pAPI interface {
	Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription
	Send(msg proto.Message, peer p2p.Peer)
	Broadcast(msg proto.Message)
}

type powChainService interface {
	LatestBlockHash() common.Hash
}

// Simulator struct.
type Simulator struct {
	ctx              context.Context
	cancel           context.CancelFunc
	p2p              p2pAPI
	web3Service      powChainService
	beaconDB         beaconDB
	enablePOWChain   bool
	blockRequestChan chan p2p.Message
}

// Config options for the simulator service.
type Config struct {
	BlockRequestBuf int
	P2P             p2pAPI
	Web3Service     powChainService
	BeaconDB        beaconDB
	EnablePOWChain  bool
}

type beaconDB interface {
	GetChainHead() (*types.Block, error)
	GetGenesisTime() (time.Time, error)
	GetActiveState([]byte) (*types.ActiveState, error)
	GetCrystallizedState([]byte) (*types.CrystallizedState, error)
}

// DefaultConfig options for the simulator.
func DefaultConfig() *Config {
	return &Config{
		BlockRequestBuf: 100,
	}
}

// NewSimulator creates a simulator instance for a syncer to consume fake, generated blocks.
func NewSimulator(ctx context.Context, cfg *Config) *Simulator {
	ctx, cancel := context.WithCancel(ctx)
	return &Simulator{
		ctx:              ctx,
		cancel:           cancel,
		p2p:              cfg.P2P,
		web3Service:      cfg.Web3Service,
		beaconDB:         cfg.BeaconDB,
		enablePOWChain:   cfg.EnablePOWChain,
		blockRequestChan: make(chan p2p.Message, cfg.BlockRequestBuf),
	}
}

// Start the sim.
func (sim *Simulator) Start() {
	log.Info("Starting service")
	genesisTime, err := sim.beaconDB.GetGenesisTime()
	if err != nil {
		log.Fatal(err)
		return
	}

	slotTicker := slotticker.GetSlotTicker(genesisTime, params.GetConfig().SlotDuration)
	go func() {
		sim.run(slotTicker.C(), sim.blockRequestChan)
		close(sim.blockRequestChan)
		slotTicker.Done()
	}()
}

// Stop the sim.
func (sim *Simulator) Stop() error {
	defer sim.cancel()
	log.Info("Stopping service")
	return nil
}

func (sim *Simulator) run(slotInterval <-chan uint64, requestChan <-chan p2p.Message) {
	blockReqSub := sim.p2p.Subscribe(&pb.BeaconBlockRequest{}, sim.blockRequestChan)
	defer blockReqSub.Unsubscribe()

	lastBlock, err := sim.beaconDB.GetChainHead()
	if err != nil {
		log.Errorf("Could not fetch latest block: %v", err)
		return
	}

	lastHash, err := lastBlock.Hash()
	if err != nil {
		log.Errorf("Could not get hash of the latest block: %v", err)
	}
	broadcastedBlocks := map[[32]byte]*types.Block{}

	for {
		select {
		case <-sim.ctx.Done():
			log.Debug("Simulator context closed, exiting goroutine")
			return
		case slot := <-slotInterval:
			aState, err := sim.beaconDB.GetActiveState(lastHash[:])
			if err != nil {
				log.Errorf("Failed to get active state: %v", err)
				continue
			}
			cState, err := sim.beaconDB.GetCrystallizedState(lastHash[:])
			if err != nil {
				log.Errorf("Failed to get crystallized state: %v", err)
				continue
			}

			cState.NewStateRecalculations(aState, )

			var powChainRef []byte
			if sim.enablePOWChain {
				powChainRef = sim.web3Service.LatestBlockHash().Bytes()
			} else {
				powChainRef = []byte{byte(slot)}
			}

			attestationSlot := slot-1
			shardAndCommittees, err := casper.GetShardAndCommitteesForSlot(cState.ShardAndCommitteesForSlots(), cState.LastStateRecalculationSlot(), attestationSlot)
			if err != nil {
				log.Errorf("Failed to get shards and committees for slot %d: %v", slot, err)
				continue
			}
			shardID := shardAndCommittees.ArrayShardAndCommittee[0].Shard

			parentHash := make([]byte, 32)
			copy(parentHash, lastHash[:])
			block := types.NewBlock(&pb.BeaconBlock{
				Slot:                  slot,
				Timestamp:             ptypes.TimestampNow(),
				PowChainRef:           powChainRef,
				ActiveStateRoot:       aStateHash[:],
				CrystallizedStateRoot: cStateHash[:],
				AncestorHashes:        [][]byte{parentHash},
				RandaoReveal:          params.GetConfig().SimulatedBlockRandao[:],
				Attestations: []*pb.AggregatedAttestation{
					{Slot: attestationSlot, AttesterBitfield: []byte{byte(128)}, JustifiedBlockHash: parentHash, Shard: shardID},
				},
			})

			hash, err := block.Hash()
			if err != nil {
				log.Errorf("Could not hash simulated block: %v", err)
				continue
			}
			sim.p2p.Broadcast(&pb.BeaconBlockHashAnnounce{
				Hash: hash[:],
			})

			log.WithFields(logrus.Fields{
				"hash": fmt.Sprintf("%#x", hash),
				"slot": slot,
			}).Info("Broadcast block hash")

			broadcastedBlocks[hash] = block
			lastHash = hash
		case msg := <-requestChan:
			data := msg.Data.(*pb.BeaconBlockRequest)
			var hash [32]byte
			copy(hash[:], data.Hash)

			block := broadcastedBlocks[hash]
			if block == nil {
				log.Errorf("Requested block not found: %#x", hash)
				continue
			}

			log.WithFields(logrus.Fields{
				"hash": fmt.Sprintf("%#x", hash),
			}).Debug("Responding to full block request")

			// Sends the full block body to the requester.
			res := &pb.BeaconBlockResponse{Block: block.Proto(), Attestation: &pb.AggregatedAttestation{
				Slot:             block.SlotNumber(),
				AttesterBitfield: []byte{byte(255)},
			}}
			sim.p2p.Send(res, msg.Peer)

			delete(broadcastedBlocks, hash)
		}
	}
}
