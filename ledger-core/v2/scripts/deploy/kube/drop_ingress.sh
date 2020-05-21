#!/usr/bin/env bash
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
KUBECTL=${KUBECTL:-"kubectl"}

delete_ingress() {
    $KUBECTL delete -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-0.31.0/deploy/static/provider/cloud/deploy.yaml
}

delete_ingress
