#!/usr/bin/make -f

BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
LEDGER_ENABLED ?= true
BINDIR ?= $(GOPATH)/bin
BUILDDIR ?= $(CURDIR)/build
DOCKER := $(shell which docker)

# don't override user values of COMMIT and VERSION
ifeq (,$(COMMIT))
  COMMIT := $(shell git log -1 --format='%H')
endif

ifeq (,$(VERSION))
  VERSION := $(shell git describe --tags)
  # if VERSION is empty, then populate it with branch's name and raw commit hash
  ifeq (,$(VERSION))
    VERSION := $(BRANCH)-$(COMMIT)
  endif
endif

TM_VERSION := $(shell go list -m github.com/cometbft/cometbft | sed 's:.* ::')

export CGO_ENABLED = 1
export GO111MODULE = on

# process build tags

build_tags = netgo
ifeq ($(LEDGER_ENABLED),true)
	ifeq ($(OS),Windows_NT)
		GCCEXE = $(shell where gcc.exe 2> NUL)
		ifeq ($(GCCEXE),)
			$(error gcc.exe not installed for ledger support, please install or set LEDGER_ENABLED=false)
		else
			build_tags += ledger
		endif
	else
		UNAME_S = $(shell uname -s)
		ifeq ($(UNAME_S),OpenBSD)
			$(warning OpenBSD detected, disabling ledger support (https://github.com/cosmos/cosmos-sdk/issues/1988))
		else
			GCC = $(shell command -v gcc 2> /dev/null)
			ifeq ($(GCC),)
				$(error gcc not installed for ledger support, please install or set LEDGER_ENABLED=false)
			else
				build_tags += ledger
			endif
		endif
	endif
endif

ifeq (cleveldb,$(findstring cleveldb,$(COSMOS_BUILD_OPTIONS)))
  build_tags += gcc
endif
# handle rocksdb
define ROCKSDB_INSTRUCTIONS

################################################################
RocksDB support requires the RocksDB shared library and headers.

macOS (Homebrew):
  brew install rocksdb
  export CGO_CFLAGS="-I$$(brew --prefix rocksdb)/include"
  export CGO_LDFLAGS="-L$$(brew --prefix rocksdb)/lib"

See https://github.com/rockset/rocksdb-cloud/blob/master/INSTALL.md for custom setups.
################################################################

endef

ifeq (rocksdb,$(findstring rocksdb,$(COSMOS_BUILD_OPTIONS)))
  $(info $(ROCKSDB_INSTRUCTIONS))
  build_tags += rocksdb grocksdb_clean_link

  ifeq ($(shell uname -s),Darwin)
    ifneq (,$(shell command -v brew 2>/dev/null))
      ROCKSDB_PREFIX := $(shell brew --prefix rocksdb 2>/dev/null)
      ifneq (,$(ROCKSDB_PREFIX))
        CGO_CFLAGS ?= -I$(ROCKSDB_PREFIX)/include
        CGO_LDFLAGS ?= -L$(ROCKSDB_PREFIX)/lib
        export CGO_CFLAGS CGO_LDFLAGS
      else
        $(warning rocksdb not installed via Homebrew; skipping CGO flags)
      endif
    else
      $(warning Homebrew not found; skipping rocksdb CGO flags)
    endif
  endif
endif
ifeq (boltdb,$(findstring boltdb,$(COSMOS_BUILD_OPTIONS)))
  build_tags += boltdb
endif

build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

whitespace :=
whitespace += $(whitespace)
comma := ,
build_tags_comma_sep := $(subst $(whitespace),$(comma),$(build_tags))

# process linker flags

ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=initia \
		  -X github.com/cosmos/cosmos-sdk/version.AppName=initiad \
		  -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
		  -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
		  -X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(build_tags_comma_sep)" \
			-X github.com/cometbft/cometbft/version.TMCoreSemVer=$(TM_VERSION)

ifeq (,$(findstring nostrip,$(COSMOS_BUILD_OPTIONS)))
  ldflags += -w -s
endif
ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'
# check for nostrip option
ifeq (,$(findstring nostrip,$(COSMOS_BUILD_OPTIONS)))
  BUILD_FLAGS += -trimpath
endif

# The below include contains the tools and runsim targets.
include contrib/devtools/Makefile

all: tools install lint test

build: go.sum
ifeq ($(OS),Windows_NT)
	exit 1
else
	go build -mod=readonly $(BUILD_FLAGS) -o build/initiad ./cmd/initiad
endif

build-vendor: go.sum
ifeq ($(OS),Windows_NT)
	exit 1
else
	go build -mod=vendor $(BUILD_FLAGS) -o build/initiad ./cmd/initiad
endif

build-linux:
	mkdir -p $(BUILDDIR)
	docker build --no-cache --tag initia/initiad \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		./
	docker create --name temp initia/initiad:latest
	docker cp temp:/usr/local/bin/initiad $(BUILDDIR)/
	docker rm temp

install: go.sum 
	go install -mod=readonly $(BUILD_FLAGS) ./cmd/initiad

update-swagger-docs: statik
	$(BINDIR)/statik -src=client/docs/swagger-ui -dest=client/docs -f -m
	@if [ -n "$(git status --porcelain)" ]; then \
        echo "\033[91mSwagger docs are out of sync!!!\033[0m";\
        exit 1;\
    else \
        echo "\033[92mSwagger docs are in sync\033[0m";\
    fi

.PHONY: build build-linux install update-swagger-docs

###############################################################################
###                                Protobuf                                 ###
###############################################################################

protoVer=0.14.0
protoImageName=ghcr.io/cosmos/proto-builder:$(protoVer)
protoImage=$(DOCKER) run --rm -v $(CURDIR):/workspace  --workdir /workspace $(protoImageName)

proto-all: proto-format proto-lint proto-gen proto-swagger-gen proto-pulsar-gen

proto-gen:
	@echo "Generating Protobuf files"
	@$(protoImage) sh ./scripts/protocgen.sh

proto-swagger-gen:
	@echo "Generating Swagger files"
	@$(protoImage) sh ./scripts/protoc-swagger-gen.sh
	$(MAKE) update-swagger-docs

proto-pulsar-gen:
	@echo "Generating Dep-Inj Protobuf files"
	@$(protoImage) sh ./scripts/protocgen-pulsar.sh

proto-format:
	@$(protoImage) find ./ -name "*.proto" -exec buf format {} -w \;

proto-lint:
	@$(protoImage) buf lint --error-format=json ./proto

proto-check-breaking:
	@$(protoImage) buf breaking --against $(HTTPS_GIT)#branch=main

.PHONY: proto-all proto-gen proto-swagger-gen proto-pulsar-gen proto-format proto-lint proto-check-breaking

########################################
### Tools & dependencies

go-mod-cache: go.sum
	@echo "--> Download go modules to local cache"
	@go mod download

go.sum: go.mod
	@echo "--> Ensure dependencies have not been modified"
	@go mod verify

draw-deps:
	@# requires brew install graphviz or apt-get install graphviz
	go install github.com/RobotsAndPencils/goviz
	@goviz -i ./cmd/initiad -d 2 | dot -Tpng -o dependency-graph.png

distclean: clean tools-clean
clean:
	rm -rf \
    $(BUILDDIR)/ \
    artifacts/ \
    tmp-swagger-gen/

.PHONY: distclean clean


###############################################################################
###                           Tests 
###############################################################################

test: test-unit

test-all: test-unit test-race test-cover

test-unit:
	@VERSION=$(VERSION) go test -mod=readonly -tags='ledger test_ledger_mock' ./...

test-race:
	@VERSION=$(VERSION) go test -mod=readonly -race -tags='ledger test_ledger_mock' ./...

test-cover:
	@go test -mod=readonly -timeout 30m -race -coverprofile=coverage.txt -covermode=atomic -tags='ledger test_ledger_mock' ./...

test-e2e:
	@go test ./integration-tests/e2e/... -mod=readonly -timeout 30m -tags='e2e' -count=1

benchmark-e2e:
	cd integration-tests/e2e && go test -v -tags benchmark -run TestBenchmark -timeout 30m -count=1 ./benchmark/

benchmark:
	@go test -timeout 20m -mod=readonly -bench=. ./... 

.PHONY: test test-all test-cover test-unit test-race test-e2e benchmark benchmark-e2e

###############################################################################
###                                Linting                                  ###
###############################################################################

lint:
	golangci-lint run --timeout=15m --tests=false

lint-fix:
	golangci-lint run --fix --timeout=15m --tests=false
.PHONY: lint lint-fix

format:
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/docs/statik/statik.go" -not -path "./tests/mocks/*" -not -path "./api/*" -not -name '*.pb.go' | xargs gofmt -w -s
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/docs/statik/statik.go" -not -path "./tests/mocks/*" -not -path "./api/*" -not -name '*.pb.go' | xargs misspell -w
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/docs/statik/statik.go" -not -path "./tests/mocks/*" -not -path "./api/*" -not -name '*.pb.go' | xargs goimports -w -local github.com/cosmos/cosmos-sdk
.PHONY: format

###############################################################################
###                               Testnet                                   ###
###############################################################################

testnet-gen:
	bash ./scripts/testnet-generation.sh
.PHONY: testnet-gen
