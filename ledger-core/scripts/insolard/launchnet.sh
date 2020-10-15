#!/usr/bin/env bash
set -em
# requires: lsof, awk, sed, grep, pgrep

export GO111MODULE=on

# Changeable environment variables (parameters)
INSOLAR_ARTIFACTS_DIR=${INSOLAR_ARTIFACTS_DIR:-".artifacts"}/
LAUNCHNET_BASE_DIR=${LAUNCHNET_BASE_DIR:-"${INSOLAR_ARTIFACTS_DIR}launchnet"}/

INSOLAR_LOG_FORMATTER=${INSOLAR_LOG_FORMATTER:-"json"}
INSOLAR_LOG_LEVEL=${INSOLAR_LOG_LEVEL:-"debug"}
# we can skip build binaries (by default in CI environment they skips)
SKIP_BUILD=${SKIP_BUILD:-${CI_ENV}}
BUILD_TAGS=${BUILD_TAGS:-'debug'}
EXTRA_BUILD_ARGS=''

# predefined/dependent environment variables

LAUNCHNET_LOGS_DIR=${LAUNCHNET_BASE_DIR}logs/
DISCOVERY_NODE_LOGS=${LAUNCHNET_LOGS_DIR}discoverynodes/

BIN_DIR=bin
INSOLAR_CLI=${BIN_DIR}/insolar
INSOLARD=$BIN_DIR/insolard
KEEPERD=$BIN_DIR/keeperd
PULSARD=$BIN_DIR/pulsard
PULSEWATCHER=$BIN_DIR/pulsewatcher

# DUMP_METRICS_ENABLE enables metrics dump to logs dir after every functest
DUMP_METRICS_ENABLE=${DUMP_METRICS_ENABLE:-"1"}

PULSAR_DATA_DIR=${LAUNCHNET_BASE_DIR}pulsar_data
PULSAR_CONFIG=${LAUNCHNET_BASE_DIR}pulsar.yaml

SCRIPTS_DIR=scripts/insolard/

CONFIGS_DIR=${LAUNCHNET_BASE_DIR}configs/

PULSAR_KEYS=${CONFIGS_DIR}pulsar_keys.json
HEAVY_GENESIS_CONFIG_FILE=${CONFIGS_DIR}heavy_genesis.json
CONTRACTS_PLUGINS_DIR=${LAUNCHNET_BASE_DIR}contracts

DISCOVERY_NODES_DATA=${LAUNCHNET_BASE_DIR}discoverynodes/

DISCOVERY_NODES_HEAVY_DATA=${DISCOVERY_NODES_DATA}1/

BOOTSTRAP_TEMPLATE=${SCRIPTS_DIR}bootstrap_template.yaml
BOOTSTRAP_CONFIG=${LAUNCHNET_BASE_DIR}bootstrap.yaml
BOOTSTRAP_INSOLARD_CONFIG=${LAUNCHNET_BASE_DIR}insolard.yaml

KEEPERD_CONFIG=${LAUNCHNET_BASE_DIR}keeperd.yaml
KEEPERD_LOG=${LAUNCHNET_LOGS_DIR}keeperd.log

PULSEWATCHER_CONFIG=${LAUNCHNET_BASE_DIR}pulsewatcher.yaml

set -x
export INSOLAR_LOG_FORMATTER=${INSOLAR_LOG_FORMATTER}
export INSOLAR_LOG_LEVEL=${INSOLAR_LOG_LEVEL}
{ set +x; } 2>/dev/null

NUM_DISCOVERY_VIRTUAL_NODES=${NUM_DISCOVERY_VIRTUAL_NODES:-20}
NUM_DISCOVERY_LIGHT_NODES=${NUM_DISCOVERY_LIGHT_NODES:-0}
NUM_DISCOVERY_HEAVY_NODES=${NUM_DISCOVERY_HEAVY_NODES:-0}
NUM_DISCOVERY_NODES=$((NUM_DISCOVERY_VIRTUAL_NODES + NUM_DISCOVERY_LIGHT_NODES + NUM_DISCOVERY_HEAVY_NODES))
NUM_NODES=$(sed -n '/^nodes:/,$p' $BOOTSTRAP_TEMPLATE | grep "host:" | grep -cv "#" | tr -d '[:space:]')
echo "discovery+other nodes: ${NUM_DISCOVERY_NODES}+${NUM_NODES}"

