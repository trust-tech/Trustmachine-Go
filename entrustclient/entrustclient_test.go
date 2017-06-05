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

package entrustclient

import "github.com/ThePleasurable/go-trustmachine"

// Verify that Client implements the trustmachine interfaces.
var (
	_ = trustmachine.ChainReader(&Client{})
	_ = trustmachine.TransactionReader(&Client{})
	_ = trustmachine.ChainStateReader(&Client{})
	_ = trustmachine.ChainSyncReader(&Client{})
	_ = trustmachine.ContractCaller(&Client{})
	_ = trustmachine.GasEstimator(&Client{})
	_ = trustmachine.GasPricer(&Client{})
	_ = trustmachine.LogFilterer(&Client{})
	_ = trustmachine.PendingStateReader(&Client{})
	// _ = trustmachine.PendingStateEventer(&Client{})
	_ = trustmachine.PendingContractCaller(&Client{})
)
