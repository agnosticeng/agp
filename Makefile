all: test build

build: 
	go generate ./...
	go build -o bin/agp ./cmd

test:
	go test -v ./...

clean:
	rm -rf bin