for i in $(seq 1 $NUM_DISCOVERY_NODES)
do
    DISCOVERY_NODE_DIRS+=("${DISCOVERY_NODES_DATA}${i}")
done

# LOGROTATOR_ENABLE enables log rotation before every functest start
LOGROTATOR_ENABLE=${LOGROTATOR_ENABLE:-""}
LOGROTATOR=tee
LOGROTATOR_BIN=${LAUNCHNET_BASE_DIR}inslogrotator
if [[ "$LOGROTATOR_ENABLE" == "1" ]]; then
  LOGROTATOR=${LOGROTATOR_BIN}
fi

function join_by { local IFS="$1"; shift; echo "$*"; }

build_logger()
{
    echo "build logger binaries"
    set -x
    pushd scripts/_logger
    GO111MODULE=on go build -o inslogrotator .
    popd
    mv scripts/_logger/inslogrotator "${LOGROTATOR_BIN}"
    { set +x; } 2>/dev/null
}

kill_port()
{
    port=$1
    pids=$(lsof -i :$port | grep "LISTEN\|UDP" | awk '{print $2}')
    for pid in $pids
    do
        echo -n "killing pid $pid at "
        date
        kill $pid
    done
}

kill_all()
{
  echo "kill all processes: insolard, pulsard"
  set +e
  pkill -9 insolard || echo "No insolard running"
  pkill -9 pulsard || echo "No pulsard running"
  pkill -9 keeperd || echo "No keeperd running"
  set -e
}

stop_listening()
{
    echo "stop_listening(): starts ..."
    ports="$ports 58090" # Pulsar

    transport_ports=$( grep "host:" ${BOOTSTRAP_CONFIG} | grep -o ":\d\+" | grep -o "\d\+" | tr '\n' ' ' )
    # keeperd_port=$( grep "listenaddress:" ${KEEPERD_CONFIG} | grep -o ":\d\+" | grep -o "\d\+" | tr '\n' ' ' )
    keeperd_port=""
    ports="$ports $transport_ports $keeperd_port"

    for port in $ports
    do
        echo "killing process using port '$port'"
        kill_port $port
    done

    echo "stop_listening() end."
}

stop_all()
{
  kill_all
}

clear_dirs()
{
    echo "clear_dirs() starts ..."
    set -x
    rm -rfv ${DISCOVERY_NODES_DATA}
    rm -rfv ${LAUNCHNET_LOGS_DIR}
    rm -rfv ${CONTRACTS_PLUGINS_DIR}
    { set +x; } 2>/dev/null

    for i in `seq 1 $NUM_DISCOVERY_NODES`
    do
        set -x
        rm -rfv ${DISCOVERY_NODE_LOGS}${i}
        { set +x; } 2>/dev/null
    done
}

create_required_dirs()
{
    echo "create_required_dirs() starts ..."
    set -x
    mkdir -p ${DISCOVERY_NODES_DATA}certs
    mkdir -p ${CONFIGS_DIR}

    mkdir -p ${PULSAR_DATA_DIR}

    { set +x; } 2>/dev/null

    for i in `seq 1 $NUM_DISCOVERY_NODES`
    do
        set -x
        mkdir -p ${DISCOVERY_NODE_LOGS}${i}
        { set +x; } 2>/dev/null
    done

    echo "create_required_dirs() end."
}

generate_insolard_configs()
{
    echo "generate configs"
    set -x
    go run -mod=vendor scripts/generate_insolar_configs.go --num-virtual-nodes="$NUM_DISCOVERY_VIRTUAL_NODES" \
      --num-light-nodes="$NUM_DISCOVERY_LIGHT_NODES" --num-heavy-nodes="$NUM_DISCOVERY_HEAVY_NODES"
    { set +x; } 2>/dev/null
}

