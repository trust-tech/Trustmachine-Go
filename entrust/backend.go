// Copyright 2014 The go-trustmachine Authors
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

// Package entrust implements the Trustmachine protocol.
package entrust

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/trust-tech/go-trustmachine/accounts"
	"github.com/trust-tech/go-trustmachine/common"
	"github.com/trust-tech/go-trustmachine/common/hexutil"
	"github.com/trust-tech/go-trustmachine/consensus"
	"github.com/trust-tech/go-trustmachine/consensus/clique"
	"github.com/trust-tech/go-trustmachine/consensus/entrustash"
	"github.com/trust-tech/go-trustmachine/core"
	"github.com/trust-tech/go-trustmachine/core/types"
	"github.com/trust-tech/go-trustmachine/core/vm"
	"github.com/trust-tech/go-trustmachine/entrust/downloader"
	"github.com/trust-tech/go-trustmachine/entrust/filters"
	"github.com/trust-tech/go-trustmachine/entrust/gasprice"
	"github.com/trust-tech/go-trustmachine/entrustdb"
	"github.com/trust-tech/go-trustmachine/event"
	"github.com/trust-tech/go-trustmachine/internal/entrustapi"
	"github.com/trust-tech/go-trustmachine/log"
	"github.com/trust-tech/go-trustmachine/miner"
	"github.com/trust-tech/go-trustmachine/node"
	"github.com/trust-tech/go-trustmachine/p2p"
	"github.com/trust-tech/go-trustmachine/params"
	"github.com/trust-tech/go-trustmachine/rlp"
	"github.com/trust-tech/go-trustmachine/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
}

// Trustmachine implements the Trustmachine full node service.
type Trustmachine struct {
	chainConfig *params.ChainConfig
	// Channel for shutting down the service
	shutdownChan  chan bool    // Channel for shutting down the trustmachine
	stopDbUpgrade func() error // stop chain db sequential key upgrade
	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer
	// DB interfaces
	chainDb entrustdb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	ApiBackend *EntrustApiBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	trustbase common.Address

	networkId     uint64
	netRPCService *entrustapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and trustbase)
}

func (s *Trustmachine) AddLesServer(ls LesServer) {
	s.lesServer = ls
}

// New creates a new Trustmachine object (including the
// initialisation of the common Trustmachine object)
func New(ctx *node.ServiceContext, config *Config) (*Trustmachine, error) {
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run entrust.Trustmachine in light sync mode, use les.LightTrustmachine")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}

	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	stopDbUpgrade := upgradeDeduplicateData(chainDb)
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	entrust := &Trustmachine{
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, config, chainConfig, chainDb),
		shutdownChan:   make(chan bool),
		stopDbUpgrade:  stopDbUpgrade,
		networkId:      config.NetworkId,
		gasPrice:       config.GasPrice,
		trustbase:      config.Trustbase,
	}

	if err := addMipmapBloomBins(chainDb); err != nil {
		return nil, err
	}
	log.Info("Initialising Trustmachine protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if !config.SkipBcVersionCheck {
		bcVersion := core.GetBlockChainVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run gotrust upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		core.WriteBlockChainVersion(chainDb, core.BlockChainVersion)
	}

	vmConfig := vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
	entrust.blockchain, err = core.NewBlockChain(chainDb, entrust.chainConfig, entrust.engine, entrust.eventMux, vmConfig)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		entrust.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	newPool := core.NewTxPool(config.TxPool, entrust.chainConfig, entrust.EventMux(), entrust.blockchain.State, entrust.blockchain.GasLimit)
	entrust.txPool = newPool

	maxPeers := config.MaxPeers
	if config.LightServ > 0 {
		// if we are running a light server, limit the number of ENTRUST peers so that we reserve some space for incoming LES connections
		// temporary solution until the new peer connectivity API is finished
		halfPeers := maxPeers / 2
		maxPeers -= config.LightPeers
		if maxPeers < halfPeers {
			maxPeers = halfPeers
		}
	}

	if entrust.protocolManager, err = NewProtocolManager(entrust.chainConfig, config.SyncMode, config.NetworkId, maxPeers, entrust.eventMux, entrust.txPool, entrust.engine, entrust.blockchain, chainDb); err != nil {
		return nil, err
	}

	entrust.miner = miner.New(entrust, entrust.chainConfig, entrust.EventMux(), entrust.engine)
	entrust.miner.SetExtra(makeExtraData(config.ExtraData))

	entrust.ApiBackend = &EntrustApiBackend{entrust, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	entrust.ApiBackend.gpo = gasprice.NewOracle(entrust.ApiBackend, gpoParams)

	return entrust, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"gotrust",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (entrustdb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*entrustdb.LDBDatabase); ok {
		db.Meter("entrust/db/chaindata/")
	}
	return db, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an Trustmachine service
func CreateConsensusEngine(ctx *node.ServiceContext, config *Config, chainConfig *params.ChainConfig, db entrustdb.Database) consensus.Engine {
	// If proof-of-authority is requested, set it up
	if chainConfig.Clique != nil {
		return clique.New(chainConfig.Clique, db)
	}
	// Otherwise assume proof-of-work
	switch {
	case config.PowFake:
		log.Warn("Entrustash used in fake mode")
		return entrustash.NewFaker()
	case config.PowTest:
		log.Warn("Entrustash used in test mode")
		return entrustash.NewTester()
	case config.PowShared:
		log.Warn("Entrustash used in shared mode")
		return entrustash.NewShared()
	default:
		engine := entrustash.New(ctx.ResolvePath(config.EntrustashCacheDir), config.EntrustashCachesInMem, config.EntrustashCachesOnDisk,
			config.EntrustashDatasetDir, config.EntrustashDatasetsInMem, config.EntrustashDatasetsOnDisk)
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}

// APIs returns the collection of RPC services the trustmachine package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *Trustmachine) APIs() []rpc.API {
	apis := entrustapi.GetAPIs(s.ApiBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "entrust",
			Version:   "1.0",
			Service:   NewPublicTrustmachineAPI(s),
			Public:    true,
		}, {
			Namespace: "entrust",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "entrust",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "entrust",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *Trustmachine) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *Trustmachine) Trustbase() (eb common.Address, err error) {
	s.lock.RLock()
	trustbase := s.trustbase
	s.lock.RUnlock()

	if trustbase != (common.Address{}) {
		return trustbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			return accounts[0].Address, nil
		}
	}
	return common.Address{}, fmt.Errorf("trustbase address must be explicitly specified")
}

