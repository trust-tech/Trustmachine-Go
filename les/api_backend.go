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

package les

import (
	"context"
	"math/big"

	"github.com/trust-tech/go-trustmachine/accounts"
	"github.com/trust-tech/go-trustmachine/common"
	"github.com/trust-tech/go-trustmachine/common/math"
	"github.com/trust-tech/go-trustmachine/core"
	"github.com/trust-tech/go-trustmachine/core/state"
	"github.com/trust-tech/go-trustmachine/core/types"
	"github.com/trust-tech/go-trustmachine/core/vm"
	"github.com/trust-tech/go-trustmachine/entrust/downloader"
	"github.com/trust-tech/go-trustmachine/entrust/gasprice"
	"github.com/trust-tech/go-trustmachine/entrustdb"
	"github.com/trust-tech/go-trustmachine/event"
	"github.com/trust-tech/go-trustmachine/light"
	"github.com/trust-tech/go-trustmachine/params"
	"github.com/trust-tech/go-trustmachine/rpc"
)

type LesApiBackend struct {
	entrust *LightTrustmachine
	gpo *gasprice.Oracle
}

func (b *LesApiBackend) ChainConfig() *params.ChainConfig {
	return b.entrust.chainConfig
}

func (b *LesApiBackend) CurrentBlock() *types.Block {
	return types.NewBlockWithHeader(b.entrust.BlockChain().CurrentHeader())
}

func (b *LesApiBackend) SetHead(number uint64) {
	b.entrust.protocolManager.downloader.Cancel()
	b.entrust.blockchain.SetHead(number)
}

func (b *LesApiBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	if blockNr == rpc.LatestBlockNumber || blockNr == rpc.PendingBlockNumber {
		return b.entrust.blockchain.CurrentHeader(), nil
	}

	return b.entrust.blockchain.GetHeaderByNumberOdr(ctx, uint64(blockNr))
}

func (b *LesApiBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, err
	}
	return b.GetBlock(ctx, header.Hash())
}

func (b *LesApiBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	return light.NewState(ctx, header, b.entrust.odr), header, nil
}

func (b *LesApiBackend) GetBlock(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return b.entrust.blockchain.GetBlockByHash(ctx, blockHash)
}

func (b *LesApiBackend) GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	return light.GetBlockReceipts(ctx, b.entrust.odr, blockHash, core.GetBlockNumber(b.entrust.chainDb, blockHash))
}

func (b *LesApiBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.entrust.blockchain.GetTdByHash(blockHash)
}

func (b *LesApiBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	context := core.NewEVMContext(msg, header, b.entrust.blockchain, nil)
	return vm.NewEVM(context, state, b.entrust.chainConfig, vmCfg), state.Error, nil
}

func (b *LesApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.entrust.txPool.Add(ctx, signedTx)
}

func (b *LesApiBackend) RemoveTx(txHash common.Hash) {
	b.entrust.txPool.RemoveTx(txHash)
}

func (b *LesApiBackend) GetPoolTransactions() (types.Transactions, error) {
	return b.entrust.txPool.GetTransactions()
}

func (b *LesApiBackend) GetPoolTransaction(txHash common.Hash) *types.Transaction {
	return b.entrust.txPool.GetTransaction(txHash)
}

func (b *LesApiBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.entrust.txPool.GetNonce(ctx, addr)
}

func (b *LesApiBackend) Stats() (pending int, queued int) {
	return b.entrust.txPool.Stats(), 0
}

func (b *LesApiBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.entrust.txPool.Content()
}

func (b *LesApiBackend) Downloader() *downloader.Downloader {
	return b.entrust.Downloader()
}

func (b *LesApiBackend) ProtocolVersion() int {
	return b.entrust.LesVersion() + 10000
}

func (b *LesApiBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *LesApiBackend) ChainDb() entrustdb.Database {
	return b.entrust.chainDb
}

func (b *LesApiBackend) EventMux() *event.TypeMux {
	return b.entrust.eventMux
}

func (b *LesApiBackend) AccountManager() *accounts.Manager {
	return b.entrust.accountManager
}
