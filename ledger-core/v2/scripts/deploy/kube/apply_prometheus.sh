#!/bin/bash -e
kubectl apply -k scripts/deploy/kube/prometheus
kubectl -n prometheus rollout status sts/prometheus --timeout=80s
