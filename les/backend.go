// Copyright 2016 The go-trustmachine Authors
// This file is part of the go-trustmachine library.
//
// The go-trustmachine library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-trustmachine library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-trustmachine library. If not, see <http://www.gnu.org/licenses/>.

// Package les implements the Light Trustmachine Subprotocol.
package les

import (
	"fmt"
	"sync"
	"time"

	"github.com/trust-tech/go-trustmachine/accounts"
	"github.com/trust-tech/go-trustmachine/common"
	"github.com/trust-tech/go-trustmachine/common/hexutil"
	"github.com/trust-tech/go-trustmachine/consensus"
	"github.com/trust-tech/go-trustmachine/core"
	"github.com/trust-tech/go-trustmachine/core/types"
	"github.com/trust-tech/go-trustmachine/entrust"
	"github.com/trust-tech/go-trustmachine/entrust/downloader"
	"github.com/trust-tech/go-trustmachine/entrust/filters"
	"github.com/trust-tech/go-trustmachine/entrust/gasprice"
	"github.com/trust-tech/go-trustmachine/entrustdb"
	"github.com/trust-tech/go-trustmachine/event"
	"github.com/trust-tech/go-trustmachine/internal/entrustapi"
	"github.com/trust-tech/go-trustmachine/light"
	"github.com/trust-tech/go-trustmachine/log"
	"github.com/trust-tech/go-trustmachine/node"
	"github.com/trust-tech/go-trustmachine/p2p"
	"github.com/trust-tech/go-trustmachine/p2p/discv5"
	"github.com/trust-tech/go-trustmachine/params"
	rpc "github.com/trust-tech/go-trustmachine/rpc"
)

type LightTrustmachine struct {
	odr         *LesOdr
	relay       *LesTxRelay
	chainConfig *params.ChainConfig
	// Channel for shutting down the service
	shutdownChan chan bool
	// Handlers
	peers           *peerSet
	txPool          *light.TxPool
	blockchain      *light.LightChain
	protocolManager *ProtocolManager
	serverPool      *serverPool
	reqDist         *requestDistributor
	retriever       *retrieveManager
	// DB interfaces
	chainDb entrustdb.Database // Block chain database

	ApiBackend *LesApiBackend

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	networkId     uint64
	netRPCService *entrustapi.PublicNetAPI

	quitSync chan struct{}
	wg       sync.WaitGroup
}

func New(ctx *node.ServiceContext, config *entrust.Config) (*LightTrustmachine, error) {
	chainDb, err := entrust.CreateDB(ctx, config, "lightchaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, isCompat := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	peers := newPeerSet()
	quitSync := make(chan struct{})

	entrust := &LightTrustmachine{
		chainConfig:    chainConfig,
		chainDb:        chainDb,
		eventMux:       ctx.EventMux,
		peers:          peers,
		reqDist:        newRequestDistributor(peers, quitSync),
		accountManager: ctx.AccountManager,
		engine:         entrust.CreateConsensusEngine(ctx, config, chainConfig, chainDb),
		shutdownChan:   make(chan bool),
		networkId:      config.NetworkId,
	}

	entrust.relay = NewLesTxRelay(peers, entrust.reqDist)
	entrust.serverPool = newServerPool(chainDb, quitSync, &entrust.wg)
	entrust.retriever = newRetrieveManager(peers, entrust.reqDist, entrust.serverPool)
	entrust.odr = NewLesOdr(chainDb, entrust.retriever)
	if entrust.blockchain, err = light.NewLightChain(entrust.odr, entrust.chainConfig, entrust.engine, entrust.eventMux); err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		entrust.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	entrust.txPool = light.NewTxPool(entrust.chainConfig, entrust.eventMux, entrust.blockchain, entrust.relay)
	if entrust.protocolManager, err = NewProtocolManager(entrust.chainConfig, true, config.NetworkId, entrust.eventMux, entrust.engine, entrust.peers, entrust.blockchain, nil, chainDb, entrust.odr, entrust.relay, quitSync, &entrust.wg); err != nil {
		return nil, err
	}
	entrust.ApiBackend = &LesApiBackend{entrust, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	entrust.ApiBackend.gpo = gasprice.NewOracle(entrust.ApiBackend, gpoParams)
	return entrust, nil
}

func lesTopic(genesisHash common.Hash) discv5.Topic {
	return discv5.Topic("LES@" + common.Bytes2Hex(genesisHash.Bytes()[0:8]))
}

type LightDummyAPI struct{}

// Trustbase is the address that mining rewards will be send to
func (s *LightDummyAPI) Trustbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Coinbase is the address that mining rewards will be send to (alias for Trustbase)
func (s *LightDummyAPI) Coinbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Hashrate returns the POW hashrate
func (s *LightDummyAPI) Hashrate() hexutil.Uint {
	return 0
}

// Mining returns an indication if this node is currently mining.
func (s *LightDummyAPI) Mining() bool {
	return false
}

// APIs returns the collection of RPC services the trustmachine package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *LightTrustmachine) APIs() []rpc.API {
	return append(entrustapi.GetAPIs(s.ApiBackend), []rpc.API{
		{
			Namespace: "entrust",
			Version:   "1.0",
			Service:   &LightDummyAPI{},
			Public:    true,
		}, {
			Namespace: "entrust",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "entrust",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, true),
			Public:    true,
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *LightTrustmachine) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *LightTrustmachine) BlockChain() *light.LightChain      { return s.blockchain }
func (s *LightTrustmachine) TxPool() *light.TxPool              { return s.txPool }
func (s *LightTrustmachine) Engine() consensus.Engine           { return s.engine }
func (s *LightTrustmachine) LesVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *LightTrustmachine) Downloader() *downloader.Downloader { return s.protocolManager.downloader }
func (s *LightTrustmachine) EventMux() *event.TypeMux           { return s.eventMux }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *LightTrustmachine) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

// Start implements node.Service, starting all internal goroutines needed by the
// Trustmachine protocol implementation.
func (s *LightTrustmachine) Start(srvr *p2p.Server) error {
	log.Warn("Light client mode is an experimental feature")
	s.netRPCService = entrustapi.NewPublicNetAPI(srvr, s.networkId)
	s.serverPool.start(srvr, lesTopic(s.blockchain.Genesis().Hash()))
	s.protocolManager.Start()
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Trustmachine protocol.
func (s *LightTrustmachine) Stop() error {
	s.odr.Stop()
	s.blockchain.Stop()
	s.protocolManager.Stop()
	s.txPool.Stop()

	s.eventMux.Stop()

	time.Sleep(time.Millisecond * 200)
	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
