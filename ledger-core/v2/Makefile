export GO111MODULE ?= on
export GOSUMDB ?= sum.golang.org
export GOFLAGS ?= -mod=vendor

BIN_DIR ?= bin
ARTIFACTS_DIR ?= .artifacts
INSGOCC = insgocc

ALL_PACKAGES = ./...

COVERPROFILE ?= coverage.txt

BUILD_NUMBER := $(TRAVIS_BUILD_NUMBER)
# skip git parsing commands if no git
ifneq ("$(wildcard ../../.git)", "")
	BUILD_DATE ?= $(shell ./scripts/dev/git-date-time.sh -d)
	BUILD_TIME ?= $(shell ./scripts/dev/git-date-time.sh -t)
	BUILD_HASH ?= $(shell git rev-parse --short HEAD)
	BUILD_VERSION ?= $(shell git describe --tags 2>/dev/null)
endif

ifeq ($(BUILD_VERSION),)
	BUILD_VERSION = $(BUILD_HASH)
endif
DOCKER_BASE_IMAGE_TAG ?= $(BUILD_VERSION)

GOPATH ?= `go env GOPATH`
LDFLAGS += -X github.com/insolar/assured-ledger/ledger-core/v2/version.Version=${BUILD_VERSION}
LDFLAGS += -X github.com/insolar/assured-ledger/ledger-core/v2/version.BuildNumber=${BUILD_NUMBER}
LDFLAGS += -X github.com/insolar/assured-ledger/ledger-core/v2/version.BuildDate=${BUILD_DATE}
LDFLAGS += -X github.com/insolar/assured-ledger/ledger-core/v2/version.BuildTime=${BUILD_TIME}
LDFLAGS += -X github.com/insolar/assured-ledger/ledger-core/v2/version.GitHash=${BUILD_HASH}

.PHONY: all
all: vendor clean submodule pre-build build ## cleanup, install deps, (re)generate all code and build all binaries

include Makefile.build
include Makefile.testing

.PHONY: submodule
submodule: ## init git submodule
	git submodule init
	git submodule update

.PHONY: lint
lint: ## CI lint
	golangci-lint run

.PHONY: clean
clean: ## run all cleanup tasks
	go clean $(ALL_PACKAGES)
	rm -f $(COVERPROFILE)
	rm -rf $(BIN_DIR)
	rm -rf $(ARTIFACTS_DIR)
	./scripts/insolard/launchnet.sh -l

.PHONY: install-build-tools
install-build-tools: ## install tools for codegen
	go run ./scripts/build/ls-tools.go | xargs -tI % go install -v %

.PHONY: pre-build
pre-build: install-build-tools generate regen-builtin ## install dependencies, (re)generates all code

.PHONY: generate
generate: ## run go generate
	##go get github.com/rigidus/in-solar/
	GOFLAGS="" go generate -x $(ALL_PACKAGES)

.PHONY: vendor
vendor: ## update vendor dependencies
	rm -rf vendor
	go mod vendor

.PHONY: regen-proxies
CONTRACTS = $(wildcard application/contract/*)
regen-proxies: $(INSGOCC) ## regen contracts proxies
	$(foreach c, $(CONTRACTS), $(BIN_DIR)/$(INSGOCC) proxy application/contract/$(notdir $(c))/$(notdir $(c)).go; )

.PHONY: generate-protobuf
generate-protobuf: ## generate protobuf structs
	protoc -I./vendor -I./ --gogoslick_out=./ network/node/internal/node/node.proto
	protoc -I./vendor -I./ --gogoslick_out=./ insolar/record/record.proto
	protoc -I./vendor -I./ --gogoslick_out=./ insolar/jet/jet.proto
	protoc -I./vendor -I./ --gogoslick_out=./ insolar/node.proto
	protoc -I./vendor -I./ --gogoslick_out=./ ledger/drop/drop.proto
	protoc -I./vendor -I./ --gogoslick_out=./ insolar/record/record.proto
	protoc -I./vendor -I./ --gogoslick_out=./ --proto_path=${GOPATH}/src insolar/payload/payload.r0.proto
	protoc -I./vendor -I./ --gogoslick_out=./ --proto_path=${GOPATH}/src insolar/payload/payload.proto
	protoc -I./vendor -I./ --gogoslick_out=./ insolar/pulse/pulse.proto
	protoc -I./vendor -I./ --gogoslick_out=./ --proto_path=${GOPATH}/src network/hostnetwork/packet/packet.proto
	protoc -I./vendor -I./ --gogoslick_out=./ --proto_path=${GOPATH}/src network/consensus/adapters/candidate/profile.proto
	protoc -I./vendor -I./ --gogoslick_out=./ network/servicenetwork/watermillmessage.proto
	protoc -I./vendor -I./ --gogoslick_out=./ ledger/heavy/executor/jetinfo.proto
	protoc -I./vendor -I./ --gogoslick_out=./ instrumentation/instracer/span_data.proto
	protoc -I./vendor -I/usr/local/include -I./ \
			-I$(GOPATH)/src \
			--gogoslick_out=plugins=grpc:./  \
			ledger/heavy/exporter/record_exporter.proto
	protoc -I./vendor -I/usr/local/include -I./ \
			-I$(GOPATH)/src \
			--gogoslick_out=plugins=grpc:./  \
			ledger/heavy/exporter/pulse_exporter.proto

.PHONY: regen-builtin
regen-builtin: $(INSGOCC) ## regenerate builtin contracts code
	$(BIN_DIR)/$(INSGOCC) regen-builtin

.PHONY: generate-introspector-proto
generate-introspector-proto: ## generate grpc api code and mocks for introspector
	protoc -I/usr/local/include -I./ \
		-I$(GOPATH)/src \
		-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--go_out=plugins=grpc:./  \
		--grpc-gateway_out=logtostderr=true:. \
		--swagger_out=logtostderr=true:. \
		instrumentation/introspector/introproto/*.proto
	GOPATH=`go env GOPATH` go generate -x ./instrumentation/introspector

.PHONY: prepare-inrospector-proto
prepare-inrospector-proto: ## install tools required for grpc development
	go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
	go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
	go get -u github.com/golang/protobuf/protoc-gen-go

.PHONY: application-git-config
application-git-config: ## change remote repo url from HTTPS to SSH in 'application' submodule
	perl -i.bak -pe 's!url\s*=\s*https?://github.com/!url = git\@github.com:!' .git/modules/application/config

$(ARTIFACTS_DIR):
	mkdir -p $(ARTIFACTS_DIR)

.PHONY: help
help: ## display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make \033[36m<target>\033[0m\n"}  \
		/^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-30s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)