// set in js console via admin interface or wrapper from cli flags
func (self *Trustmachine) SetTrustbase(trustbase common.Address) {
	self.lock.Lock()
	self.trustbase = trustbase
	self.lock.Unlock()

	self.miner.SetTrustbase(trustbase)
}

func (s *Trustmachine) StartMining(local bool) error {
	eb, err := s.Trustbase()
	if err != nil {
		log.Error("Cannot start mining without trustbase", "err", err)
		return fmt.Errorf("trustbase missing: %v", err)
	}
	if clique, ok := s.engine.(*clique.Clique); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("Trustbase account unavailable locally", "err", err)
			return fmt.Errorf("singer missing: %v", err)
		}
		clique.Authorize(eb, wallet.SignHash)
	}
	if local {
		// If local (CPU) mining is started, we can disable the transaction rejection
		// mechanism introduced to speed sync times. CPU mining on mainnet is ludicrous
		// so noone will ever hit this path, whereas marking sync done on CPU mining
		// will ensure that private networks work in single miner mode too.
		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)
	}
	go s.miner.Start(eb)
	return nil
}

func (s *Trustmachine) StopMining()         { s.miner.Stop() }
func (s *Trustmachine) IsMining() bool      { return s.miner.Mining() }
func (s *Trustmachine) Miner() *miner.Miner { return s.miner }

func (s *Trustmachine) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *Trustmachine) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *Trustmachine) TxPool() *core.TxPool               { return s.txPool }
func (s *Trustmachine) EventMux() *event.TypeMux           { return s.eventMux }
func (s *Trustmachine) Engine() consensus.Engine           { return s.engine }
func (s *Trustmachine) ChainDb() entrustdb.Database            { return s.chainDb }
func (s *Trustmachine) IsListening() bool                  { return true } // Always listening
func (s *Trustmachine) EntrustVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Trustmachine) NetVersion() uint64                 { return s.networkId }
func (s *Trustmachine) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *Trustmachine) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	} else {
		return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
	}
}

// Start implements node.Service, starting all internal goroutines needed by the
// Trustmachine protocol implementation.
func (s *Trustmachine) Start(srvr *p2p.Server) error {
	s.netRPCService = entrustapi.NewPublicNetAPI(srvr, s.NetVersion())

	s.protocolManager.Start()
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Trustmachine protocol.
func (s *Trustmachine) Stop() error {
	if s.stopDbUpgrade != nil {
		s.stopDbUpgrade()
	}
	s.blockchain.Stop()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
