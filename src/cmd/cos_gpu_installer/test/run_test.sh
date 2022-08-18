#!/bin/bash

set -o errexit
set -o pipefail
set -o nounset

readonly PROG_NAME="$(basename "$0")"
readonly SCRIPT_DIR="$(dirname "$0")"
readonly PROTOC_BIN="protoc"
readonly GCLOUD_BIN="gcloud"

usage() {
  cat <<EOF

${PROG_NAME}: Run check_drivers_test.go to check COS precompiled drivers availability.

Prerequisites:
    The following commands have to be installed and be able to found in \$PATH:
        \`gcloud\`: https://cloud.google.com/sdk/
        \`protoc\`: https://github.com/protocolbuffers/protobuf

    Besides, the test uses Application Default Credentials for authentication. So you need to run \`gcloud auth application-default login\` to set up ADC.
EOF
  exit "${1}"
}

check_command_exist() {
    cmd="$1"
    command -v "${cmd}" &> /dev/null
}

check_application_default_credentials() {
    "${GCLOUD_BIN}" auth application-default print-access-token 1> /dev/null
}

check_prerequisites() {
    check_command_exist "${PROTOC_BIN}" && \
      check_command_exist "${GCLOUD_BIN}" && \
      check_application_default_credentials
}

compile_proto() {
    ~/protoc/bin/protoc -I "${SCRIPT_DIR}"/../versions --go_out=paths=source_relative:"${SCRIPT_DIR}"/../versions "${SCRIPT_DIR}"/../versions/versions.proto
}


run_test() {
    go test -v "${SCRIPT_DIR}"/check_drivers_test.go
}

main() {
    echo "Checking prerequisites..."
    set +e
    if ! check_prerequisites; then
      usage 1
    fi
    set -e

    echo "Compiling protobuf..."
    compile_proto

    echo "Running test..."
    run_test
}

main
