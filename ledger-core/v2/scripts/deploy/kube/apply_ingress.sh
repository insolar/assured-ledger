#!/usr/bin/env bash
kubectl apply -k traefik
kubectl -n kube-system rollout status deploy/traefik-ingress-controller --timeout=80s
echo "ingress installed"
