#!/bin/bash
kubectl apply -k prometheus
kubectl -n prometheus rollout status sts/prometheus --timeout=80s
