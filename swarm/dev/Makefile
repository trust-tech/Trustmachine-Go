.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/trust-tech/go-trustmachine/cmd/swarm
	go build -o bin/gotrust     github.com/trust-tech/go-trustmachine/cmd/gotrust
	go build -o bin/bootnode github.com/trust-tech/go-trustmachine/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/trust-tech/go-trustmachine/swarm/...
