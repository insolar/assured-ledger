#!/usr/bin/env bash
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
KUBECTL=${KUBECTL:-"kubectl"}
USE_MANIFESTS=${USE_MANIFESTS:-"local"}

stop_network() {
  $KUBECTL delete -k "$DIR/$USE_MANIFESTS/"
}

echo "Stopping insolar"
stop_network
echo "Insolar stopped"
