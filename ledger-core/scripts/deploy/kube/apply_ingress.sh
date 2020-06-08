#!/bin/bash -e
kubectl apply -k scripts/deploy/kube/traefik
kubectl -n kube-system rollout status deploy/traefik-ingress-controller --timeout=80s
