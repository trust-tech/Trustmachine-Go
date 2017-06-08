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

// Contains the metrics collected by the fetcher.

package fetcher

import (
	"github.com/trust-tech/go-trustmachine/metrics"
)

var (
	propAnnounceInMeter   = metrics.NewMeter("entrust/fetcher/prop/announces/in")
	propAnnounceOutTimer  = metrics.NewTimer("entrust/fetcher/prop/announces/out")
	propAnnounceDropMeter = metrics.NewMeter("entrust/fetcher/prop/announces/drop")
	propAnnounceDOSMeter  = metrics.NewMeter("entrust/fetcher/prop/announces/dos")

	propBroadcastInMeter   = metrics.NewMeter("entrust/fetcher/prop/broadcasts/in")
	propBroadcastOutTimer  = metrics.NewTimer("entrust/fetcher/prop/broadcasts/out")
	propBroadcastDropMeter = metrics.NewMeter("entrust/fetcher/prop/broadcasts/drop")
	propBroadcastDOSMeter  = metrics.NewMeter("entrust/fetcher/prop/broadcasts/dos")

	headerFetchMeter = metrics.NewMeter("entrust/fetcher/fetch/headers")
	bodyFetchMeter   = metrics.NewMeter("entrust/fetcher/fetch/bodies")

	headerFilterInMeter  = metrics.NewMeter("entrust/fetcher/filter/headers/in")
	headerFilterOutMeter = metrics.NewMeter("entrust/fetcher/filter/headers/out")
	bodyFilterInMeter    = metrics.NewMeter("entrust/fetcher/filter/bodies/in")
	bodyFilterOutMeter   = metrics.NewMeter("entrust/fetcher/filter/bodies/out")
)
