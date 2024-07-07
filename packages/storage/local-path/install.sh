#! /bin/bash
set -euao pipefail

base=$(dirname "$0")

echo "##########################################################################"
echo "### Install Local-Path Provisioner ###"

# Install
envsubst '${IDO_DOCKER_CONTAINER_MIRROR}' < "${base}/local-path-storage-template.yaml" > "${base}/local-path-storage.yaml"
"${base}/../../check-undefined-env.sh" "${base}/local-path-storage.yaml"
kubectl apply -f "${base}"/local-path-storage.yaml
