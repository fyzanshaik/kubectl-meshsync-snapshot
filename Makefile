.PHONY: build install clean

BINARY_NAME=kubectl-meshsync_snapshot

build:
	go build -o bin/$(BINARY_NAME) ./cmd/kubectl-meshsync_snapshot

install: build
	cp bin/$(BINARY_NAME) ~/bin/$(BINARY_NAME)

clean:
	rm -rf bin/