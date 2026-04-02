#!/usr/bin/env bash
#
# build.sh - Linux/macOS Build Script
#
# This script automates the process of building and running the Docker container
# with version information dynamically injected at build time.

# Hidden feature: Preserve usage statistics across rebuilds
# Usage: ./docker-build.sh --with-usage
# First run prompts for management API key, saved to temp/stats/.api_secret

set -euo pipefail

STATS_DIR="temp/stats"
STATS_FILE="${STATS_DIR}/.usage_backup.json"
SECRET_FILE="${STATS_DIR}/.api_secret"
COMPOSE_FILE="docker-compose.yml"
CONFIG_FILE="config.yaml"
DEFAULT_PORT="8317"
SERVICE_NAME="cli-proxy-api"
WITH_USAGE=false
ENABLE_CODEX_BRIDGE=false

config_exists() {
  [[ -f "${CONFIG_FILE}" ]]
}

ensure_config_exists() {
  if config_exists; then
    return
  fi

  echo "Error: ${CONFIG_FILE} is required by ${COMPOSE_FILE} but was not found."
  if [[ -f "config.example.yaml" ]]; then
    echo "Create it first, for example:"
    echo "  cp config.example.yaml ${CONFIG_FILE}"
  fi
  exit 1
}

get_port() {
  local configured_port

  if config_exists; then
    configured_port=$(grep -E "^port:" "${CONFIG_FILE}" | sed -E 's/^port: *["'"'"']?([0-9]+)["'"'"']?.*$/\1/' | head -n1 || true)
    if [[ -n "${configured_port}" ]]; then
      echo "${configured_port}"
      return
    fi
  fi

  echo "${DEFAULT_PORT}"
}

compose_has_service_image() {
  [[ -f "${COMPOSE_FILE}" ]] || return 1

  awk -v service="${SERVICE_NAME}" '
    $0 ~ "^  " service ":" {
      in_service = 1
      next
    }
    in_service && $0 ~ "^  [A-Za-z0-9_-]+:" {
      in_service = 0
    }
    in_service && $0 ~ "^    image:" {
      found = 1
      exit
    }
    END {
      exit(found ? 0 : 1)
    }
  ' "${COMPOSE_FILE}"
}

get_local_port() {
  local container_port published_port
  container_port=$(get_port)

  if [[ -f "${COMPOSE_FILE}" ]]; then
    published_port=$(
      awk -v service="${SERVICE_NAME}" -v target="${container_port}" '
        $0 ~ "^  " service ":" {
          in_service = 1
          next
        }
        in_service && $0 ~ "^  [A-Za-z0-9_-]+:" {
          in_service = 0
          in_ports = 0
        }
        in_service && $0 ~ "^    ports:" {
          in_ports = 1
          next
        }
        in_service && in_ports && $0 ~ "^    [A-Za-z0-9_-]+:" {
          in_ports = 0
        }
        in_service && in_ports {
          line = $0
          gsub(/"/, "", line)
          sub(/^[[:space:]]*-[[:space:]]*/, "", line)
          split(line, ports, ":")
          if (length(ports) >= 2 && ports[2] == target) {
            print ports[1]
            exit
          }
        }
      ' "${COMPOSE_FILE}"
    )
    if [[ -n "${published_port}" ]]; then
      echo "${published_port}"
      return
    fi
  fi

  echo "${container_port}"
}

has_existing_local_image() {
  docker compose images -q "${SERVICE_NAME}" 2>/dev/null | grep -q '[^[:space:]]'
}

prompt_codex_bridge() {
  local answer
  if ! read -r -p "Enable chat2api codex-bridge profile? [y/N]: " answer; then
    answer=""
  fi
  case "${answer}" in
    [Yy]|[Yy][Ee][Ss])
      ENABLE_CODEX_BRIDGE=true
      ;;
    *)
      ENABLE_CODEX_BRIDGE=false
      ;;
  esac
}

compose_up() {
  if [[ "${ENABLE_CODEX_BRIDGE}" == "true" ]]; then
    docker compose --profile codex-bridge up "$@"
  else
    docker compose up "$@"
  fi
}

export_stats_api_secret() {
  if [[ -f "${SECRET_FILE}" ]]; then
    API_SECRET=$(cat "${SECRET_FILE}")
  else
    if [[ ! -d "${STATS_DIR}" ]]; then
      mkdir -p "${STATS_DIR}"
    fi
    echo "First time using --with-usage. Management API key required."
    read -r -p "Enter management key: " -s API_SECRET
    echo
    echo "${API_SECRET}" > "${SECRET_FILE}"
    chmod 600 "${SECRET_FILE}"
  fi
}

check_container_running() {
  local port
  port=$(get_local_port)

  if ! curl -s -o /dev/null -w "%{http_code}" "http://localhost:${port}/" | grep -q "200"; then
    echo "Error: cli-proxy-api service is not responding at localhost:${port}"
    echo "Please start the container first or use without --with-usage flag."
    exit 1
  fi
}

