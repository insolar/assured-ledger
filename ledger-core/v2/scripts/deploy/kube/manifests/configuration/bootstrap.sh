#!/bin/bash -e
# BE CAREFUL, THIS FILE DIFFERS FROM PL 1.X
apt-get update -qq
apt-get install curl -y -qq
curl -L https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl -o /usr/bin/kubectl
chmod +x /usr/bin/kubectl
if (kubectl get secret node-keys -o yaml | grep -q 'pulsar-0.json'); then echo "Bootstrap data already present, exiting"; exit; fi
kubectl -n insolar create secret generic node-certs
kubectl -n insolar create secret generic node-keys
INSOLAR_BIN=${INSOLAR_BIN:-"insolar"}
BOOTSTRAP_CONFIG=${BOOTSTRAP_CONFIG:-"/etc/bootstrap/bootstrap.yaml"}
CONFIGS_DIR=${CONFIGS_DIR:-"/var/data/bootstrap/configs/"}
PULSAR_KEYS=/var/data/bootstrap/discovery-keys/pulsar-0.json
generate_pulsar_keys() {
    echo "generate pulsar keys: ${PULSAR_KEYS}"
    ${INSOLAR_BIN} gen-key-pair > ${PULSAR_KEYS}
}
generate_root_member_keys() {
    echo "generate members keys in dir: $CONFIGS_DIR"
<<<<<<< HEAD
    ${INSOLAR_BIN} gen-key-pair > ${CONFIGS_DIR}root_member_keys.json
    ${INSOLAR_BIN} gen-key-pair > ${CONFIGS_DIR}fee_member_keys.json
    ${INSOLAR_BIN} gen-key-pair > ${CONFIGS_DIR}migration_admin_member_keys.json
    for b in {0..9}; do
        ${INSOLAR_BIN} gen-key-pair > ${CONFIGS_DIR}migration_daemon_${b}_member_keys.json
    done
    for b in {0..139}; do
        ${INSOLAR_BIN} gen-key-pair > ${CONFIGS_DIR}network_incentives_${b}_member_keys.json
    done
    for b in {0..39}; do
        ${INSOLAR_BIN} gen-key-pair > ${CONFIGS_DIR}application_incentives_${b}_member_keys.json
    done
    for b in {0..39}; do
        ${INSOLAR_BIN} gen-key-pair > ${CONFIGS_DIR}foundation_${b}_member_keys.json
    done
    ${INSOLAR_BIN} gen-key-pair > ${CONFIGS_DIR}funds_0_member_keys.json
    for b in {0..7}; do
        ${INSOLAR_BIN} gen-key-pair > ${CONFIGS_DIR}enterprise_${b}_member_keys.json
=======
    ${INSOLAR_BIN} gen-key-pair > "${CONFIGS_DIR}"root_member_keys.json
    ${INSOLAR_BIN} gen-key-pair > "${CONFIGS_DIR}"fee_member_keys.json
    ${INSOLAR_BIN} gen-key-pair > "${CONFIGS_DIR}"migration_admin_member_keys.json
    for b in {0..9}; do
        ${INSOLAR_BIN} gen-key-pair > "${CONFIGS_DIR}"migration_daemon_${b}_member_keys.json
    done
    for b in {0..139}; do
        ${INSOLAR_BIN} gen-key-pair > "${CONFIGS_DIR}"network_incentives_${b}_member_keys.json
    done
    for b in {0..39}; do
        ${INSOLAR_BIN} gen-key-pair > "${CONFIGS_DIR}"application_incentives_${b}_member_keys.json
    done
    for b in {0..39}; do
        ${INSOLAR_BIN} gen-key-pair > "${CONFIGS_DIR}"foundation_${b}_member_keys.json
    done
    ${INSOLAR_BIN} gen-key-pair > "${CONFIGS_DIR}"funds_0_member_keys.json
    for b in {0..7}; do
        ${INSOLAR_BIN} gen-key-pair > "${CONFIGS_DIR}"enterprise_${b}_member_keys.json
>>>>>>> 80ed601376d4994db1df8d7ba49dbd725b61259a
    done
}
generate_migration_addresses() {
    echo "generate migration addresses: ${CONFIGS_DIR}migration_addresses.json"
<<<<<<< HEAD
    ${INSOLAR_BIN} gen-migration-addresses > ${CONFIGS_DIR}migration_addresses.json
=======
    ${INSOLAR_BIN} gen-migration-addresses > "${CONFIGS_DIR}"migration_addresses.json
>>>>>>> 80ed601376d4994db1df8d7ba49dbd725b61259a
}
bootstrap() {
    echo "bootstrap start"
    generate_pulsar_keys
    generate_root_member_keys
    generate_migration_addresses
}
<<<<<<< HEAD
mkdir -p ${CONFIGS_DIR} /var/data/bootstrap/certs
mkdir -p ${CONFIGS_DIR} /var/data/bootstrap/discovery-keys
bootstrap
$INSOLAR_BIN bootstrap -c $BOOTSTRAP_CONFIG --propernames=true
=======
mkdir -p "${CONFIGS_DIR}" /var/data/bootstrap/certs
mkdir -p "${CONFIGS_DIR}" /var/data/bootstrap/discovery-keys
bootstrap
${INSOLAR_BIN} bootstrap -c "$BOOTSTRAP_CONFIG" --propernames=true
>>>>>>> 80ed601376d4994db1df8d7ba49dbd725b61259a
MY_BIN_DIR=$( dirname "${BASH_SOURCE[0]}" )
cd /var/data/bootstrap
#todo fix cert path configuration in insolar
kubectl -n insolar create secret generic node-certs --from-file=/go/src/github.com/insolar/assured-ledger --dry-run=client -o yaml | kubectl -n insolar apply -f -
kubectl -n insolar create secret generic node-keys  --from-file=/var/data/bootstrap/discovery-keys --dry-run=client -o yaml | kubectl -n insolar apply -f -
