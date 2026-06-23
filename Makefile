BINARY := znux
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
GOCACHE ?= /tmp/znux-go-cache

.PHONY: all build install test clean

all: build

build:
	GOCACHE=$(GOCACHE) go build -o dist/$(BINARY) ./cmd

install: build
	mkdir -p $(BINDIR)
	cp dist/$(BINARY) $(BINDIR)/$(BINARY)
	@echo "Installed $(BINARY) to $(BINDIR)/$(BINARY)"

test:
	GOCACHE=$(GOCACHE) go test ./...

clean:
	rm -f dist/$(BINARY)
