BINARY  := xamltech-site
GO      ?= go
PKG     := ./cmd/server
DIST    := dist
HOST    ?= root@173.249.56.66
SSH     ?= ssh -o BatchMode=yes -i $(HOME)/.ssh/id_ed25519

.PHONY: all build run test vet fmt check clean linux deploy-preview

all: check build

build:
	$(GO) build -o $(BINARY) $(PKG)

run:
	XT_ADDR=127.0.0.1:8090 $(GO) run $(PKG)

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fmt:
	gofmt -w .

check: fmt vet test

# Cross-compile a static linux/amd64 binary for the origin VPS.
linux:
	mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -trimpath -ldflags "-s -w" -o $(DIST)/$(BINARY) $(PKG)

# Install the preview service on the origin (127.0.0.1:8090), no DNS change.
deploy-preview: linux
	$(SSH) $(HOST) 'mkdir -p /opt/xamltech-site && touch /opt/xamltech-site/submissions.jsonl && chown -R www-data:www-data /opt/xamltech-site'
	rsync -az $(DIST)/$(BINARY) deploy/site.env deploy/xamltech-site.service $(HOST):/opt/xamltech-site/
	$(SSH) $(HOST) 'chmod 755 /opt/xamltech-site/$(BINARY) && chmod 644 /opt/xamltech-site/site.env && install -m644 /opt/xamltech-site/xamltech-site.service /etc/systemd/system/xamltech-site.service && systemctl daemon-reload && systemctl enable xamltech-site && systemctl restart xamltech-site && sleep 1 && systemctl is-active xamltech-site && curl -s http://127.0.0.1:8090/health'

clean:
	rm -f $(BINARY)
	rm -rf $(DIST)
