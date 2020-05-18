# Kuberenetes deploy manifests

## Run functional tests
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
Node Logs will be saved in /tmp/insolar/logs

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
On github actions we use k3s cluster with a local registry.
As a further improvement, we can run tests inside the cluster on one image.