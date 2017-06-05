// Copyright 2016 The go-trustmachine Authors
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
	"crypto/rand"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ThePleasurable/go-trustmachine/params"
)

const (
	ipcAPIs  = "admin:1.0 debug:1.0 entrust:1.0 miner:1.0 net:1.0 personal:1.0 rpc:1.0 shh:1.0 txpool:1.0 web3:1.0"
	httpAPIs = "entrust:1.0 net:1.0 rpc:1.0 web3:1.0"
)

// Tests that a node embedded within a console can be started up properly and
// then terminated by closing the input stream.
func TestConsoleWelcome(t *testing.T) {
	coinbase := "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"

	// Start a gotrust console, make sure it's cleaned up and terminate the console
	gotrust := runGotrust(t,
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none",
		"--trustbase", coinbase, "--shh",
		"console")

	// Gather all the infos the welcome message needs to contain
	gotrust.setTemplateFunc("goos", func() string { return runtime.GOOS })
	gotrust.setTemplateFunc("goarch", func() string { return runtime.GOARCH })
	gotrust.setTemplateFunc("gover", runtime.Version)
	gotrust.setTemplateFunc("gotrustver", func() string { return params.Version })
	gotrust.setTemplateFunc("niltime", func() string { return time.Unix(0, 0).Format(time.RFC1123) })
	gotrust.setTemplateFunc("apis", func() string { return ipcAPIs })

	// Verify the actual welcome message to the required template
	gotrust.expect(`
Welcome to the Gotrust JavaScript console!

instance: Gotrust/v{{gotrustver}}/{{goos}}-{{goarch}}/{{gover}}
coinbase: {{.Trustbase}}
at block: 0 ({{niltime}})
 datadir: {{.Datadir}}
 modules: {{apis}}

> {{.InputLine "exit"}}
`)
	gotrust.expectExit()
}

// Tests that a console can be attached to a running node via various means.
func TestIPCAttachWelcome(t *testing.T) {
	// Configure the instance for IPC attachement
	coinbase := "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"
	var ipc string
	if runtime.GOOS == "windows" {
		ipc = `\\.\pipe\gotrust` + strconv.Itoa(trulyRandInt(100000, 999999))
	} else {
		ws := tmpdir(t)
		defer os.RemoveAll(ws)
		ipc = filepath.Join(ws, "gotrust.ipc")
	}
	// Note: we need --shh because testAttachWelcome checks for default
	// list of ipc modules and shh is included there.
	gotrust := runGotrust(t,
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none",
		"--trustbase", coinbase, "--shh", "--ipcpath", ipc)

	time.Sleep(2 * time.Second) // Simple way to wait for the RPC endpoint to open
	testAttachWelcome(t, gotrust, "ipc:"+ipc, ipcAPIs)

	gotrust.interrupt()
	gotrust.expectExit()
}

func TestHTTPAttachWelcome(t *testing.T) {
	coinbase := "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"
	port := strconv.Itoa(trulyRandInt(1024, 65536)) // Yeah, sometimes this will fail, sorry :P
	gotrust := runGotrust(t,
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none",
		"--trustbase", coinbase, "--rpc", "--rpcport", port)

	time.Sleep(2 * time.Second) // Simple way to wait for the RPC endpoint to open
	testAttachWelcome(t, gotrust, "http://localhost:"+port, httpAPIs)

	gotrust.interrupt()
	gotrust.expectExit()
}

func TestWSAttachWelcome(t *testing.T) {
	coinbase := "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"
	port := strconv.Itoa(trulyRandInt(1024, 65536)) // Yeah, sometimes this will fail, sorry :P

	gotrust := runGotrust(t,
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none",
		"--trustbase", coinbase, "--ws", "--wsport", port)

	time.Sleep(2 * time.Second) // Simple way to wait for the RPC endpoint to open
	testAttachWelcome(t, gotrust, "ws://localhost:"+port, httpAPIs)

	gotrust.interrupt()
	gotrust.expectExit()
}

func testAttachWelcome(t *testing.T, gotrust *testgotrust, endpoint, apis string) {
	// Attach to a running gotrust note and terminate immediately
	attach := runGotrust(t, "attach", endpoint)
	defer attach.expectExit()
	attach.stdin.Close()

	// Gather all the infos the welcome message needs to contain
	attach.setTemplateFunc("goos", func() string { return runtime.GOOS })
	attach.setTemplateFunc("goarch", func() string { return runtime.GOARCH })
	attach.setTemplateFunc("gover", runtime.Version)
	attach.setTemplateFunc("gotrustver", func() string { return params.Version })
	attach.setTemplateFunc("trustbase", func() string { return gotrust.Trustbase })
	attach.setTemplateFunc("niltime", func() string { return time.Unix(0, 0).Format(time.RFC1123) })
	attach.setTemplateFunc("ipc", func() bool { return strings.HasPrefix(endpoint, "ipc") })
	attach.setTemplateFunc("datadir", func() string { return gotrust.Datadir })
	attach.setTemplateFunc("apis", func() string { return apis })

	// Verify the actual welcome message to the required template
	attach.expect(`
Welcome to the Gotrust JavaScript console!

instance: Gotrust/v{{gotrustver}}/{{goos}}-{{goarch}}/{{gover}}
coinbase: {{trustbase}}
at block: 0 ({{niltime}}){{if ipc}}
 datadir: {{datadir}}{{end}}
 modules: {{apis}}

> {{.InputLine "exit" }}
`)
	attach.expectExit()
}

// trulyRandInt generates a crypto random integer used by the console tests to
// not clash network ports with other tests running cocurrently.
func trulyRandInt(lo, hi int) int {
	num, _ := rand.Int(rand.Reader, big.NewInt(int64(hi-lo)))
	return int(num.Int64()) + lo
}
