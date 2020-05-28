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

##About CI
On github actions we use k3s cluster (https://k3s.io/) with a local registry.
As a further improvement, we can run tests inside the cluster on one image.

##Deployment tool
Firstly it provides management 