export_stats() {
  local port
  port=$(get_local_port)

  if [[ ! -d "${STATS_DIR}" ]]; then
    mkdir -p "${STATS_DIR}"
  fi
  check_container_running
  echo "Exporting usage statistics..."
  EXPORT_RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Management-Key: ${API_SECRET}" \
    "http://localhost:${port}/v0/management/usage/export")
  HTTP_CODE=$(echo "${EXPORT_RESPONSE}" | tail -n1)
  RESPONSE_BODY=$(echo "${EXPORT_RESPONSE}" | sed '$d')

  if [[ "${HTTP_CODE}" != "200" ]]; then
    echo "Export failed (HTTP ${HTTP_CODE}): ${RESPONSE_BODY}"
    exit 1
  fi

  echo "${RESPONSE_BODY}" > "${STATS_FILE}"
  echo "Statistics exported to ${STATS_FILE}"
}

import_stats() {
  local port
  port=$(get_local_port)

  echo "Importing usage statistics..."
  IMPORT_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
    -H "X-Management-Key: ${API_SECRET}" \
    -H "Content-Type: application/json" \
    -d @"${STATS_FILE}" \
    "http://localhost:${port}/v0/management/usage/import")
  IMPORT_CODE=$(echo "${IMPORT_RESPONSE}" | tail -n1)
  IMPORT_BODY=$(echo "${IMPORT_RESPONSE}" | sed '$d')

  if [[ "${IMPORT_CODE}" == "200" ]]; then
    echo "Statistics imported successfully"
  else
    echo "Import failed (HTTP ${IMPORT_CODE}): ${IMPORT_BODY}"
  fi

  rm -f "${STATS_FILE}"
}

wait_for_service() {
  local port
  port=$(get_local_port)

  echo "Waiting for service to be ready..."
  for i in {1..30}; do
    if curl -s -o /dev/null -w "%{http_code}" "http://localhost:${port}/" | grep -q "200"; then
      break
    fi
    sleep 1
  done
  sleep 2
}

print_menu() {
  echo "Please select an option:"
  if compose_has_service_image; then
    echo "1) Run using Pre-built Image (Recommended)"
  else
    echo "1) Run using Existing Local Image (No Rebuild)"
  fi
  echo "2) Build from Source and Run (For Developers)"
}

run_no_rebuild_mode() {
  ensure_config_exists

  if compose_has_service_image; then
    echo "--- Running with Pre-built Image ---"
  else
    echo "--- Running with Existing Local Image (No Rebuild) ---"
    if ! has_existing_local_image; then
      echo "Current ${COMPOSE_FILE} does not define a pre-built image for ${SERVICE_NAME}."
      echo "No existing local image was found."
      echo "Use option 2 to build locally first."
      exit 1
    fi
  fi

  prompt_codex_bridge

  if [[ "${WITH_USAGE}" == "true" ]]; then
    export_stats
  fi

  compose_up -d --remove-orphans --no-build

  if [[ "${WITH_USAGE}" == "true" ]]; then
    wait_for_service
    import_stats
  fi

  echo "Services are starting."
  echo "Run 'docker compose logs -f' to see the logs."
}

run_build_mode() {
  local version commit build_date

  ensure_config_exists

  echo "--- Building from Source and Running ---"

  version="$(git describe --tags --always --dirty)"
  commit="$(git rev-parse --short HEAD)"
  build_date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

  echo "Building with the following info:"
  echo "  Version: ${version}"
  echo "  Commit: ${commit}"
  echo "  Build Date: ${build_date}"
  echo "----------------------------------------"

  echo "Building the Docker image..."
  docker compose build \
    --build-arg VERSION="${version}" \
    --build-arg COMMIT="${commit}" \
    --build-arg BUILD_DATE="${build_date}"

  prompt_codex_bridge

  if [[ "${WITH_USAGE}" == "true" ]]; then
    export_stats
  fi

  echo "Starting the services..."
  compose_up -d --remove-orphans --pull never

  if [[ "${WITH_USAGE}" == "true" ]]; then
    wait_for_service
    import_stats
  fi

  echo "Build complete. Services are starting."
  echo "Run 'docker compose logs -f' to see the logs."
}

main() {
  local choice

  if [[ "${1:-}" == "--with-usage" ]]; then
    WITH_USAGE=true
    export_stats_api_secret
  fi

  print_menu
  read -r -p "Enter choice [1-2]: " choice

  case "${choice}" in
    1)
      run_no_rebuild_mode
      ;;
    2)
      run_build_mode
      ;;
    *)
      echo "Invalid choice. Please enter 1 or 2."
      exit 1
      ;;
  esac
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "$@"
fi
