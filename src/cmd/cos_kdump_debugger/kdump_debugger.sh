#!/bin/bash
# This script sets up the environment for kernel crash dump debugging.

set -eu
set -o pipefail


readonly PROG_NAME=$(basename "$0")
readonly REPORT_TARBALL_DIR="/sos"
readonly REPORT_DIR="sos"
CRASH_COMMAND=""
VMLINUX_PATH=""

#
# usage <exit_code>
#
# Print usage and exit.
#
usage() {
        local exit_code="${1}"

        cat <<EOF
Usage:
        ${PROG_NAME} [-s <sos>] [-c <crash_command>] [-p <vmlinux_path>]
        -s, --sos            The filename of the sosreport tarball.
        -c, --crash_command  The crash command to run. Will open a shell for interactive debugging after running this command.
        -p, --vmlinux_path   The gsutil path to the matching vmlinux. If not set, the vmlinux will be fetched from gs://cos-tools.

Examples:
        $ ${PROG_NAME} --sos sosreport-kdump-next-20190130220549.tar.xz
        $ ${PROG_NAME} -s sosreport-kdump-next-20190130220549.tar.xz -c bt -p gs://cos-tools/15978.0.0/vmlinux

Note:
        Expecting the sosreport tarball to be located at REPORT_TARBALL_DIR.
EOF
        exit "${exit_code}"
}


#
# parse_args <args...>
#
# Parse command line arguments.
#
parse_args() {
  local args

  if ! args=$(getopt \
          --options "c: s: p: n" \
          --longoptions "crash_command: sos: vmlinux_path:" \
          -- "$@"); then
    usage 1
  fi
  eval set -- "${args}"

  while :; do
    arg="${1}"
    shift
    case "${arg}" in
    -c|--crash_command) CRASH_COMMAND="${1}"; shift ;;
    -s|--sos) REPORT_TARBALL="${1}"; shift ;;
    -p|--vmlinux_path) VMLINUX_PATH="${1}"; shift ;;
    --) break ;;
    *) echo "internal error parsing arguments!"; usage 1 ;;
    esac
  done
}


#
# setup
#
# Setup the work directory for debugging, containing:
# 1. Uncompressed sosreport tarball.
# 2. vmlinux for the COS kernel.
# 3. Latest kernel crash dump.
#
setup() {
  local sosreport_dir
  local buildnumber

  # We want all data get automatically removed after container exit.
  WORKDIR=$(mktemp -d)
  cd "${WORKDIR}"

  # Uncompress sosreport tarball, and rename the folder into $REPORT_TARBALL
  echo "Uncompressing sosreport tarball."
  tar -xf "${REPORT_TARBALL_DIR}/${REPORT_TARBALL}"
  sosreport_dir=$(find ./sosreport-* -maxdepth 0 | sed -n 1p)
  mv "${sosreport_dir}" "${REPORT_DIR}"

  # Copy the latest kernel crash dump to the workdir for easy access
  cp "${REPORT_DIR}/var/kdump" .

  # If VMLINUX_PATH is not set, fetch the vmlinux from gs://cos-tools
  if [[ -z "${VMLINUX_PATH}" ]]; then
    echo "--vmlinux_path not set, fetching vmlinux from gs://cos-tools"
    buildnumber=$(grep BUILD_ID "${REPORT_DIR}/etc/os-release" | cut -d "=" -f 2)
    VMLINUX_PATH="gs://cos-tools/${buildnumber}/vmlinux"
  fi

  echo "Downloading ${VMLINUX_PATH}."
  gsutil -q cp "${VMLINUX_PATH}" .
}


#
# run
#
# This steps will start inspect the kernel crash dump:
# If CRASH_COMMAND is set, this step will execute the given command.
# Otherwise, this step will open a shell for interactive debugging.
#
run() {
  if [[ -n "${CRASH_COMMAND}" ]]; then
    echo "Running crash command: ${CRASH_COMMAND}."
    echo -e "${CRASH_COMMAND}\nq" | crash vmlinux kdump
  fi

  # Give control to user, if no debugging command is specified.
  exec /bin/bash
}


main() {
  parse_args "$@"

  setup

  run
}

main "$@"
