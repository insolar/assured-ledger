#### Load tests

##### Local run
Start network
```
scripts/insolard/launchnet.sh -g
```

Run all handles sequential test
```
go run load/cmd/load/main.go -config load/run_configs/all_sequence.yaml -gen_config load/gen_cfg/generator_local.yaml
```
see results.csv/runners.log after test