.PHONY: build clean test install package nfpm-init nfpm-package nfpm-package-all nfpm-check nfpm-install

BINARY=debian-composer-go
VERSION=$(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/debian-composer

install: build
	sudo cp bin/$(BINARY) /usr/local/bin/debian-composer-go

clean:
	rm -rf bin/
	rm -rf dist/

test:
	go test ./...

run: build
	./bin/$(BINARY) --help

dev:
	go build -o bin/$(BINARY) ./cmd/debian-composer && ./bin/$(BINARY) --help

# ============================================================================
# nFPM Packaging Targets
# ============================================================================

nfpm-check:
	@command -v nfpm >/dev/null 2>&1 || { \
		echo "nFPM not found. Installing..."; \
		go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest; \
	}

nfpm-init: nfpm-check
	@if [ ! -f nfpm.yaml ]; then \
		nfpm init -f nfpm.yaml; \
	else \
		echo "nfpm.yaml already exists"; \
	fi

nfpm-package: nfpm-check build
	@mkdir -p dist
	@export VERSION=$(VERSION) GOARCH=amd64 && \
	nfpm package --packager deb --target dist/debian-composer_$${VERSION}_linux_amd64.deb
	@echo "Package created: dist/debian-composer_$${VERSION}_linux_amd64.deb"

nfpm-package-all: nfpm-check build
	@mkdir -p dist
	@export VERSION=$(VERSION) && \
	GOARCH=amd64 nfpm package --packager deb --target dist/debian-composer_$${VERSION}_linux_amd64.deb && \
	GOARCH=arm64 nfpm package --packager deb --target dist/debian-composer_$${VERSION}_linux_arm64.deb && \
	echo "Packages created in dist/"

package: nfpm-package

# Install nFPM explicitly
nfpm-install:
	go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
