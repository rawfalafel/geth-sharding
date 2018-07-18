package powchain

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	log "github.com/sirupsen/logrus"
)

// Reader defines a struct that can fetch latest header events from a web3 endpoint.
type Reader interface {
	SubscribeNewHead(ctx context.Context, ch chan<- *gethTypes.Header) (ethereum.Subscription, error)
}

// Web3Service fetches important information about the canonical
// Ethereum PoW chain via a web3 endpoint using an ethclient. The Random
// Beacon Chain requires synchronization with the PoW chain's current
// blockhash, block number, and access to logs within the
// Validator Registration Contract on the PoW chain to kick off the beacon
// chain's validator registration process.
type Web3Service struct {
	ctx         context.Context
	cancel      context.CancelFunc
	headerChan  chan *gethTypes.Header
	endpoint    string
	blockNumber *big.Int    // the latest PoW chain blocknumber.
	blockHash   common.Hash // the latest PoW chain blockhash.
}

// NewWeb3Service sets up a new instance with an ethclient when
// given a web3 endpoint as a string.
func NewWeb3Service(ctx context.Context, endpoint string) (*Web3Service, error) {
	if !strings.HasPrefix(endpoint, "ws") && !strings.HasPrefix(endpoint, "ipc") {
		return nil, fmt.Errorf("web3service requires either an IPC or WebSocket endpoint, provided %s", endpoint)
	}
	web3ctx, cancel := context.WithCancel(ctx)
	return &Web3Service{
		ctx:         web3ctx,
		cancel:      cancel,
		headerChan:  make(chan *gethTypes.Header),
		endpoint:    endpoint,
		blockNumber: nil,
		blockHash:   common.BytesToHash([]byte{}),
	}, nil
}

// Start a web3 service's main event loop.
func (w *Web3Service) Start() {
	log.Infof("Starting web3 PoW chain service at %s", w.endpoint)
	rpcClient, err := rpc.Dial(w.endpoint)
	if err != nil {
		log.Errorf("Cannot connect to PoW chain RPC client: %v", err)
		return
	}
	client := ethclient.NewClient(rpcClient)
	go w.latestPOWChainInfo(client, w.ctx.Done())
}

// Stop the web3 service's main event loop and associated goroutines.
func (w *Web3Service) stop() error {
	defer w.cancel()
	defer close(w.headerChan)
	log.Info("Stopping web3 PoW chain service")
	return nil
}

func (w *Web3Service) latestPOWChainInfo(reader Reader, done <-chan struct{}) {
	if _, err := reader.SubscribeNewHead(w.ctx, w.headerChan); err != nil {
		log.Errorf("Unable to subscribe to incoming PoW chain headers: %v", err)
		return
	}
	for {
		select {
		case <-done:
			w.stop()
			return
		case header := <-w.headerChan:
			w.blockNumber = header.Number
			w.blockHash = header.Hash()
			log.Debugf("Latest PoW chain blocknumber: %v", w.blockNumber)
			log.Debugf("Latest PoW chain blockhash: %v", w.blockHash.Hex())
		}
	}
}

// LatestBlockNumber is a getter for blockNumber to make it read-only.
func (w *Web3Service) LatestBlockNumber() *big.Int {
	return w.blockNumber
}

// LatestBlockHash is a getter for blockHash to make it read-only.
func (w *Web3Service) LatestBlockHash() common.Hash {
	return w.blockHash
}
