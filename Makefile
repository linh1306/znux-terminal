BINARY := znux
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
GOCACHE ?= /tmp/znux-go-cache

.PHONY: all build install test clean

all: build

build:
	GOCACHE=$(GOCACHE) go build -o dist/$(BINARY) ./cmd

install: 
	go build -o /home/nguyenlinh13602/.local/bin/znux ./cmd

test:
	GOCACHE=$(GOCACHE) go test ./...

clean:
	rm -f dist/$(BINARY)
