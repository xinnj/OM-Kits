#!/bin/bash

# Remote build for OM-Kits.
# Syncs the project to a remote Linux machine and builds the installer
# binary there. The result is ${REMOTE_PATH}/om-kits-installer, ready to
# run next to packages/. Nothing is copied back.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

echo_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

echo_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

ENV_FILE="${SCRIPT_DIR}/.env.local"
if [[ ! -f "${ENV_FILE}" ]]; then
    echo_error "${ENV_FILE} not found"
    echo_error "Copy .env.example to .env.local and fill in the values."
    exit 1
fi

# shellcheck source=/dev/null
source "${ENV_FILE}"

for var in REMOTE_USER REMOTE_HOST REMOTE_PORT REMOTE_PATH; do
    if [[ -z "${!var:-}" ]]; then
        echo_error "${var} is not set in ${ENV_FILE}"
        exit 1
    fi
done

# REMOTE_PATH is interpolated unquoted into remote commands so a leading
# "~/" is expanded by the remote shell.
if [[ "${REMOTE_PATH}" == *" "* ]]; then
    echo_error "REMOTE_PATH must not contain spaces"
    exit 1
fi

SSH_OPTS=(-p "${REMOTE_PORT}")
REMOTE="${REMOTE_USER}@${REMOTE_HOST}"

# Step 1: Rsync project to remote build machine
step1_rsync() {
    echo_info "Step 1: Syncing project to ${REMOTE_HOST}..."

    ssh "${SSH_OPTS[@]}" "${REMOTE}" "mkdir -p ${REMOTE_PATH}"

    rsync -avz --delete \
        --exclude='.git' \
        --exclude='.idea' \
        --exclude='.DS_Store' \
        --exclude='.env.local' \
        -e "ssh -p ${REMOTE_PORT}" \
        "${SCRIPT_DIR}/" \
        "${REMOTE}:${REMOTE_PATH}/"

    echo_info "Project synced."
}

# Step 2: Build the linux/amd64 binary on the remote machine
step2_build() {
    echo_info "Step 2: Building linux/amd64 binary on ${REMOTE_HOST}..."

    ssh "${SSH_OPTS[@]}" "${REMOTE}" \
        "cd ${REMOTE_PATH}/installer && source ~/.profile && go version && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../om-kits-installer ."

    echo_info "Build completed."
}

# Main execution
main() {
    step1_rsync
    step2_build

    echo_info "Binary ready: ${REMOTE}:${REMOTE_PATH}/om-kits-installer"
}

main "$@"
