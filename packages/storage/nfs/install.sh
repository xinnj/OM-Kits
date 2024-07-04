#! /bin/bash
set -euaxo pipefail

base=$(dirname "$0")

echo "##########################################################################"
echo "### Install NFS Provisioner ###"

# Install nfs provisioner
envsubst < "${base}/values-override.yaml" > "${base}/values.yaml"
"${base}/../../check-undefined-env.sh" "${base}/values.yaml"
helm upgrade nfs-subdir-external-provisioner --install --create-namespace --namespace nfs-provisioner -f "${base}"/values.yaml "${base}"/nfs-subdir-external-provisioner-chart
