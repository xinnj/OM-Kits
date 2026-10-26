#! /bin/bash
set -euao pipefail

base=$(dirname "$0")

# The installer always sets these. Require them for direct runs so an
# accidental run without the flags does not remove the adapter and its Secret.
: "${IDO_DINGTALK_ENABLED:?IDO_DINGTALK_ENABLED must be set to true or false}"
: "${IDO_FEISHU_ENABLED:?IDO_FEISHU_ENABLED must be set to true or false}"
: "${IDO_WECHAT_ENABLED:?IDO_WECHAT_ENABLED must be set to true or false}"
: "${IDO_ALERTING_LANG:?IDO_ALERTING_LANG must be set to en or zh}"

echo "##########################################################################"
echo "### Install Prometheus Stack ###"

# Create namespaces
kubectl create ns monitoring --dry-run=client -o yaml | kubectl apply -f -

adapterRelease="alertmanager-webhook-adapter"
adapterSecret="alertmanager-webhook-adapter"

dingtalk=false
feishu=false
wechat=false
if [ "$IDO_DINGTALK_ENABLED" = "true" ]; then
  dingtalk=true
fi
if [ "$IDO_FEISHU_ENABLED" = "true" ]; then
  feishu=true
fi
if [ "$IDO_WECHAT_ENABLED" = "true" ]; then
  wechat=true
fi

# Install the webhook adapter before Prometheus so the AlertmanagerConfig CR
# points at a reachable Service as soon as it is reconciled.
if [ "$dingtalk" = true ] || [ "$feishu" = true ] || [ "$wechat" = true ]; then
  secret_args=()
  if [ "$dingtalk" = true ]; then
    : "${IDO_DINGTALK_TOKEN:?DingTalk token is required}"
    : "${IDO_DINGTALK_SECRET:?DingTalk secret is required}"
    secret_args+=(--from-literal="DINGTALK_TOKEN=${IDO_DINGTALK_TOKEN}")
    secret_args+=(--from-literal="DINGTALK_SECRET=${IDO_DINGTALK_SECRET}")
  fi
  if [ "$feishu" = true ]; then
    : "${IDO_FEISHU_TOKEN:?Feishu token is required}"
    : "${IDO_FEISHU_SECRET:?Feishu secret is required}"
    secret_args+=(--from-literal="FEISHU_TOKEN=${IDO_FEISHU_TOKEN}")
    secret_args+=(--from-literal="FEISHU_SECRET=${IDO_FEISHU_SECRET}")
  fi
  if [ "$wechat" = true ]; then
    : "${IDO_WECHAT_TOKEN:?WeChat bot key is required}"
    # The adapter reads the WeCom group-robot key from WEIXIN_TOKEN.
    secret_args+=(--from-literal="WEIXIN_TOKEN=${IDO_WECHAT_TOKEN}")
  fi

  # --from-literal preserves values containing '='; server-side apply prunes
  # keys for channels that have been disabled.
  kubectl create secret generic "$adapterSecret" -n monitoring "${secret_args[@]}" \
    --dry-run=client -o yaml \
    | kubectl apply --server-side --force-conflicts --field-manager=ido-installer-adapter -n monitoring -f -

  envsubst < "${base}/values-override-adapter.yaml" > "${base}/values-adapter.yaml"
  "${base}/../check-undefined-env.sh" "${base}/values-adapter.yaml"
  helm upgrade "$adapterRelease" --install --namespace monitoring --timeout 15m \
    -f "${base}/values-adapter.yaml" "${base}/alertmanager-webhook-adapter"
  # envFrom is read at container start; restart so credential changes take effect.
  kubectl -n monitoring rollout restart "deployment/${adapterRelease}"
  kubectl -n monitoring rollout status "deployment/${adapterRelease}" --timeout=5m
fi

# Install prometheus
envsubst < "${base}/values-override.yaml" > "${base}/values.yaml"
"${base}/../check-undefined-env.sh" "${base}/values.yaml"
helm upgrade prometheus --install --create-namespace --namespace monitoring --timeout 30m -f "${base}"/values.yaml "${base}"/kube-prometheus-stack

# Remove the adapter when every channel is disabled.
if [ "$dingtalk" = false ] && [ "$feishu" = false ] && [ "$wechat" = false ]; then
  if helm -n monitoring status "$adapterRelease" >/dev/null 2>&1; then
    helm -n monitoring uninstall "$adapterRelease" \
      || echo "Warning: could not uninstall ${adapterRelease}"
  fi
  kubectl -n monitoring delete secret "$adapterSecret" --ignore-not-found
fi

# Remove the legacy DingTalk bridge (best effort; the old release and ConfigMap
# may still exist on clusters installed before this change).
if helm -n monitoring status prometheus-webhook-dingtalk >/dev/null 2>&1; then
  helm -n monitoring uninstall prometheus-webhook-dingtalk \
    || echo "Warning: could not uninstall legacy DingTalk bridge"
fi
kubectl -n monitoring delete configmap prometheus-webhook-dingtalk --ignore-not-found \
  || echo "Warning: could not delete legacy DingTalk ConfigMap"
