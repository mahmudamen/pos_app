# posgo-ops - the POS.Go control plane as a terminal application.
#
# This is a separate Go module so the API's dependency graph stays free of
# FluffyUI. Every target here works on this module alone.

MODULE      := posgo-ops
BIN         := bin/posgo-ops
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS     := -s -w -X main.version=$(VERSION)

.PHONY: all build run test race vet fmt fmt-check check tidy clean install

all: check build

build:
	go build -trimpath -ldflags '$(LDFLAGS)' -o $(BIN) ./cmd/posgo-ops

# Run against a local backend. The console needs the SaaS platform tenant id,
# which the API returns in its login payload; POSGO_OPS_TENANT_ID overrides it.
run:
	go run ./cmd/posgo-ops --api http://127.0.0.1:8080

test:
	go test -count=1 ./...

race:
	go test -race -count=1 ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

# check is what CI should run: format, vet, then the suite.
check: fmt-check vet test

tidy:
	go mod tidy

clean:
	rm -rf bin

install: build
	install -m 0755 $(BIN) $(DESTDIR)/usr/local/bin/posgo-ops
