#!/usr/bin/env bash
set -euo pipefail

repo_root=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"

run_tests() {
  (cd infra_tests && go test -v -count=1 -timeout 30m)
}

case ${TEST_BACKEND:-} in
  ''|localstack)
    run_tests
    exit
    ;;
  aws-emulator) ;;
  *)
    printf 'invalid TEST_BACKEND: %s (want localstack or aws-emulator)\n' "$TEST_BACKEND" >&2
    exit 2
    ;;
esac

fail() {
  printf '%s\n' "$*" >&2
  exit 1
}

for tool in docker jq curl aws go terraform; do
  command -v "$tool" >/dev/null || fail "AWS Emulator requires $tool on PATH"
done

manifest=infra/test-backends.json
endpoint_base=$(jq -er '.["aws-emulator"].endpoint_base' "$manifest")
default_port=$(jq -er '.["aws-emulator"].port' "$manifest")
default_region=$(jq -er '.["aws-emulator"].region' "$manifest")
port=${AWS_EMULATOR_PORT:-$default_port}
region=${AWS_EMULATOR_REGION:-$default_region}
if ! [[ $port =~ ^[1-9][0-9]{0,4}$ ]] || (( port > 65535 )); then
  fail "invalid AWS Emulator port: $port (want a decimal integer from 1 to 65535 without leading zeros)"
fi

docker compose version >/dev/null || fail 'AWS Emulator requires Docker Compose'
aws_version=$(aws --version 2>&1) || fail "could not run aws --version: $aws_version"
[[ $aws_version == aws-cli/2.* ]] || fail "AWS Emulator requires AWS CLI v2 (found $aws_version)"

docker_arch=$(docker info --format '{{.Architecture}}')
case $docker_arch in
  aarch64|arm64) export EMULATOR_PLATFORM_ARCH=arm64 EMULATOR_RPM_ARCH=aarch64 ;;
  x86_64|amd64) export EMULATOR_PLATFORM_ARCH=amd64 EMULATOR_RPM_ARCH=x86_64 ;;
  *) fail "unsupported Docker server architecture: $docker_arch" ;;
esac
EMULATOR_UID=$(id -u)
EMULATOR_GID=$(id -g)
export EMULATOR_UID EMULATOR_GID EMULATOR_PORT="$port"

mkdir -p "$HOME/.aws/emulator"
while IFS= read -r name; do
  case $name in AWS_*) unset "$name" ;; esac
done < <(compgen -e)
unset TERRATEST_IAM_ROLE TF_DATA_DIR TF_WORKSPACE

aws_dir="$HOME/.aws/emulator/$port"
export AWS_ENDPOINT_URL="$endpoint_base:$port"
export AWS_CONFIG_FILE="$aws_dir/config"
export AWS_SHARED_CREDENTIALS_FILE="$aws_dir/emulator_credentials"
export AWS_CA_BUNDLE="$aws_dir/ca.pem"
export AWS_REGION="$region" AWS_DEFAULT_REGION="$region"
export AWS_EC2_METADATA_DISABLED=true AWS_DISABLE_HOST_PREFIX_INJECTION=true AWS_PAGER=''
export NO_PROXY="${NO_PROXY:+$NO_PROXY,}127.0.0.1"
export no_proxy="${no_proxy:+$no_proxy,}127.0.0.1"

project="terratest-aws-emulator-$port-$$"
compose() {
  docker compose -f docker-compose.aws-emulator.yml -p "$project" "$@"
}

cleanup() {
  local status=$? teardown_status
  trap - EXIT
  trap '' INT TERM
  if (( status != 0 )); then
    compose logs aws-emulator >&2 || :
  fi
  if compose down; then
    exit "$status"
  else
    teardown_status=$?
    printf 'could not stop Compose project %s (status %s)\n' "$project" "$teardown_status" >&2
    if (( status == 0 )); then
      exit "$teardown_status"
    fi
    exit "$status"
  fi
}

printf 'AWS Emulator Compose project: %s\n' "$project"
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
compose up -d --build aws-emulator

health=''
for ((attempt = 1; attempt <= 30; attempt++)); do
  if [[ -r $AWS_CONFIG_FILE && -s $AWS_CONFIG_FILE &&
        -r $AWS_SHARED_CREDENTIALS_FILE && -s $AWS_SHARED_CREDENTIALS_FILE &&
        -r $AWS_CA_BUNDLE && -s $AWS_CA_BUNDLE ]] &&
     health=$(curl --fail --silent --connect-timeout 1 --max-time 2 \
       --cacert "$AWS_CA_BUNDLE" "$AWS_ENDPOINT_URL/__admin/health") &&
     [[ $health == ok ]]; then
    break
  fi
  if (( attempt < 30 )); then sleep 1; fi
done
(( attempt <= 30 )) || fail "AWS Emulator did not become ready at $AWS_ENDPOINT_URL"
printf 'AWS Emulator health: %s\n' "$health"

identity=$(aws sts get-caller-identity --output json --no-cli-pager)
printf '%s\n' "$identity"
jq -e '.Account == "000000000000"' <<< "$identity" >/dev/null || fail 'unexpected AWS Emulator STS account'

run_tests
