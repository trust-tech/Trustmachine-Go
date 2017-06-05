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

// Contains all the wrappers from the node package to support client side node
// management on mobile platforms.

package gotrust

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/ThePleasurable/go-trustmachine/core"
	"github.com/ThePleasurable/go-trustmachine/entrust"
	"github.com/ThePleasurable/go-trustmachine/entrust/downloader"
	"github.com/ThePleasurable/go-trustmachine/entrustclient"
	"github.com/ThePleasurable/go-trustmachine/entruststats"
	"github.com/ThePleasurable/go-trustmachine/les"
	"github.com/ThePleasurable/go-trustmachine/node"
	"github.com/ThePleasurable/go-trustmachine/p2p"
	"github.com/ThePleasurable/go-trustmachine/p2p/nat"
	"github.com/ThePleasurable/go-trustmachine/params"
	whisper "github.com/ThePleasurable/go-trustmachine/whisper/whisperv5"
)

// NodeConfig represents the collection of configuration values to fine tune the Gotrust
// node embedded into a mobile process. The available values are a subset of the
// entire API provided by go-trustmachine to reduce the maintenance surface and dev
// complexity.
type NodeConfig struct {
	// Bootstrap nodes used to establish connectivity with the rest of the network.
	BootstrapNodes *Enodes

	// MaxPeers is the maximum number of peers that can be connected. If this is
	// set to zero, then only the configured static and trusted peers can connect.
	MaxPeers int

	// TrustmachineEnabled specifies whether the node should run the Trustmachine protocol.
	TrustmachineEnabled bool

	// TrustmachineNetworkID is the network identifier used by the Trustmachine protocol to
	// decide if remote peers should be accepted or not.
	TrustmachineNetworkID int64 // uint64 in truth, but Java can't handle that...

	// TrustmachineGenesis is the genesis JSON to use to seed the blockchain with. An
	// empty genesis state is equivalent to using the mainnet's state.
	TrustmachineGenesis string

	// TrustmachineDatabaseCache is the system memory in MB to allocate for database caching.
	// A minimum of 16MB is always reserved.
	TrustmachineDatabaseCache int

	// TrustmachineNetStats is a netstats connection string to use to report various
	// chain, transaction and node stats to a monitoring server.
	//
	// It has the form "nodename:secret@host:port"
	TrustmachineNetStats string

	// WhisperEnabled specifies whether the node should run the Whisper protocol.
	WhisperEnabled bool
}

// defaultNodeConfig contains the default node configuration values to use if all
// or some fields are missing from the user's specified list.
var defaultNodeConfig = &NodeConfig{
	BootstrapNodes:        FoundationBootnodes(),
	MaxPeers:              25,
	TrustmachineEnabled:       true,
	TrustmachineNetworkID:     1,
	TrustmachineDatabaseCache: 16,
}

// NewNodeConfig creates a new node option set, initialized to the default values.
func NewNodeConfig() *NodeConfig {
	config := *defaultNodeConfig
	return &config
}

// Node represents a Gotrust Trustmachine node instance.
type Node struct {
	node *node.Node
}

// NewNode creates and configures a new Gotrust node.
func NewNode(datadir string, config *NodeConfig) (stack *Node, _ error) {
	// If no or partial configurations were specified, use defaults
	if config == nil {
		config = NewNodeConfig()
	}
	if config.MaxPeers == 0 {
		config.MaxPeers = defaultNodeConfig.MaxPeers
	}
	if config.BootstrapNodes == nil || config.BootstrapNodes.Size() == 0 {
		config.BootstrapNodes = defaultNodeConfig.BootstrapNodes
	}
	// Create the empty networking stack
	nodeConf := &node.Config{
		Name:        clientIdentifier,
		Version:     params.Version,
		DataDir:     datadir,
		KeyStoreDir: filepath.Join(datadir, "keystore"), // Mobile should never use internal keystores!
		P2P: p2p.Config{
			NoDiscovery:      true,
			DiscoveryV5:      true,
			DiscoveryV5Addr:  ":0",
			BootstrapNodesV5: config.BootstrapNodes.nodes,
			ListenAddr:       ":0",
			NAT:              nat.Any(),
			MaxPeers:         config.MaxPeers,
		},
	}
	rawStack, err := node.New(nodeConf)
	if err != nil {
		return nil, err
	}

	var genesis *core.Genesis
	if config.TrustmachineGenesis != "" {
		// Parse the user supplied genesis spec if not mainnet
		genesis = new(core.Genesis)
		if err := json.Unmarshal([]byte(config.TrustmachineGenesis), genesis); err != nil {
			return nil, fmt.Errorf("invalid genesis spec: %v", err)
		}
		// If we have the testnet, hard code the chain configs too
		if config.TrustmachineGenesis == TestnetGenesis() {
			genesis.Config = params.TestnetChainConfig
			if config.TrustmachineNetworkID == 1 {
				config.TrustmachineNetworkID = 3
			}
		}
	}
	// Register the Trustmachine protocol if requested
	if config.TrustmachineEnabled {
		entrustConf := entrust.DefaultConfig
		entrustConf.Genesis = genesis
		entrustConf.SyncMode = downloader.LightSync
		entrustConf.NetworkId = uint64(config.TrustmachineNetworkID)
		entrustConf.DatabaseCache = config.TrustmachineDatabaseCache
		if err := rawStack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
			return les.New(ctx, &entrustConf)
		}); err != nil {
			return nil, fmt.Errorf("trustmachine init: %v", err)
		}
		// If netstats reporting is requested, do it
		if config.TrustmachineNetStats != "" {
			if err := rawStack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
				var lesServ *les.LightTrustmachine
				ctx.Service(&lesServ)

				return entruststats.New(config.TrustmachineNetStats, nil, lesServ)
			}); err != nil {
				return nil, fmt.Errorf("netstats init: %v", err)
			}
		}
	}
	// Register the Whisper protocol if requested
	if config.WhisperEnabled {
		if err := rawStack.Register(func(*node.ServiceContext) (node.Service, error) { return whisper.New(), nil }); err != nil {
			return nil, fmt.Errorf("whisper init: %v", err)
		}
	}
	return &Node{rawStack}, nil
}

// Start creates a live P2P node and starts running it.
func (n *Node) Start() error {
	return n.node.Start()
}

// Stop terminates a running node along with all it's services. In the node was
// not started, an error is returned.
func (n *Node) Stop() error {
	return n.node.Stop()
}

// GetTrustmachineClient retrieves a client to access the Trustmachine subsystem.
func (n *Node) GetTrustmachineClient() (client *TrustmachineClient, _ error) {
	rpc, err := n.node.Attach()
	if err != nil {
		return nil, err
	}
	return &TrustmachineClient{entrustclient.NewClient(rpc)}, nil
}

// GetNodeInfo gathers and returns a collection of metadata known about the host.
func (n *Node) GetNodeInfo() *NodeInfo {
	return &NodeInfo{n.node.Server().NodeInfo()}
}

// GetPeersInfo returns an array of metadata objects describing connected peers.
func (n *Node) GetPeersInfo() *PeerInfos {
	return &PeerInfos{n.node.Server().PeersInfo()}
}
