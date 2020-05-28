# Assured Ledger

[![linux-checks](https://github.com/insolar/assured-ledger/workflows/linux-checks/badge.svg)](https://github.com/insolar/assured-ledger/actions?query=workflow%3Alinux-checks+branch%3Amaster)
[![windows-checks](https://github.com/insolar/assured-ledger/workflows/windows-checks/badge.svg)](https://github.com/insolar/assured-ledger/actions?query=workflow%3Awindows-checks+branch%3Amaster)
[![codecov](https://codecov.io/gh/insolar/assured-ledger/branch/master/graph/badge.svg)](https://codecov.io/gh/insolar/assured-ledger)

Assured Ledger is a byzantine fault tolerant distributed application server.

This project is currently under development. It's the next generation of [Insolar Platform](https://github.com/insolar/insolar).

## Quick start

To build the project run:

```
cd ledger-core/v2
make vendor pre-build build
````


## Prometheus

Prometheus can be deployed via `make kube_apply_prometheus` command.

Prometheus is scraping all targets that have prometheus annotations:
```
      annotations:
        prometheus.io/port: "8001"
        prometheus.io/scrape: "true"
```

You can specify the port which your app is using. Any new deployed software with correct annotations will be scraped by prometheus.

After prometheus and ingress controller are up and running, you can check what's going on:

To get a list of metrics names and their series count:
```
curl -s 'http://localhost/prometheus/api/v1/status/tsdb' | jq .
```

To get particular metric:
```
curl 'http://localhost/prometheus/api/v1/query?query=insolar_consensus_packets_recv' | jq .
```
