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

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/trust-tech/go-trustmachine/metrics"
)

var (
	headerInMeter      = metrics.NewMeter("entrust/downloader/headers/in")
	headerReqTimer     = metrics.NewTimer("entrust/downloader/headers/req")
	headerDropMeter    = metrics.NewMeter("entrust/downloader/headers/drop")
	headerTimeoutMeter = metrics.NewMeter("entrust/downloader/headers/timeout")

	bodyInMeter      = metrics.NewMeter("entrust/downloader/bodies/in")
	bodyReqTimer     = metrics.NewTimer("entrust/downloader/bodies/req")
	bodyDropMeter    = metrics.NewMeter("entrust/downloader/bodies/drop")
	bodyTimeoutMeter = metrics.NewMeter("entrust/downloader/bodies/timeout")

	receiptInMeter      = metrics.NewMeter("entrust/downloader/receipts/in")
	receiptReqTimer     = metrics.NewTimer("entrust/downloader/receipts/req")
	receiptDropMeter    = metrics.NewMeter("entrust/downloader/receipts/drop")
	receiptTimeoutMeter = metrics.NewMeter("entrust/downloader/receipts/timeout")

	stateInMeter   = metrics.NewMeter("entrust/downloader/states/in")
	stateDropMeter = metrics.NewMeter("entrust/downloader/states/drop")
)
