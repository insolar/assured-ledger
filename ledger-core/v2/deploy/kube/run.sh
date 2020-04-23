#!/usr/bin/env bash
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
NUM_DISCOVERY_NODES=5
INSOLAR_IMAGE="insolar/assured-ledger:latest"
ARTIFACTS_DIR="/tmp/insolar"
set -x
check_environment() {
  command -v docker >/dev/null 2>&1 || {
    echo >&2 "docker is required. Aborting."
    exit 1
  }
  command -v kubectl >/dev/null 2>&1 || {
    echo >&2 "kubectl is required. Aborting."
    exit 1
  }
 check_docker_images
}

# Delete this after image templating will be done, and images will be in insolar hub
check_docker_images(){
  if [ "$(docker images $INSOLAR_IMAGE -q)" = "" ]; then
    echo >&2 "make sure you made 'make docker_build'"
    exit 1
  fi
}

run_network() {
  kubectl apply -f $DIR 2>&1 || {
    echo >&2 "kubectl apply failed. Aborting."
    exit 1
  }
}

wait_for_complete_network_state() {
  set +x
  while true; do
    if [ "$(kubectl -n insolar get po bootstrap -o jsonpath="{.status.phase}")" = "Succeeded" ]; then
      break
    fi
    sleep 1s
  done

  echo "bootstrap completed"

  while true; do
    num=$(kubectl -n insolar logs --tail=10 services/pulsewatcher | grep -c "CompleteNetworkState")
    echo "$num/$NUM_DISCOVERY_NODES discovery nodes ready"
    if [[ "$num" -eq "$NUM_DISCOVERY_NODES" ]]; then
      break
    fi
    sleep 2s
  done
  set -x
}

copy_bootstrap_config_to_temp() {
  mkdir -p $ARTIFACTS_DIR
  kubectl -n insolar get cm bootstrap-yaml -o jsonpath='{.data.bootstrap\.yaml}' >"$ARTIFACTS_DIR/bootstrap.yaml"
}

stop_network() {
  kubectl delete -f $DIR
}

collect_logs() {
  log_dir="$ARTIFACTS_DIR/logs"
  rm -rf $log_dir
  mkdir -p $log_dir
  kubectl -n insolar logs heavy-0 >"$log_dir/heavy-0"
  kubectl -n insolar logs light-0 >"$log_dir/light-0"
  kubectl -n insolar logs light-1 >"$log_dir/light-1"
  kubectl -n insolar logs virtual-0 >"$log_dir/virtual-0"
  kubectl -n insolar logs virtual-1 >"$log_dir/virtual-1"
}

check_environment
echo "Starting insolar"
run_network
wait_for_complete_network_state
copy_bootstrap_config_to_temp
echo "Insolar started"
set +x
