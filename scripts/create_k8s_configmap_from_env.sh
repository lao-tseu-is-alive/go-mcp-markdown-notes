#!/bin/bash
# https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/
kubectl create configmap app-configs --from-env-file=.env --output yaml --dry-run=client