prepare()
{
    echo "prepare() starts ..."
    if [[ "$NO_STOP_LISTENING_ON_PREPARE" != "1" ]]; then
        stop_listening
    fi
    clear_dirs
    create_required_dirs
    echo "prepare() end."
}

build_binaries()
{
    echo "build binaries"
    set -x
    if [ -n "$BUILD_TAGS" ];
      then
        EXTRA_BUILD_ARGS="-tags=\"${BUILD_TAGS}\""
        export EXTRA_BUILD_ARGS
    fi
    make build
    { set +x; } 2>/dev/null
}

rebuild_binaries()
{
    echo "rebuild binaries"
    make clean
    build_binaries
}

generate_pulsar_keys()
{
    echo "generate pulsar keys: ${PULSAR_KEYS}"
    bin/insolar gen-key-pair > ${PULSAR_KEYS}
}

check_working_dir()
{
    echo "check_working_dir() starts ..."
    if ! pwd | grep -q "ledger-core$"
    then
        echo "Run me from insolar root"
        exit 1
    fi
    echo "check_working_dir() end."
}

usage()
{
    echo "usage: $0 [options]"
    echo "possible options: "
    echo -e "\t-h - show help"
    echo -e "\t-g - start launchnet"
    echo -e "\t-d - use conveyor text logger"
    echo -e "\t-b - do bootstrap only and exit, show bootstrap logs"
    echo -e "\t-l - clear all and exit"
    echo -e "\t-C - generate configs only"
    echo -e "\t-p - start without pulsar"
    echo -e "\t-w - start without pulse watcher"
    echo -e "\t-g headless - start launchnet in headless mode"
}

# Flag indicate that we should start network in headless mode
headless=false

process_input_params()
{
    echo "run arguments: $@"

    # shell does not reset OPTIND automatically;
    # it must be manually reset between multiple calls to getopts
    # within the same shell invocation if a new set of parameters is to be used
    OPTIND=1
    while getopts "h?gbdlwpCm" opt; do
        case "$opt" in
        h|\?)
            usage
            exit 0
            ;;
        g)
            if [[ "$OPTARG" == "headless" ]]
                then headless=true
            fi

            GENESIS=1
            bootstrap
            ;;
        d)
            BUILD_TAGS="$BUILD_TAGS debug convlogtxt"
          ;;
        b)
            NO_BOOTSTRAP_LOG_REDIRECT=1
            NO_STOP_LISTENING_ON_PREPARE=${NO_STOP_LISTENING_ON_PREPARE:-"1"}
            bootstrap
            exit 0
            ;;
        l)
            prepare
            exit 0
            ;;
        w)
            watch_pulse=false
            ;;
        p)
            run_pulsar=false
            ;;
        m)
            cloud_mode=true
            ;;
        C)
            generate_insolard_configs
            exit $?
        esac
    done
}

launch_keeperd()
{
    echo "launch_keeperd() starts ..."
    ${KEEPERD} --config=${KEEPERD_CONFIG} \
    &> ${KEEPERD_LOG} &

    echo "launch_keeperd() end."
}

wait_for_complete_network_state()
{
    while true
    do
        num=`scripts/insolard/check_status.sh 2>/dev/null | grep "CompleteNetworkState" | wc -l`
        echo "$num/$NUM_DISCOVERY_NODES discovery nodes ready"
        if [[ "$num" -eq "$NUM_DISCOVERY_NODES" ]]
        then
            break
        fi
        sleep 5s
    done
}

