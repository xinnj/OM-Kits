#!/usr/bin/env bash
# Regression test for the Helm 4 server-side apply flags in install.sh.
#
# Helm 4 rejects --force-conflicts unless server-side apply is explicitly
# enabled. An existing release created client-side (for example by Helm 3)
# makes the default --server-side=auto resolve to false, so the installer
# must pass --server-side=true together with --force-conflicts.
set -euo pipefail

repo=$(cd "$(dirname "$0")" && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

mkdir -p "$work/bin" "$work/packages/prometheus/kube-prometheus-stack"
cp "$repo/install.sh" "$work/packages/prometheus/install.sh"
cp "$repo/../check-undefined-env.sh" "$work/packages/check-undefined-env.sh"
chmod +x "$work/packages/check-undefined-env.sh"
printf 'test: ${IDO_TEST_VALUE}\n' > "$work/packages/prometheus/values-override.yaml"

cat > "$work/bin/kubectl" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF

cat > "$work/bin/helm" <<'EOF'
#!/usr/bin/env bash
if [ "${1:-}" = "upgrade" ] && [[ " $* " == *" --help "* ]]; then
  cat "$FAKE_HELM_HELP"
  exit 0
fi
if [ "${1:-}" = "upgrade" ]; then
  printf '%s\n' "$*" >> "$FAKE_HELM_LOG"
  exit 0
fi
exit 1
EOF
chmod +x "$work/bin/kubectl" "$work/bin/helm"

# Help text shapes: Helm 4 exposes both flags; Helm 3 exposes neither.
cat > "$work/help-helm4.txt" <<'EOF'
      --force-conflicts    if set server-side apply will force changes against conflicts
      --server-side string must be "true", "false" or "auto" (default "auto")
EOF
cat > "$work/help-helm3.txt" <<'EOF'
      --atomic             if set, upgrade process rolls back changes made in case of failed upgrade
EOF

failures=0

run_install() {
  local help_file=$1
  local log_file=$2
  : > "$log_file"
  PATH="$work/bin:$PATH" \
    FAKE_HELM_HELP="$help_file" \
    FAKE_HELM_LOG="$log_file" \
    IDO_DINGTALK_ENABLED=false \
    IDO_FEISHU_ENABLED=false \
    IDO_WECHAT_ENABLED=false \
    IDO_ALERTING_LANG=en \
    IDO_GHCR_CONTAINER_MIRROR=ghcr.io \
    IDO_TEST_VALUE=value \
    bash "$work/packages/prometheus/install.sh" >/dev/null 2>&1
}

# Helm 4: --force-conflicts must be paired with explicit server-side apply.
if run_install "$work/help-helm4.txt" "$work/helm4.log"; then
  line=$(cat "$work/helm4.log")
  if [[ " $line " != *" --server-side=true "* || " $line " != *" --force-conflicts "* ]]; then
    echo "FAIL: Helm 4 invocation missing SSA flags: $line"
    failures=$((failures + 1))
  fi
else
  echo "FAIL: install.sh failed under the Helm 4 stub"
  failures=$((failures + 1))
fi

# Helm 3: neither flag exists, so neither may be passed.
if run_install "$work/help-helm3.txt" "$work/helm3.log"; then
  line=$(cat "$work/helm3.log")
  if [[ " $line " == *" --server-side"* || " $line " == *" --force-conflicts"* ]]; then
    echo "FAIL: Helm 3 invocation received unsupported flags: $line"
    failures=$((failures + 1))
  fi
else
  echo "FAIL: install.sh failed under the Helm 3 stub"
  failures=$((failures + 1))
fi

if [ "$failures" -ne 0 ]; then
  exit 1
fi
echo "PASS: Helm 3/4 server-side apply flags"
