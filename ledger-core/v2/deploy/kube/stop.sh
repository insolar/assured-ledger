#!/usr/bin/env bash
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
ARTIFACTS_DIR="/tmp/insolar"
LOG_DIR="$ARTIFACTS_DIR/logs"
set -x

stop_network() {
  kubectl delete -f "$DIR/generated.yaml"
}

save_logs_to_files() {
  LOG_DIR="$ARTIFACTS_DIR/logs"
  rm -rf $LOG_DIR
  mkdir -p $LOG_DIR
  kubectl -n insolar logs virtual-0 >"$LOG_DIR/virtual-0"
  kubectl -n insolar logs virtual-1 >"$LOG_DIR/virtual-1"
  kubectl -n insolar logs virtual-2 >"$LOG_DIR/virtual-2"
  kubectl -n insolar logs virtual-3 >"$LOG_DIR/virtual-3"
  kubectl -n insolar logs virtual-4 >"$LOG_DIR/virtual-4"
}

echo "Stopping insolar"
save_logs_to_files
echo "Logs saved to $LOG_DIR"
stop_network
echo "Insolar stoped"
set +x