bootstrap()
{
    echo "bootstrap start"
    prepare
    if [[ "$SKIP_BUILD" != "1" ]]; then
        build_binaries
    else
        echo "SKIP: build binaries (SKIP_BUILD=$SKIP_BUILD)"
    fi
    generate_pulsar_keys
    generate_insolard_configs


    echo "start bootstrap ..."
    CMD="${INSOLAR_CLI} bootstrap --config=${BOOTSTRAP_CONFIG} --certificates-out-dir=${DISCOVERY_NODES_DATA}certs"

    GENESIS_EXIT_CODE=0
    set +e
    if [[ "$NO_BOOTSTRAP_LOG_REDIRECT" != "1" ]]; then
        set -x
        ${CMD} &> ${LAUNCHNET_LOGS_DIR}bootstrap.log
        GENESIS_EXIT_CODE=$?
        { set +x; } 2>/dev/null
        echo "bootstrap log: ${LAUNCHNET_LOGS_DIR}bootstrap.log"
    else
        set -x
        ${CMD}
        GENESIS_EXIT_CODE=$?
        { set +x; } 2>/dev/null
    fi
    set -e
    if [[ ${GENESIS_EXIT_CODE} -ne 0 ]]; then
        echo "Genesis failed"
        if [[ "${NO_BOOTSTRAP_LOG_REDIRECT}" != "1" ]]; then
            echo "check log: ${LAUNCHNET_LOGS_DIR}/bootstrap.log"
        fi
        exit ${GENESIS_EXIT_CODE}
    fi
    echo "bootstrap is done"
}

watch_pulse=true
run_pulsar=true
cloud_mode=false
check_working_dir
process_input_params "$@"

kill_all
trap 'stop_all' INT TERM EXIT

if [[ "$run_pulsar" == "true" ]]
then
    echo "start pulsar ..."
    echo "   log: ${LAUNCHNET_LOGS_DIR}pulsar_output.log"
    set -x
    ${PULSARD} --config ${PULSAR_CONFIG} &> ${LAUNCHNET_LOGS_DIR}pulsar_output.log &
    { set +x; } 2>/dev/null
    echo "pulsar log: ${LAUNCHNET_LOGS_DIR}pulsar_output.log"
else
    echo "Skip launching of pulsar"
fi

# launch_keeperd

handle_sigchld()
{
  jobs -pn
  echo "someone left the network"
}

if [[ "$LOGROTATOR_ENABLE" == "1" ]]; then
  echo "prepare logger"
  build_logger
fi

trap 'handle_sigchld' SIGCHLD

echo "start discovery nodes ..."
if [ "$cloud_mode" == true ]
then
  echo "run in cloud mode ( one process )"
  go run -mod=vendor scripts/cloud/generate_cloud_config.go
  set -x
  $INSOLARD test cloud --config ${CONFIGS_DIR}/base_cloud.yaml \
            2>&1 | ${LOGROTATOR} ${DISCOVERY_NODE_LOGS}/cloud.log > /dev/null &
        { set +x; } 2>/dev/null
else
  for i in `seq 1 $NUM_DISCOVERY_NODES`
  do
    if [ "$headless" == "true" ]
    then
       echo "start node $i in headless mode"
       set -x
       $INSOLARD test headless \
            --config ${DISCOVERY_NODES_DATA}${i}/insolard.yaml \
            2>&1 | ${LOGROTATOR} ${DISCOVERY_NODE_LOGS}${i}/output.log > /dev/null &
       { set +x; } 2>/dev/null
    else
        ROLE="virtual"
        set -x
        $INSOLARD test node \
            --role=${ROLE} \
            --config ${DISCOVERY_NODES_DATA}${i}/insolard.yaml \
            2>&1 | ${LOGROTATOR} ${DISCOVERY_NODE_LOGS}${i}/output.log > /dev/null &
        { set +x; } 2>/dev/null
    fi

    echo "discovery node $i started in background"
    echo "log: ${DISCOVERY_NODE_LOGS}${i}/output.log"
  done
fi

echo "discovery nodes started ..."

if [[ "$NUM_NODES" -ne "0" ]]
then
    wait_for_complete_network_state
    if [[ "$GENESIS" == "1" ]]; then
        ./scripts/insolard/start_nodes.sh -g
    else
        ./scripts/insolard/start_nodes.sh
    fi
fi

if [[ "$watch_pulse" == "true" ]]
then
    echo "starting pulse watcher..."
    echo "${PULSEWATCHER} --config ${PULSEWATCHER_CONFIG}"
    ${PULSEWATCHER} --config ${PULSEWATCHER_CONFIG}
else
    echo "waiting..."
    wait
fi

echo "FINISHING ..."
