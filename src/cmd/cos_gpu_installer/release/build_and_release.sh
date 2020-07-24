#!/bin/bash
#
# Script to build and release cos-gpu-installer to GCR.
#
# Usage: ./build_and_release.sh <project_id> <tag>
#
# Example: ./build_and_release.sh cos-cloud v2.0.1

set -o errexit
set -o pipefail
set -o nounset

SCRIPTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"

build() {
  pushd "$(dirname "${SCRIPTDIR}")"
  go build -o release/cos-gpu-installer main.go
  popd
}

release() {
  project_id="$1"
  tag="$2"
  pushd "${SCRIPTDIR}"
  gcloud builds submit . --config cloud_build_request.yaml --substitutions _PROJECT_ID="${project_id}",TAG_NAME="${tag}"
  rm cos-gpu-installer
  popd
}

main() {
  build
  release "$@"
}

main "$@"
