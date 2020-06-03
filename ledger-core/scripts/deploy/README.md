# Kubernetes deployment manifests

This is insolar deployment in k8s cluster
If you know what you are doing, you can just exec `kubectl apply -k deploy/kube/local/`
Default behaviour is to run 5-node insolar net, to change node number go to /scripts/deploy/kube/manifests/nodes-patch.yaml.
Everything will be executed in **your kubernetes context**
There are make targets below for most useful cases, that all uses `kubectl` under hood.

## Run functional tests
Run once "apply ingress", it will be listening 443/80 ports on your host
```
make kube_apply_ingress
```
then run tests:
```
make test_func_kubernetes
```
Network will still be active after tests, you can run tests many times.
You don't need to rebuild images if you change only tests code

## Run network manually
```
make kube_start_net
```

## Stop network manually
```
make kube_stop_net
```

## Collect logs
```
make kube_collect_artifacts
```
Node logs will be saved in /tmp/insolar/logs

## Drop ingress
```
make kube_drop_ingress
```

## Rebuild images
If you want to change application code, you need to rebuild images and restart network
```
make docker-build
```

## How to run tests from IDE
Add in "run parameters" environment variables
```
INSOLAR_FUNC_RPC_URL_PUBLIC=http://localhost/api/rpc;INSOLAR_FUNC_RPC_URL=http://localhost/admin-api/rpc;INSOLAR_FUNC_KEYS_PATH=/tmp/insolar/;INSOLAR_FUNC_TESTWALLET_HOST=localhost
```
If you want to change application code

## Prometheus

Prometheus can by deployed/dropped in your local cluster by 
```
make kube_apply_prometheus
make kube_drop_prometheus
```

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
curl -s http://localhost/prometheus/api/v1/status/tsdb | jq .
```

To get particular metric:
```
curl http://localhost/prometheus/api/v1/query?query=insolar_consensus_packets_recv | jq .
```

##About CI
On github actions we use k3s cluster (https://k3s.io/) with a local registry.
As a further improvement, we can run tests inside the cluster on one image.

##Deployment tool
Provides management of insolar network in k8s for consensus test.
Default behaviour is to start/stop insolar net according to config, with prometheus optionally.
Simple usage and config template is in ./scripts/deploy/kube-deploy-tool path.
```
go build -o ./bin/kube-deploy-tool ./scripts/deploy/kube-deploy-tool && ./bin/kube-deploy-tool --config=scripts/deploy/kube-deploy-tool/config.yaml
```

Deploy tool modifies bootstrap files in ledger-core/scripts/deploy/kube/manifests/configuration. 
This files has default config for 5-node network for CI and local runs, so if you use the tool locally be careful, don't commit changes in this dir, if you don't know why you doing this.

## Debug
List insolar pods
```
watch -n 1 -d kubectl -n insolar get po
```
Look pulsewatcher
```
kubectl -n insolar logs -f --tail=20 services/pulsewatcher
```
Describe pods
```
kubectl -n insolar describe pods
```
