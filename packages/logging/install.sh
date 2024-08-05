#! /bin/bash
set -euao pipefail

base=$(dirname "$0")

echo "##########################################################################"
echo "### Install Logging ###"

# Install elasticsearch
if [ "$IDO_TLS_KEY" == "tls" ]; then
  export IDO_TLS_ENABLED=true
else
  export IDO_TLS_ENABLED=false
fi

envsubst < "${base}/values-elasticsearch-override.yaml" > "${base}/values-elasticsearch.yaml"
"${base}/../check-undefined-env.sh" "${base}/values-elasticsearch.yaml"
helm upgrade elasticsearch --install --create-namespace --namespace logging --wait --timeout 30m -f "${base}"/values-elasticsearch.yaml "${base}"/elasticsearch

# Install fluent-bit
envsubst < "${base}/values-fluent-bit-override.yaml" > "${base}/values-fluent-bit.yaml"
"${base}/../check-undefined-env.sh" "${base}/values-fluent-bit.yaml"
helm upgrade fluent-bit --install --create-namespace --namespace logging --timeout 30m -f "${base}"/values-fluent-bit.yaml "${base}"/fluent-bit