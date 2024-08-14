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
envsubst '${IDO_FLUENT_LOG_PATH}, ${IDO_FLUENT_ALERT_LOG_LEVEL}' < "${base}/values-fluent-bit-override.yaml" > "${base}/values-fluent-bit.yaml"
"${base}/../check-undefined-env.sh" "${base}/values-fluent-bit.yaml"
helm upgrade fluent-bit --install --create-namespace --namespace logging --timeout 30m -f "${base}"/values-fluent-bit.yaml "${base}"/fluent-bit

# Install fluent-bit-to-alertmanager
if [ "$IDO_FLUENT_ALERT_LOG_LEVEL" != "none" ]; then
  envsubst < "${base}/values-fluent-bit-to-alertmanager-override.yaml" > "${base}/values-fluent-bit-to-alertmanager.yaml"
  "${base}/../check-undefined-env.sh" "${base}/values-fluent-bit-to-alertmanager.yaml"
  helm upgrade fluent-bit-to-alertmanager --install --create-namespace --namespace logging --timeout 30m -f "${base}"/values-fluent-bit-to-alertmanager.yaml "${base}"/fluent-bit-to-alertmanager
fi