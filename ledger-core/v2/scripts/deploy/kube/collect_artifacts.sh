#!/usr/bin/env bash
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
KUBECTL=${KUBECTL:-"kubectl"}
ARTIFACTS_DIR=${ARTIFACTS_DIR:-"/tmp/insolar"}
LOG_DIR="$ARTIFACTS_DIR/logs"

save_logs_to_files() {
  LOG_DIR="$ARTIFACTS_DIR/logs"
  rm -rf "$LOG_DIR"
  mkdir -p "$LOG_DIR"
  $KUBECTL -n insolar logs bootstrap >"$LOG_DIR/bootstrap"
  $KUBECTL -n insolar logs virtual-0 >"$LOG_DIR/virtual-0"
  $KUBECTL -n insolar logs virtual-1 >"$LOG_DIR/virtual-1"
  $KUBECTL -n insolar logs virtual-2 >"$LOG_DIR/virtual-2"
  $KUBECTL -n insolar logs virtual-3 >"$LOG_DIR/virtual-3"
  $KUBECTL -n insolar logs virtual-4 >"$LOG_DIR/virtual-4"
}

save_logs_to_files
echo "Logs saved to $LOG_DIR"
