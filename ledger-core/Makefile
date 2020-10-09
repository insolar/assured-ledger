export GO111MODULE ?= on
export GOSUMDB ?= sum.golang.org
export GOFLAGS ?= -mod=vendor

BIN_DIR ?= ./bin
ARTIFACTS_DIR ?= .artifacts
INSGOCC = insgocc

ALL_PACKAGES = ./...

COVERPROFILE ?= coverage.txt

BUILD_NUMBER := $(TRAVIS_BUILD_NUMBER)
# skip git parsing commands if no git
ifneq ("$(wildcard ../.git)", "")
	BUILD_DATE ?= $(shell ./scripts/dev/git-date-time.sh -d)
	BUILD_TIME ?= $(shell ./scripts/dev/git-date-time.sh -t)
	BUILD_HASH ?= $(shell git rev-parse --short HEAD)
	BUILD_VERSION ?= $(shell git describe --tags 2>/dev/null)
endif

ifeq ($(BUILD_VERSION),)
	BUILD_VERSION = $(BUILD_HASH)
endif
DOCKER_IMAGE_TAG ?= $(BUILD_VERSION)

GOPATH ?= `go env GOPATH`
LDFLAGS += -X github.com/insolar/assured-ledger/ledger-core/version.Version=${BUILD_VERSION}
LDFLAGS += -X github.com/insolar/assured-ledger/ledger-core/version.BuildNumber=${BUILD_NUMBER}
LDFLAGS += -X github.com/insolar/assured-ledger/ledger-core/version.BuildDate=${BUILD_DATE}
LDFLAGS += -X github.com/insolar/assured-ledger/ledger-core/version.BuildTime=${BUILD_TIME}
LDFLAGS += -X github.com/insolar/assured-ledger/ledger-core/version.GitHash=${BUILD_HASH}

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
	GOFLAGS="" go generate -x $(ALL_PACKAGES)

.PHONY: vendor
vendor: ## update vendor dependencies
	go mod vendor

.PHONY: generate-proto-options
generate-proto-options:
	protoc -I./vendor -I./ --gogofaster_out=Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor:./ insproto/ins.proto

.PHONY: generate-protobuf
generate-protobuf: protoc-gen-ins ## generate protobuf structs
	protoc -I./vendor -I./ --gogoslick_out=./ network/servicenetwork/watermillmessage.proto
	protoc -I./vendor -I./ --gogoslick_out=./ instrumentation/instracer/span_data.proto

	cd rms; protoc -I../vendor -I./ --ins_out=./ --proto_path=${GOPATH}/src proto_*.proto
	cd rms/example; protoc -I../../vendor -I./ --ins_out=./ --proto_path=${GOPATH}/src proto_*.proto

.PHONY: proto
proto: generate-proto-options protoc-gen-ins generate-protobuf

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
