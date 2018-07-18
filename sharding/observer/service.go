// Package observer launches a service attached to the sharding node
// that simply observes activity across the sharded Ethereum network.
package observer

import (
	"context"

	"github.com/prysmaticlabs/geth-sharding/sharding/database"
	"github.com/prysmaticlabs/geth-sharding/sharding/mainchain"
	"github.com/prysmaticlabs/geth-sharding/sharding/p2p"
	"github.com/prysmaticlabs/geth-sharding/sharding/syncer"
	"github.com/prysmaticlabs/geth-sharding/sharding/types"
	log "github.com/sirupsen/logrus"
)

// Observer holds functionality required to run an observer service
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Observer struct {
	dbService *database.ShardDB
	shardID   int
	shard     *types.Shard
	ctx       context.Context
	cancel    context.CancelFunc
	sync      *syncer.Syncer
	client    *mainchain.SMCClient
}

// NewObserver creates a struct instance of a observer service,
// it will have access to a p2p server and a shardChainDB.
func NewObserver(p2p *p2p.Server, dbService *database.ShardDB, shardID int, sync *syncer.Syncer, client *mainchain.SMCClient) (*Observer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &Observer{dbService, shardID, nil, ctx, cancel, sync, client}, nil
}

// Start the main loop for observer service.
func (o *Observer) Start() {
	log.Info("Starting observer service")
}

// Stop the main loop for observer service.
func (o *Observer) Stop() error {
	log.Info("Stopping observer service")
	return nil
}
