#!/bin/sh
while true; do
  if [ "$(kubectl -n insolar get po bootstrap -o jsonpath="{.status.phase}")" = "Succeeded" ]; then
    break
  fi
  sleep 1s
done
