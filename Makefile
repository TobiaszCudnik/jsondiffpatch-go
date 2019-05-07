build:
	cd src && go build main.go
	mv src/main jdp-go

run:
	./jdp-go
