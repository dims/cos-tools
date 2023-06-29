#!/bin/bash
# Run xfstests for a given file system configuration.

set -eu
set -o pipefail

readonly PROG_NAME=$(basename "$0")
readonly XFSTESTS_BLD_DIR="/home/fstests"
readonly XFSTESTS_CFG_DIR="test-appliance/files/root/fs/"

CONFIG=""
INSTANCE_NAME=""
RESULTS_BUCKET=""
PROJECT=""
ZONE=""
GROUP="auto"
XFSTESTS_EXCLUDED=""
ARCH="x86_64"

#
# usage <exit_code>
#
# Print usage and exit.
#
usage() {
        local exit_code="${1}"

        cat <<EOF
Usage:
        ${PROG_NAME} [-x <xfstests_config>] [-n <instance_name>]
        -x, --xfstests_config   the configuration to run xfstests on
        -n, --instance_name     the name of the instance running xfstests
        -r, --result_bucket     the GCS bucket to store test result
        -p, --project           the GCP project to run the test VM
        -z, --zone              the GCE zone to run the test VM
        -g, --group             the xfstest group to run
        -b, --blocked_tests     the list of tests to not run
        -i, --rootfs_image      the test appliance image to use
        -a, --arch              the architecture of the image

Note that this script expect kernel to be located at [result_bucket]/bzImage.

Examples:
        $ ${PROG_NAME} -x overlay -n xfs-vm -r gs://xfstests/R93-11647.62.0 -p cos-xfstests -z us-west1-c -b generic/269,generic/500

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

  args=$(getopt \
          --options "x:n:r:p:z:g:b:a:" \
          --longoptions "xfstests_config: instance_name: result_bucket: project: zone: group: blocked_tests: arch:" \
          -- "$@")
  [[ $? -eq 0 ]] || usage 1
  eval set -- "${args}"

  while :; do
    arg="${1}"
    shift
    case "${arg}" in
    -x|--xfstests_config) CONFIG="${1}"; shift ;;
    -n|--instance_name) INSTANCE_NAME="${1}"; shift ;;
    -r|--result_bucket) RESULTS_BUCKET="${1}"; shift ;;
    -p|--project) PROJECT="${1}"; shift ;;
    -z|--zone) ZONE="${1}"; shift ;;
    -g|--group) GROUP="${1}"; shift ;;
    -b|--blocked_tests) XFSTESTS_EXCLUDED="${1}"; shift ;;
    -i|--rootfs_image) TEST_APPLIANCE_IMAGE="${1}"; shift ;;
    -a|--arch) ARCH="${1}"; shift ;;
    --) break ;;
    *) echo "internal error parsing arguments!"; usage 1 ;;
    esac
  done

  if [[ -z "$CONFIG" || -z "$RESULTS_BUCKET" || -z "$PROJECT" || -z "$ZONE" ]] ; then
    usage 1
  fi

  if [[ -z "$INSTANCE_NAME" ]] ; then
    echo "Instance name missing. Randomly generating one."
    CONFIG_ALNUM=${CONFIG//[^[:alnum:]]/}
    INSTANCE_NAME="cos-xfstests-$(date +"%Y%m%d%H%M")${RANDOM}-${CONFIG_ALNUM}"
    echo "Instance name: $INSTANCE_NAME"
  else
    echo "Given instance name: $INSTANCE_NAME"
  fi
  INSTANCE_NAME=$(echo "${INSTANCE_NAME}" | sed 's/_/-/g')
  echo "Using instance name: $INSTANCE_NAME"
}

setup_xfstests_bld() {
    git clone https://github.com/tytso/xfstests-bld $XFSTESTS_BLD_DIR
    cd "$XFSTESTS_BLD_DIR" && make
}

#
# gce_xfstests_run
#
# Run the gce_xfstests command
#
gce_xfstests_run() {
  # The result bucket name without gs:// prefix.
  local bucket_name="${RESULTS_BUCKET#gs://}"
  if [[ "${ARCH}" = "x86_64" ]]; then
    kernel="${RESULTS_BUCKET}/bzImage"
  elif [[ "${ARCH}" = "arm64" ]]; then
    kernel="${RESULTS_BUCKET}/Image"
  else
    echo "Unknown architecture: ${ARCH}"
    exit 1
  fi

  # Sets up the config file for gce-xfstests.
  mkdir -p /root/.config
  cat <<EOF > /root/.config/gce-xfstests
GS_BUCKET=${bucket_name}
GCE_PROJECT=${PROJECT}
GCE_IMAGE_PROJECT=${PROJECT}
GCE_ZONE=${ZONE}
GCE_KERNEL=${kernel}
GCE_UPLOAD_SUMMARY=true
EOF

  local excluded_arg=""
  local excluded_val=""
  if [[ ! -z "${XFSTESTS_EXCLUDED}" ]] ; then
    # In COS xfsquick test suite has many excluded test cases(~120 test cases),
    # which are passed as command line arguments to gce-xfstests script.
    # The arguments that we are passing to gce-xfstests is passed as kernel
    # command line to kexec(This is added recently ). Since cos xfs is having
    # too many excluded test cases, the kernel command line becomes too big and
    # kexec fails with "Kernel command line too long for kernel! Cannot load /root/bzImage"
    # Hence adding the excluded tests in a file instead of passing as an argument.

    if [[ "${CONFIG}" = "xfs" ]] ; then
      echo "${XFSTESTS_EXCLUDED}" | sed 's/,/\n/g' >> \
           "${XFSTESTS_BLD_DIR}"/"${XFSTESTS_CFG_DIR}"/"${CONFIG}"/exclude
      excluded_arg="--update-files"
    else
      excluded_arg="-X"
      excluded_val="${XFSTESTS_EXCLUDED}"

    fi
  fi
  echo $XFSTESTS_BLD_DIR/gce-xfstests --instance-name "${INSTANCE_NAME}" \
    --kernel "${RESULTS_BUCKET}/bzImage" ${excluded_arg} "${excluded_val}" \
    -c "${CONFIG}" -g "${GROUP}" --arch "${ARCH}"
  # shellcheck disable=SC2086
  # putting ${excluded_val} in quotes causes gce-xfstests to fail because it
  # it takes empty value as an argument.
  $XFSTESTS_BLD_DIR/gce-xfstests --instance-name "${INSTANCE_NAME}" \
    --kernel "${kernel}" ${excluded_arg} ${excluded_val} \
    -c "${CONFIG}" -g "${GROUP}" --arch "${ARCH}"
}

main() {
  parse_args "$@"

  setup_xfstests_bld

  gce_xfstests_run
}

main "$@"
