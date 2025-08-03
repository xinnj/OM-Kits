#! /bin/bash
set -euao pipefail

base=$(dirname "$0")

echo "##########################################################################"
echo "### Install Permission Manager ###"

envsubst < "${base}/values-override.yaml" > "${base}/values.yaml"
"${base}/../check-undefined-env.sh" "${base}/values.yaml"
helm upgrade permission-manager --install --create-namespace --namespace permission-manager --timeout 30m -f "${base}"/values.yaml "${base}"/permission-manager-chart
