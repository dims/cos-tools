#!/bin/bash
#
# Copyright 2023 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o pipefail

usage() {
  cat <<'EOF'
Usage: ./run_unit_tests.sh
run_unit_tests.sh runs all unit tests under src/pkg and src/cmd in the cos/tools
repository.

EOF
}

setup_test() {
  # build changelogctl binary
  go build -o src/cmd/changelogctl/changelogctl src/cmd/changelogctl/main.go

  apt-get update && apt-get install -y sudo fdisk sysstat mtools

  # clean up to save disk space
  rm -rf "$(readlink -f bazel-bin)"
  rm -rf "$(readlink -f bazel-out)"
  rm -rf "$(readlink -f bazel-testlogs)"
  rm -rf "$(readlink -f bazel-tools)"

  rm -rf bazel-*
}

main() {
  cd /workspace
  setup_test

  # Run tests from each directory where go test files are present.
  test_dirs=$(find /workspace/src -name "*_test.go" -printf "%h\n" | sort | uniq)
  local exit_code=0
  for test_dir in ${test_dirs}; do
    cd "${test_dir}"
    # TODO(nrengaraj@): run profilertest in separate docker image.
    # If the directory contains BUILD.bazel, the tests have already
    # run using bazel in the previous step.
    if [ ! -f "BUILD.bazel" ] || [[ ${test_dir} == *"pkg/utils"* ]]; then
      if [[ ${test_dir} != *"profilertest"* ]]; then
        echo "Running tests for ${test_dir}"
        # TestCheckDrivers is outdated, the older COS milestones are not supported
        go test -v -skip "(TestFindCL|TestCheckDrivers)"
        if [ $? -ne 0 ]; then
          exit_code=1
        fi
      fi
    fi
  done
  return "${exit_code}"
}

main
