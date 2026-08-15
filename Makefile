# Load .env file if it exists.
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

GO        := go
MODULE    := github.com/ronnieholm/resellerloyalty
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 windows/amd64
SERVICE   := service
SIMULATOR := simulator
TESTDIR   := bin/tests
GOOSE     := $(GO) run github.com/pressly/goose/v3/cmd/goose -dir ./migrations
DB_NAMES  := $(DB_LOCAL_NAME) $(DB_LOCAL_INTEGRATION_TEST_NAME)

.PHONY: all
all: build

# Developer build without embedded version metadata.
.PHONY: build
build:
	CGO_ENABLED=0 $(GO) build -o bin/$(SERVICE) ./cmd/$(SERVICE)
	CGO_ENABLED=0 $(GO) build -o bin/$(SIMULATOR) ./cmd/$(SIMULATOR)

    # Build but don't run tests to detect compiler errors only.
	@echo "building tests"
	@mkdir -p $(TESTDIR)
	@for pkg in $(shell go list ./...); do \
		out=$$(echo $$pkg | tr '/' '-' ) ; \
		go test -c -o $(TESTDIR)/$$out.test $$pkg 1>/dev/null || exit $$? ; \
	done

# Test without race detector.
.PHONY: test
test:
	# The -p option doesn't have a long-form name. If it did, it would be
	# --parallel-packages. So -p controls how many packages are run in parallel
	# whereas -parallel controls how many tests are run in parallel inside a
	# single package. -parallel only applies to tests marked with t.Parallel().
	$(GO) test ./... -v -p=1 -parallel 1 -shuffle=on 

# Test with race detector.
#.PHONY: test-race
test-race:
	$(GO) test ./... -v -race -p=1 -parallel=1 -shuffle=on 

.PHONY: lint
lint:
	go tool golangci-lint run ./...

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: generate
generate:
	$(GO) generate ./...

# Generate build info to embed in binaries.
gen-build-info:
	@$(eval VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0"))
	@$(eval BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ"))
	@echo "VERSION=$(VERSION)"
	@echo "BUILD_TIME=$(BUILD_TIME)"

# Release for multiple platforms.
.PHONY: release
release: clean gen-build-info
	@mkdir -p dist
	@for p in $(PLATFORMS); do \
		OS=$${p%/*}; \
		ARCH=$${p#*/}; \
		for exe in $(SERVICE) $(SIMULATOR); do \
			OUT=dist/$${exe}-$${OS}-$${ARCH}; \
			if [ "$${OS}" = "windows" ]; then OUT=$${OUT}.exe; fi; \
			CGO_ENABLED=0 GOOS=$${OS} GOARCH=$${ARCH} $(GO) build -ldflags \
				"-X '$(MODULE)/internal/build.Version=$(VERSION)' \
				 -X '$(MODULE)/internal/build.BuildTime=$(BUILD_TIME)'" -o $${OUT} ./cmd/$${exe}; \
		done; \
	done

# Build Docker image
#DOCKER_IMG := youruser/yourimage:latest
#.PHONY: docker
#docker: build
#	docker build -t $(DOCKER_IMG) .

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf bin dist

db-create:
	@for db in $(DB_NAMES); do \
		echo "Checking database: $$db"; \
		docker exec reseller-loyalty-postgres psql -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = '$$db'" | grep -q 1 || \
		docker exec reseller-loyalty-postgres psql -U postgres -c "CREATE DATABASE $$db;"; \
	done

db-drop:
	@for db in $(DB_NAMES); do \
		echo "Dropping $$db if it exists"; \
		docker exec reseller-loyalty-postgres psql -U postgres -c "DROP DATABASE IF EXISTS $$db;"; \
	done

db-migrate-status:
	@for db in $(DB_NAMES); do \
		echo "Status for $$db"; \
		$(GOOSE) postgres "$(DB_SERVER_URL)/$$db" status; \
	done

db-migrate-up:
	@for db in $(DB_NAMES); do \
		echo "Migrating $$db up"; \
		$(GOOSE) postgres "$(DB_SERVER_URL)/$$db" up || exit 1; \
	done

# goose supports two types of migrations:
#   sql: The most common. You write standard SQL statements.
#   go: Used for complex migrations that SQL can't handle easily, e.g.,
#       migrating data from a blob storage to a database, or complex password
#       hashing.
#
# By specifying sql, goose creates a .sql file rather than a .go file.
db-migrate-create:
	@read -p "Migration name: " name; \
	$(GOOSE) create $$name sql