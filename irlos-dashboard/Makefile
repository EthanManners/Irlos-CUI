# SPDX-License-Identifier: GPL-3.0-or-later
# Makefile for irlosd — Irlos dashboard

BINARY   := ./bin/irlosd
INSTALL  := /usr/local/bin/irlosd
MODULE   := github.com/ethanmanners/irlos-dashboard
CMD      := ./cmd/irlosd

LDFLAGS  := -ldflags="-s -w"

.PHONY: all build install test clean vet fmt

all: build

build:
	@mkdir -p ./bin
	go build $(LDFLAGS) -o $(BINARY) $(CMD)
	@echo "Built $(BINARY)"

install: build
	sudo install -m 0755 $(BINARY) $(INSTALL)
	@echo "Installed to $(INSTALL)"

test:
	go test ./...

vet:
	go vet ./...

fmt:
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "Unformatted files:"; \
		gofmt -l .; \
		exit 1; \
	fi

clean:
	rm -rf ./bin

# Static binary (useful for deployment where CGO may be unavailable).
# Note: NVML and D-Bus require CGO on Linux; use the standard build for Irlos.
build-static:
	@mkdir -p ./bin
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY)-static $(CMD)
