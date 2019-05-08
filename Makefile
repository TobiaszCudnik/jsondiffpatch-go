build:
	cd src && go build main.go
	mv src/main jdp-go

build-wasm:
	cd wasm && GOOS=js GOARCH=wasm go build main.go
	mv wasm/main jdp-go.wasm

server:
	echo "npm i http-server"
	http-server

run:
	./jdp-go
