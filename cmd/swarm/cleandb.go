// Copyright 2017 The go-trustmachine Authors
// This file is part of go-trustmachine.
//
// go-trustmachine is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-trustmachine is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-trustmachine. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"github.com/trust-tech/go-trustmachine/cmd/utils"
	"github.com/trust-tech/go-trustmachine/swarm/storage"
	"gopkg.in/urfave/cli.v1"
)

func cleandb(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) != 1 {
		utils.Fatalf("Need path to chunks database as the first and only argument")
	}

	chunkDbPath := args[0]
	hash := storage.MakeHashFunc("SHA3")
	dbStore, err := storage.NewDbStore(chunkDbPath, hash, 10000000, 0)
	if err != nil {
		utils.Fatalf("Cannot initialise dbstore: %v", err)
	}
	dbStore.Cleanup()
}
