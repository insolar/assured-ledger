#!/usr/bin/env bash
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
KUBECTL=${KUBECTL:-"kubectl"}

# todo remove ingress
install_ingress() {
  if [ "$($KUBECTL get pods -n ingress-nginx -l app.kubernetes.io/name=ingress-nginx | grep -c Running)" = "0" ]; then
    $KUBECTL apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-0.31.0/deploy/static/provider/cloud/deploy.yaml
  fi

  ready=0
  for try in {0..60}; do
    if [ "$($KUBECTL get pods -n ingress-nginx | grep controller | grep -v svclb | grep -c Running)" = "1" ]; then
      ready=1
      break
    fi
    sleep 2s
  done

  if [ $ready = 0 ]; then
    echo "ingress was not started"
    $KUBECTL -n ingress-nginx describe pods
    exit 1
  fi
  # dirty hack
  $KUBECTL delete validatingwebhookconfiguration ingress-nginx-admission
}

install_ingress
echo "ingress installed"