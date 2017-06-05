// Copyright 2015 The go-trustmachine Authors
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
	"context"
	"math/big"

	"github.com/ThePleasurable/go-trustmachine/accounts"
	"github.com/ThePleasurable/go-trustmachine/common"
	"github.com/ThePleasurable/go-trustmachine/common/math"
	"github.com/ThePleasurable/go-trustmachine/core"
	"github.com/ThePleasurable/go-trustmachine/core/state"
	"github.com/ThePleasurable/go-trustmachine/core/types"
	"github.com/ThePleasurable/go-trustmachine/core/vm"
	"github.com/ThePleasurable/go-trustmachine/entrust/downloader"
	"github.com/ThePleasurable/go-trustmachine/entrust/gasprice"
	"github.com/ThePleasurable/go-trustmachine/entrustdb"
	"github.com/ThePleasurable/go-trustmachine/event"
	"github.com/ThePleasurable/go-trustmachine/internal/entrustapi"
	"github.com/ThePleasurable/go-trustmachine/params"
	"github.com/ThePleasurable/go-trustmachine/rpc"
)

// EntrustApiBackend implements entrustapi.Backend for full nodes
type EntrustApiBackend struct {
	entrust *Trustmachine
	gpo *gasprice.Oracle
}

func (b *EntrustApiBackend) ChainConfig() *params.ChainConfig {
	return b.entrust.chainConfig
}

func (b *EntrustApiBackend) CurrentBlock() *types.Block {
	return b.entrust.blockchain.CurrentBlock()
}

func (b *EntrustApiBackend) SetHead(number uint64) {
	b.entrust.protocolManager.downloader.Cancel()
	b.entrust.blockchain.SetHead(number)
}

func (b *EntrustApiBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.entrust.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.entrust.blockchain.CurrentBlock().Header(), nil
	}
	return b.entrust.blockchain.GetHeaderByNumber(uint64(blockNr)), nil
}

func (b *EntrustApiBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.entrust.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.entrust.blockchain.CurrentBlock(), nil
	}
	return b.entrust.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (b *EntrustApiBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (entrustapi.State, *types.Header, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block, state := b.entrust.miner.Pending()
		return EntrustApiState{state}, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := b.entrust.BlockChain().StateAt(header.Root)
	return EntrustApiState{stateDb}, header, err
}

func (b *EntrustApiBackend) GetBlock(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return b.entrust.blockchain.GetBlockByHash(blockHash), nil
}

func (b *EntrustApiBackend) GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	return core.GetBlockReceipts(b.entrust.chainDb, blockHash, core.GetBlockNumber(b.entrust.chainDb, blockHash)), nil
}

func (b *EntrustApiBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.entrust.blockchain.GetTdByHash(blockHash)
}

func (b *EntrustApiBackend) GetEVM(ctx context.Context, msg core.Message, state entrustapi.State, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	statedb := state.(EntrustApiState).state
	from := statedb.GetOrNewStateObject(msg.From())
	from.SetBalance(math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.entrust.BlockChain(), nil)
	return vm.NewEVM(context, statedb, b.entrust.chainConfig, vmCfg), vmError, nil
}

func (b *EntrustApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	b.entrust.txMu.Lock()
	defer b.entrust.txMu.Unlock()

	b.entrust.txPool.SetLocal(signedTx)
	return b.entrust.txPool.Add(signedTx)
}

func (b *EntrustApiBackend) RemoveTx(txHash common.Hash) {
	b.entrust.txMu.Lock()
	defer b.entrust.txMu.Unlock()

	b.entrust.txPool.Remove(txHash)
}

func (b *EntrustApiBackend) GetPoolTransactions() (types.Transactions, error) {
	b.entrust.txMu.Lock()
	defer b.entrust.txMu.Unlock()

	pending, err := b.entrust.txPool.Pending()
	if err != nil {
		return nil, err
	}

	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *EntrustApiBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	b.entrust.txMu.Lock()
	defer b.entrust.txMu.Unlock()

	return b.entrust.txPool.Get(hash)
}

func (b *EntrustApiBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	b.entrust.txMu.Lock()
	defer b.entrust.txMu.Unlock()

	return b.entrust.txPool.State().GetNonce(addr), nil
}

func (b *EntrustApiBackend) Stats() (pending int, queued int) {
	b.entrust.txMu.Lock()
	defer b.entrust.txMu.Unlock()

	return b.entrust.txPool.Stats()
}

func (b *EntrustApiBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	b.entrust.txMu.Lock()
	defer b.entrust.txMu.Unlock()

	return b.entrust.TxPool().Content()
}

func (b *EntrustApiBackend) Downloader() *downloader.Downloader {
	return b.entrust.Downloader()
}

func (b *EntrustApiBackend) ProtocolVersion() int {
	return b.entrust.EntrustVersion()
}

func (b *EntrustApiBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *EntrustApiBackend) ChainDb() entrustdb.Database {
	return b.entrust.ChainDb()
}

func (b *EntrustApiBackend) EventMux() *event.TypeMux {
	return b.entrust.EventMux()
}

func (b *EntrustApiBackend) AccountManager() *accounts.Manager {
	return b.entrust.AccountManager()
}

type EntrustApiState struct {
	state *state.StateDB
}

func (s EntrustApiState) GetBalance(ctx context.Context, addr common.Address) (*big.Int, error) {
	return s.state.GetBalance(addr), nil
}

func (s EntrustApiState) GetCode(ctx context.Context, addr common.Address) ([]byte, error) {
	return s.state.GetCode(addr), nil
}

func (s EntrustApiState) GetState(ctx context.Context, a common.Address, b common.Hash) (common.Hash, error) {
	return s.state.GetState(a, b), nil
}

func (s EntrustApiState) GetNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return s.state.GetNonce(addr), nil
}
