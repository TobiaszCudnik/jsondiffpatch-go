build:
	cd src && go build main.go
	mv src/main jdp-go

build-wasm:
	cd src/wasm && GOOS=js GOARCH=wasm go build main.go
	mv src/wasm/main jdp-go.wasm

build-benchmark-go:
	cd benchmarks/go && go build main.go
	mv benchmarks/go/main benchmark-go

build-benchmark-node:
	cd benchmarks/node && npm i

server:
	echo "npm i http-server"
	http-server

run:
	./jdp-go

benchmark-go:
	./benchmark-go

benchmark-node:
	node benchmarks/node/main.js

.PHONY: benchmark-go