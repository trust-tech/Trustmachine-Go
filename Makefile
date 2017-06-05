# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: gotrust android ios gotrust-cross evm all test clean
.PHONY: gotrust-linux gotrust-linux-386 gotrust-linux-amd64 gotrust-linux-mips64 gotrust-linux-mips64le
.PHONY: gotrust-linux-arm gotrust-linux-arm-5 gotrust-linux-arm-6 gotrust-linux-arm-7 gotrust-linux-arm64
.PHONY: gotrust-darwin gotrust-darwin-386 gotrust-darwin-amd64
.PHONY: gotrust-windows gotrust-windows-386 gotrust-windows-amd64

GOBIN = build/bin
GO ?= latest

gotrust:
	build/env.sh go run build/ci.go install ./cmd/gotrust
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gotrust\" to launch gotrust."

evm:
	build/env.sh go run build/ci.go install ./cmd/evm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/evm\" to start the evm."

all:
	build/env.sh go run build/ci.go install

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/gotrust.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Gotrust.framework\" to use the library."

test: all
	build/env.sh go run build/ci.go test

clean:
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/jteeuwen/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go install ./cmd/abigen

# Cross Compilation Targets (xgo)

gotrust-cross: gotrust-linux gotrust-darwin gotrust-windows gotrust-android gotrust-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/gotrust-*

gotrust-linux: gotrust-linux-386 gotrust-linux-amd64 gotrust-linux-arm gotrust-linux-mips64 gotrust-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/gotrust-linux-*

gotrust-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/gotrust
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/gotrust-linux-* | grep 386

gotrust-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/gotrust
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gotrust-linux-* | grep amd64

gotrust-linux-arm: gotrust-linux-arm-5 gotrust-linux-arm-6 gotrust-linux-arm-7 gotrust-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/gotrust-linux-* | grep arm

gotrust-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/gotrust
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/gotrust-linux-* | grep arm-5

gotrust-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/gotrust
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/gotrust-linux-* | grep arm-6

gotrust-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/gotrust
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/gotrust-linux-* | grep arm-7

gotrust-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/gotrust
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/gotrust-linux-* | grep arm64

gotrust-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/gotrust
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/gotrust-linux-* | grep mips

gotrust-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/gotrust
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/gotrust-linux-* | grep mipsle

gotrust-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/gotrust
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/gotrust-linux-* | grep mips64

gotrust-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/gotrust
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/gotrust-linux-* | grep mips64le

gotrust-darwin: gotrust-darwin-386 gotrust-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/gotrust-darwin-*

gotrust-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/gotrust
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/gotrust-darwin-* | grep 386

gotrust-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/gotrust
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gotrust-darwin-* | grep amd64

gotrust-windows: gotrust-windows-386 gotrust-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/gotrust-windows-*

gotrust-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/gotrust
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/gotrust-windows-* | grep 386

gotrust-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/gotrust
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gotrust-windows-* | grep amd64
