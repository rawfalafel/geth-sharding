// Package notary defines all relevant functionality for a Notary actor
// within a sharded Ethereum blockchain.
package notary

import (
	"context"

	"github.com/prysmaticlabs/geth-sharding/sharding/database"
	"github.com/prysmaticlabs/geth-sharding/sharding/mainchain"
	"github.com/prysmaticlabs/geth-sharding/sharding/p2p"
	"github.com/prysmaticlabs/geth-sharding/sharding/params"
	log "github.com/sirupsen/logrus"
)

// Notary holds functionality required to run a collation notary
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Notary struct {
	ctx       context.Context
	cancel    context.CancelFunc
	config    *params.Config
	smcClient *mainchain.SMCClient
	p2p       *p2p.Server
	dbService *database.ShardDB
}

// NewNotary creates a new notary instance.
func NewNotary(ctx context.Context, config *params.Config, smcClient *mainchain.SMCClient, p2p *p2p.Server, dbService *database.ShardDB) (*Notary, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &Notary{ctx, cancel, config, smcClient, p2p, dbService}, nil
}

// Start the main routine for a notary.
func (n *Notary) Start() {
	log.Info("Starting notary service")
	go n.notarizeCollations(n.ctx.Done())
}

// Stop the main loop for notarizing collations.
func (n *Notary) cleanup() error {
	log.Info("Stopping notary service")
	return nil
}

// notarizeCollations checks incoming block headers and determines if
// we are an eligible notary for collations.
func (n *Notary) notarizeCollations(done <-chan struct{}) {
	// TODO: handle this better through goroutines. Right now, these methods
	// are blocking.
	if n.smcClient.DepositFlag() {
		if err := joinNotaryPool(n.smcClient, n.smcClient); err != nil {
			log.Errorf("Could not fetch current block number: %v", err)
			return
		}
	}

	if err := subscribeBlockHeaders(n.smcClient.ChainReader(), n.smcClient, n.smcClient.Account()); err != nil {
		log.Errorf("Could not fetch current block number: %v", err)
		return
	}

	<-done
	n.cleanup()
}
