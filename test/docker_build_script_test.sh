#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  if [[ "${haystack}" != *"${needle}"* ]]; then
    fail "expected output to contain: ${needle}"
  fi
}

assert_not_contains() {
  local haystack="$1"
  local needle="$2"
  if [[ "${haystack}" == *"${needle}"* ]]; then
    fail "expected output not to contain: ${needle}"
  fi
}

new_fixture() {
  local fixture
  fixture="$(mktemp -d)"
  cp "${ROOT_DIR}/docker-build.sh" "${fixture}/docker-build.sh"
  cp "${ROOT_DIR}/docker-compose.yml" "${fixture}/docker-compose.yml"
  cat > "${fixture}/config.yaml" <<'EOF'
port: 8317
remote-management:
  secret-key: "test-secret"
EOF
  mkdir -p "${fixture}/stubbin" "${fixture}/temp/stats"
  echo "${fixture}"
}

test_existing_image_mode_requires_a_local_image_when_compose_has_no_service_image() {
  local fixture output status docker_log
  fixture="$(new_fixture)"
  docker_log="${fixture}/docker.log"

  cat > "${fixture}/stubbin/docker" <<EOF
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "\$*" >> "${docker_log}"
if [[ "\${1:-}" == "compose" && "\${2:-}" == "images" ]]; then
  exit 0
fi
echo "unexpected docker invocation: \$*" >&2
exit 99
EOF
  chmod +x "${fixture}/stubbin/docker"

  set +e
  output="$(
    cd "${fixture}" &&
      PATH="${fixture}/stubbin:${PATH}" bash ./docker-build.sh <<<"1" 2>&1
  )"
  status=$?
  set -e

  [[ ${status} -ne 0 ]] || fail "expected option 1 to fail without an existing local image"
  assert_contains "${output}" "Use option 2 to build locally first."
  assert_not_contains "$(cat "${docker_log}")" "compose up"
}

test_usage_preservation_targets_the_published_host_port() {
  local fixture output status docker_log curl_log
  fixture="$(new_fixture)"
  docker_log="${fixture}/docker.log"
  curl_log="${fixture}/curl.log"

  echo "test-secret" > "${fixture}/temp/stats/.api_secret"

  cat > "${fixture}/stubbin/docker" <<EOF
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "\$*" >> "${docker_log}"
case "\${1:-} \${2:-}" in
  "compose build"|"compose up")
    exit 0
    ;;
esac
echo "unexpected docker invocation: \$*" >&2
exit 99
EOF
  chmod +x "${fixture}/stubbin/docker"

  cat > "${fixture}/stubbin/curl" <<EOF
#!/usr/bin/env bash
set -euo pipefail
url="\${*: -1}"
printf '%s\n' "\${url}" >> "${curl_log}"
case "\${url}" in
  */v0/management/usage/export|*/v0/management/usage/import)
    printf '{"ok":true}\n200'
    ;;
  *)
    printf '200'
    ;;
esac
EOF
  chmod +x "${fixture}/stubbin/curl"

  cat > "${fixture}/stubbin/git" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
case "${1:-}" in
  describe)
    echo "v0.0.0-test"
    ;;
  rev-parse)
    echo "abc1234"
    ;;
  *)
    echo "unexpected git invocation: $*" >&2
    exit 99
    ;;
esac
EOF
  chmod +x "${fixture}/stubbin/git"

  cat > "${fixture}/stubbin/date" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "2026-04-02T00:00:00Z"
EOF
  chmod +x "${fixture}/stubbin/date"

  cat > "${fixture}/stubbin/sleep" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
  chmod +x "${fixture}/stubbin/sleep"

  set +e
  output="$(
    cd "${fixture}" &&
      PATH="${fixture}/stubbin:${PATH}" bash ./docker-build.sh --with-usage <<<"2" 2>&1
  )"
  status=$?
  set -e

  [[ ${status} -eq 0 ]] || fail "expected build mode with usage preservation to succeed, got: ${output}"
  assert_contains "$(cat "${curl_log}")" "http://localhost:8317/"
  assert_not_contains "$(cat "${curl_log}")" "http://localhost:8318/"
}

test_build_mode_can_enable_codex_bridge_profile() {
  local fixture output status docker_log
  fixture="$(new_fixture)"
  docker_log="${fixture}/docker.log"

  cat > "${fixture}/stubbin/docker" <<EOF
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "\$*" >> "${docker_log}"
case "\${1:-} \${2:-}" in
  "compose build"|"compose --profile")
    exit 0
    ;;
esac
echo "unexpected docker invocation: \$*" >&2
exit 99
EOF
  chmod +x "${fixture}/stubbin/docker"

  cat > "${fixture}/stubbin/git" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
case "${1:-}" in
  describe)
    echo "v0.0.0-test"
    ;;
  rev-parse)
    echo "abc1234"
    ;;
  *)
    echo "unexpected git invocation: $*" >&2
    exit 99
    ;;
esac
EOF
  chmod +x "${fixture}/stubbin/git"

  cat > "${fixture}/stubbin/date" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "2026-04-02T00:00:00Z"
EOF
  chmod +x "${fixture}/stubbin/date"

  set +e
  output="$(
    cd "${fixture}" &&
      PATH="${fixture}/stubbin:${PATH}" bash ./docker-build.sh <<'EOF'
2
y
EOF
  2>&1
  )"
  status=$?
  set -e

  [[ ${status} -eq 0 ]] || fail "expected build mode with codex bridge enabled to succeed, got: ${output}"
  assert_contains "$(cat "${docker_log}")" "compose --profile codex-bridge up -d --remove-orphans --pull never"
}

test_existing_image_mode_requires_a_local_image_when_compose_has_no_service_image
test_usage_preservation_targets_the_published_host_port
test_build_mode_can_enable_codex_bridge_profile

echo "PASS: docker build script regression tests"
