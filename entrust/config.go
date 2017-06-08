// Copyright 2017 The go-trustmachine Authors
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

package entrust

import (
	"math/big"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/trust-tech/go-trustmachine/common"
	"github.com/trust-tech/go-trustmachine/common/hexutil"
	"github.com/trust-tech/go-trustmachine/core"
	"github.com/trust-tech/go-trustmachine/entrust/downloader"
	"github.com/trust-tech/go-trustmachine/entrust/gasprice"
	"github.com/trust-tech/go-trustmachine/params"
)

// DefaultConfig contains default settings for use on the Trustmachine main net.
var DefaultConfig = Config{
	SyncMode:             downloader.FastSync,
	EntrustashCacheDir:       "entrustash",
	EntrustashCachesInMem:    2,
	EntrustashCachesOnDisk:   3,
	EntrustashDatasetsInMem:  1,
	EntrustashDatasetsOnDisk: 2,
	NetworkId:            1,
	LightPeers:           20,
	DatabaseCache:        128,
	GasPrice:             big.NewInt(18 * params.Shannon),

	TxPool: core.DefaultTxPoolConfig,
	GPO: gasprice.Config{
		Blocks:     10,
		Percentile: 50,
	},
}

func init() {
	home := os.Getenv("HOME")
	if home == "" {
		if user, err := user.Current(); err == nil {
			home = user.HomeDir
		}
	}
	if runtime.GOOS == "windows" {
		DefaultConfig.EntrustashDatasetDir = filepath.Join(home, "AppData", "Entrustash")
	} else {
		DefaultConfig.EntrustashDatasetDir = filepath.Join(home, ".entrustash")
	}
}

//go:generate gencodec -type Config -field-override configMarshaling -formats toml -out gen_config.go

type Config struct {
	// The genesis block, which is inserted if the database is empty.
	// If nil, the Trustmachine main net block is used.
	Genesis *core.Genesis `toml:",omitempty"`

	// Protocol options
	NetworkId uint64 // Network ID to use for selecting peers to connect to
	SyncMode  downloader.SyncMode

	// Light client options
	LightServ  int `toml:",omitempty"` // Maximum percentage of time allowed for serving LES requests
	LightPeers int `toml:",omitempty"` // Maximum number of LES client peers
	MaxPeers   int `toml:"-"`          // Maximum number of global peers

	// Database options
	SkipBcVersionCheck bool `toml:"-"`
	DatabaseHandles    int  `toml:"-"`
	DatabaseCache      int

	// Mining-related options
	Trustbase    common.Address `toml:",omitempty"`
	MinerThreads int            `toml:",omitempty"`
	ExtraData    []byte         `toml:",omitempty"`
	GasPrice     *big.Int

	// Entrustash options
	EntrustashCacheDir       string
	EntrustashCachesInMem    int
	EntrustashCachesOnDisk   int
	EntrustashDatasetDir     string
	EntrustashDatasetsInMem  int
	EntrustashDatasetsOnDisk int

	// Transaction pool options
	TxPool core.TxPoolConfig

	// Gas Price Oracle options
	GPO gasprice.Config

	// Enables tracking of SHA3 preimages in the VM
	EnablePreimageRecording bool

	// Miscellaneous options
	DocRoot   string `toml:"-"`
	PowFake   bool   `toml:"-"`
	PowTest   bool   `toml:"-"`
	PowShared bool   `toml:"-"`
}

type configMarshaling struct {
	ExtraData hexutil.Bytes
}
