#!/usr/bin/env bash
# Runs Terraform inside a throwaway Podman container so the host needs no
# Terraform installation. The container talks to the host rootless Podman
# socket and operates in /workspace (this repository).
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TF_IMAGE="${TF_IMAGE:-docker.io/hashicorp/terraform:1.9}"
RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"
PODMAN_SOCK="${RUNTIME_DIR}/podman/podman.sock"
DATA_HOME="${XDG_DATA_HOME:-$HOME/.local/share}/family-finance-tf"
TF_DATA_DIR="${DATA_HOME}/data"
PLUGIN_CACHE="${DATA_HOME}/plugin-cache"

if [[ ! -S "${PODMAN_SOCK}" ]]; then
  echo "Podman API socket not found at ${PODMAN_SOCK}" >&2
  echo "Enable it with: systemctl --user enable --now podman.socket" >&2
  exit 1
fi

mkdir -p "${TF_DATA_DIR}" "${PLUGIN_CACHE}"

tty_args=()
if [[ -t 0 ]]; then
  tty_args=(-t)
fi

exec podman run --rm -i "${tty_args[@]}" \
  --security-opt label=disable \
  --env TF_IN_AUTOMATION=1 \
  --env TF_DATA_DIR=/tf-data/data \
  --env TF_PLUGIN_CACHE_DIR=/tf-data/plugin-cache \
  --env DOCKER_HOST=unix:///tmp/podman.sock \
  --volume "${REPO_DIR}:/workspace" \
  --volume "${DATA_HOME}:/tf-data" \
  --volume "${PODMAN_SOCK}:/tmp/podman.sock" \
  --workdir /workspace/terraform \
  "${TF_IMAGE}" "$@"
