# 1) build step (approx local build time ~4m w/o cache)
ARG GOLANG_VERSION=1.14
FROM golang:${GOLANG_VERSION} AS build

ADD . /go/src/github.com/insolar/assured-ledger
WORKDIR /go/src/github.com/insolar/assured-ledger

RUN make vendor

# pass build variables as arguments to avoid adding .git directory
ARG BUILD_NUMBER
ARG BUILD_DATE
ARG BUILD_TIME
ARG BUILD_HASH
ARG BUILD_VERSION
# build step
RUN BUILD_NUMBER=${BUILD_NUMBER} \
    BUILD_DATE=${BUILD_DATE} \
    BUILD_TIME=${BUILD_TIME} \
    BUILD_HASH=${BUILD_HASH} \
    BUILD_VERSION=${BUILD_VERSION} \
    make build-short

FROM debian:buster-slim
WORKDIR /go/src/github.com/insolar/assured-ledger
RUN  set -eux; \
     groupadd -r insolar --gid=999; \
     useradd -r -g insolar --uid=999 --shell=/bin/bash insolar; \
     apt-get update; \
     apt-get install -y curl
COPY --from=build /go/src/github.com/insolar/assured-ledger/application/api/spec/api-exported.yaml /app/api-exported.yaml

COPY --from=build \
    /go/src/github.com/insolar/assured-ledger/bin/insolar \
    /go/src/github.com/insolar/assured-ledger/bin/insolard \
    /go/src/github.com/insolar/assured-ledger/bin/pulsard \
    /go/src/github.com/insolar/assured-ledger/bin/pulsewatcher \
    /usr/local/bin/
