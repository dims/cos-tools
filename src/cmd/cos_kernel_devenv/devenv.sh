#!/bin/bash
# Create kernel development environment for COS

set -o errexit
set -o pipefail

ROOT_MOUNT_DIR="${ROOT_MOUNT_DIR:-/root}"
RETRY_COUNT=${RETRY_COUNT:-5}

readonly COS_CI_DOWNLOAD_GCS="gs://cos-infra-prod-artifacts-official"
readonly CHROMIUMOS_SDK_GCS="https://storage.googleapis.com/chromiumos-sdk"
readonly TOOLCHAIN_URL_FILENAME="toolchain_path"
readonly KERNEL_HEADERS="kernel-headers.tgz"
readonly KERNEL_HEADERS_DIR="kernel-headers"
readonly TOOLCHAIN_ARCHIVE="toolchain.tar.xz"
readonly TOOLCHAIN_ENV_FILENAME="toolchain_env"
ROOT_OS_RELEASE="${ROOT_MOUNT_DIR}/etc/os-release"
readonly RETCODE_ERROR=1
RELEASE_ID=""  # Loaded from host during execution
BUILD_DIR="" # based on RELEASE_ID
KERNEL_CONFIG="defconfig"
BUILD_DEBUG_PACKAGE="false"
BUILD_HEADERS_PACKAGE="false"
CLEAN_BEFORE_BUILD="false"

BOARD=""
BUILD_ID=""

# official release, CI build, or cross-toolchain
MODE=""

CROS_TC_VERSION="2021.06.26.094653"
CROS_TC_DOWNLOAD_GCS="https://storage.googleapis.com/chromiumos-sdk/"

# Can be overridden by the command-line argument
TOOLCHAIN_ARCH="x86_64"
KERNEL_ARCH="x86"

# CC and CXX will be set by set_compilation_env
CC=""
CXX=""

# Use out-of-tree build for full kernel build
KBUILD_OUTPUT="."

_log() {
  local -r prefix="$1"
  shift
  echo "[${prefix}$(date -u "+%Y-%m-%d %H:%M:%S %Z")] ""$*" >&2
}

info() {
  _log "INFO    " "$*"
}

warn() {
  _log "WARNING " "$*"
}

error() {
  _log "ERROR   " "$*"
}

#######################################
# Choose the public GCS bucket of COS to fetch files from
# "cos-tools", "cos-tools-eu" and "cos-tools-asia"
# based on where the VM is running.
# Arguments:
#   None
# Globals:
#   COS_DOWNLOAD_GCS
#######################################
get_cos_tools_bucket() {
	# Get the zone the VM is running in.
	# Example output: projects/438692578867/zones/us-west2-a
	# If not running on GCE, use "cos-tools" by default.
	metadata_zone="$(curl -s -H Metadata-Flavor:Google http://metadata/computeMetadata/v1/instance/zone)" || {
		readonly COS_DOWNLOAD_GCS="https://storage.googleapis.com/cos-tools"
		return
	}
	zone="$( echo $metadata_zone | rev | cut -d '/' -f 1 | rev )"
	prefix="$( echo $zone | cut -d '-' -f 1 )"
	case $prefix in
		"us" | "northamerica" | "southamerica")
			readonly COS_DOWNLOAD_GCS="https://storage.googleapis.com/cos-tools"
			;;
		"europe")
			readonly COS_DOWNLOAD_GCS="https://storage.googleapis.com/cos-tools-eu"
			;;
		"asia" | "australia")
			readonly COS_DOWNLOAD_GCS="https://storage.googleapis.com/cos-tools-asia"
			;;
		*)
			readonly COS_DOWNLOAD_GCS="https://storage.googleapis.com/cos-tools"
			;;
	esac
}

load_etc_os_release() {
  if [[ ! -f "${ROOT_OS_RELEASE}" ]]; then
    error "File ${ROOT_OS_RELEASE} not found, /etc/os-release from COS host must be mounted."
    exit ${RETCODE_ERROR}
  fi
  . "${ROOT_OS_RELEASE}"
  info "Running on COS build id ${RELEASE_ID}"
}

download_from_url() {
  local -r url="$1"
  local -r output="$2"
  info "Downloading from URL: ${url}"
  info "Local download location: ${output}"
  local attempts=0
  until curl --http1.1 -sfS "${url}" -o "${output}"; do
    attempts=$(( attempts + 1))
    if (( "${attempts}" >= "${RETRY_COUNT}" )); then
      error "Could not download from ${url}"
      return ${RETCODE_ERROR}
    fi
    warn "Error downloading from ${url}, retrying"
    sleep 1
  done
  info "Download finished"
}

download_from_gcs() {
  local -r url="$1"
  local -r output="$2"
  info "Downloading from Google Storage: ${url}"
  info "Local download location: ${output}"
  local attempts=0
  until gsutil -q cp "${url}" "${output}"; do
    attempts=$(( attempts + 1))
    if (( "${attempts}" >= "${RETRY_COUNT}" )); then
      error "Could not download from ${url}"
      return ${RETCODE_ERROR}
    fi
    warn "Error downloading from ${url}, retrying"
    sleep 1
  done
  info "Download finished"
}

# Get toolchain tarball from Chromium GCS bucket when
# toolchain tarball is not found in COS GCS bucket
get_cross_toolchain_pkg() {
  # First, check if the toolchain path is available locally.
  local -r tc_path_file="${ROOT_MOUNT_DIR}/etc/toolchain-path"
  if [[ -f "${tc_path_file}" ]]; then
    info "Found toolchain path file locally"
    local -r tc_path="$(cat "${tc_path_file}")"
    local -r tc_download_url="${CHROMIUMOS_SDK_GCS}/${tc_path}"
  else
    # Next, check if the toolchain path is available in GCS.
    local -r tc_path_url="${COS_DOWNLOAD_GCS}/${RELEASE_ID}/${TOOLCHAIN_URL_FILENAME}"
    info "Obtaining toolchain download URL from ${tc_path_url}"
    local -r tc_download_url="$(curl --http1.1 -sfS "${tc_path_url}")"
  fi
  echo "${tc_download_url}"
}

install_cross_toolchain_pkg() {
  local -r download_url=$1
  info "Downloading prebuilt toolchain from ${download_url}"
  local -r pkg_name="$(basename "${download_url}")"
  download_from_url "${download_url}" "${BUILD_DIR}/${pkg_name}"
  # Don't unpack Rust toolchain elements because they are not needed and they
  # use a lot of disk space.
  tar axf "${BUILD_DIR}/${pkg_name}" -C "${BUILD_DIR}" \
    --exclude='./usr/lib64/rustlib*' \
    --exclude='./usr/lib64/libstd-*.so' \
    --exclude='./lib/libstd-*.so' \
    --exclude='./lib/librustc*' \
    --exclude='./usr/lib64/librustc*'
  rm "${BUILD_DIR}/${pkg_name}"
}

# Set-up compilation environment using toolchain used for
# kernel compilation
install_release_cross_toolchain() {
  info "Downloading and installing a toolchain"
  # Get toolchain_env path from COS GCS bucket
  local -r tc_env_file_path="${COS_DOWNLOAD_GCS}/${RELEASE_ID}/${TOOLCHAIN_ENV_FILENAME}"
  info "Obtaining toolchain_env file from ${tc_env_file_path}"

  # Download toolchain_env if present
  if ! download_from_url "${tc_env_file_path}" "${BUILD_DIR}/${TOOLCHAIN_ENV_FILENAME}"; then
    error "Failed to download toolchain file"
    error "Make sure build id '$RELEASE_ID' is valid"
    return ${RETCODE_ERROR}
  fi

  local -r tc_download_url="${COS_DOWNLOAD_GCS}/${RELEASE_ID}/${TOOLCHAIN_ARCHIVE}"

  # Install toolchain pkg
  install_cross_toolchain_pkg "${tc_download_url}"
}

install_release_kernel_headers() {
  info "Downloading and installing a kernel headers"
  local -r kernel_headers_file_path="${COS_DOWNLOAD_GCS}/${RELEASE_ID}/${KERNEL_HEADERS}"
  info "Obtaining kernel headers file from ${kernel_headers_file_path}"

  if ! download_from_url "${kernel_headers_file_path}" "${BUILD_DIR}/${KERNEL_HEADERS}"; then
        return ${RETCODE_ERROR}
  fi
  mkdir -p "${BUILD_DIR}/${KERNEL_HEADERS_DIR}"
  tar axf "${BUILD_DIR}/${KERNEL_HEADERS}" -C "${BUILD_DIR}/${KERNEL_HEADERS_DIR}"
  rm -f "${BUILD_DIR}/${KERNEL_HEADERS}"
}

# Download and install toolchain from the CI or tryjob build directory
install_build_cross_toolchain() {
  local -r bucket="$1"

  info "Downloading and installing a toolchain"
  # Get toolchain_env path from COS GCS bucket
  local -r tc_env_file_path="${bucket}/${TOOLCHAIN_ENV_FILENAME}"
  local -r tc_url_file_path="${bucket}/${TOOLCHAIN_URL_FILENAME}"

  info "Obtaining toolchain_env file from ${tc_env_file_path}"

  # Download toolchain_env if present
  if ! download_from_gcs "${tc_env_file_path}" "${BUILD_DIR}/${TOOLCHAIN_ENV_FILENAME}"; then
        error "Failed to download toolchain file"
        error "Make sure build id '$RELEASE_ID' is valid"
        return ${RETCODE_ERROR}
  fi

  # Download toolchain_path if present
  if ! download_from_gcs "${tc_url_file_path}" "${BUILD_DIR}/${TOOLCHAIN_URL_FILENAME}"; then
        error "Failed to download toolchain file"
        error "Make sure build id '$RELEASE_ID' is valid"
        return ${RETCODE_ERROR}
  fi

  local -r tc_download_url="${CROS_TC_DOWNLOAD_GCS}$(cat ${BUILD_DIR}/${TOOLCHAIN_URL_FILENAME})"
  if [[ -z "$tc_download_url" ]]; then
    error "Failed to download toolchain URL file"
    error "Make sure build id '$RELEASE_ID' is valid"
    return ${RETCODE_ERROR}
  fi

  # Install toolchain pkg
  install_cross_toolchain_pkg "${tc_download_url}"
}

install_build_kernel_headers() {
  local -r bucket="$1"

  info "Downloading and installing a kernel headers"
  local -r kernel_headers_file_path="${bucket}/${KERNEL_HEADERS}"
  info "Obtaining kernel headers file from ${kernel_headers_file_path}"

  if ! download_from_gcs "${kernel_headers_file_path}" "${BUILD_DIR}/${KERNEL_HEADERS}"; then
        return ${RETCODE_ERROR}
  fi
  mkdir -p "${BUILD_DIR}/${KERNEL_HEADERS_DIR}"
  tar axf "${BUILD_DIR}/${KERNEL_HEADERS}" -C "${BUILD_DIR}/${KERNEL_HEADERS_DIR}"
  rm -f "${BUILD_DIR}/${KERNEL_HEADERS}"
}

install_generic_cross_toolchain() {
  info "Downloading and installing a toolchain"
  # Download toolchain_env if present
  local -r tc_date="$(echo ${CROS_TC_VERSION} | sed  -E 's/\.(..).*/\/\1/')"
  local -r tc_download_url="${CROS_TC_DOWNLOAD_GCS}${tc_date}/${TOOLCHAIN_ARCH}-cros-linux-gnu-${CROS_TC_VERSION}.tar.xz"

  # Install toolchain pkg
  install_cross_toolchain_pkg "${tc_download_url}"
}

set_compilation_env() {
  local -r tc_env_file_path="${COS_DOWNLOAD_GCS}/${RELEASE_ID}/${TOOLCHAIN_ENV_FILENAME}"
  # toolchain_env file will set 'CC' and 'CXX' environment
  # variable based on the toolchain used for kernel compilation
  if [[ -f "${BUILD_DIR}/${TOOLCHAIN_ENV_FILENAME}" ]]; then
    source "${BUILD_DIR}/${TOOLCHAIN_ENV_FILENAME}"
    export CC
    export CXX
  else
    export CC="${TOOLCHAIN_ARCH}-cros-linux-gnu-clang"
    export CXX="${TOOLCHAIN_ARCH}-cros-linux-gnu-clang"
  fi
  info "Configuring environment variables for cross-compilation"
  # CC and CXX are already set in toolchain_env
  export PATH="${BUILD_DIR}/bin:${BUILD_DIR}/usr/bin:${PATH}"
  export SYSROOT="${BUILD_DIR}/usr/${TOOLCHAIN_ARCH}-cros-linux-gnu"
  export HOSTCC="x86_64-pc-linux-gnu-clang"
  export HOSTCXX="x86_64-pc-linux-gnu-clang++"
  export LD="${TOOLCHAIN_ARCH}-cros-linux-gnu-ld.lld"
  export HOSTLD="x86_64-pc-linux-gnu-ld.lld"
  export OBJCOPY=llvm-objcopy
  export STRIP=llvm-strip
  export KERNEL_ARCH
  export TOOLCHAIN_ARCH
  export LLVM_IAS=1
  if [[ "${MODE}" = "release" || "${MODE}" = "build" || "${MODE}" = "custom" ]]; then
    local -r headers_dir=$(ls -d ${BUILD_DIR}/${KERNEL_HEADERS_DIR}/usr/src/linux-headers*)
    export KHEADERS="${headers_dir}"
  fi
}

kmake() {
  local output_dir_arg="KBUILD_OUTPUT="
  if [[ "${KBUILD_OUTPUT}" != "." ]]; then
    output_dir_arg="KBUILD_OUTPUT=${KBUILD_OUTPUT}"
  fi
  env ARCH=${KERNEL_ARCH} make ARCH=${KERNEL_ARCH} \
    CC="${CC}" CXX="${CXX}" LD="${LD}" \
    STRIP="${STRIP}" OBJCOPY="${OBJCOPY}" \
    HOSTCC="${HOSTCC}" HOSTCXX="${HOSTCXX}" HOSTLD="${HOSTLD}" \
    "${output_dir_arg}" \
    "$@"
}
export -f kmake

tar_kernel_headers() {
  local -r version=$(kmake "$@" -s kernelrelease)
  local -r tmpdir="$(mktemp -d)"
  (
    find . -name Makefile\* -o -name Kconfig\* -o -name \*.pl
    find arch/*/include include scripts -type f -o -type l
    find "arch/${KERNEL_ARCH}" -name module.lds -o -name Kbuild.platforms -o -name Platform
    find "arch/${KERNEL_ARCH}" -name include -o -name scripts -type d | while IFS='' read -r line; do
      find "${line}" -type f
    done
  ) > "${tmpdir}/hdrsrcfiles"

  pushd "${KBUILD_OUTPUT}"
  (
    if [[ -d tools/objtool ]]; then
      find tools/objtool -type f -executable
    fi
    find "arch/${KERNEL_ARCH}/include" Module.symvers System.map \
      include scripts .config \
      -type f ! -name "*.cmd"  ! -name "*.o"
  ) > "${tmpdir}/hdrobjfiles"
  popd

  local -r destdir="${tmpdir}/headers_tmp/usr/src/linux-headers-${version}"
  mkdir -p "${destdir}"
  mkdir -p "${destdir}/build"
  tar -c -f - -T "${tmpdir}/hdrsrcfiles" | tar -xf - -C "${destdir}"
  # separate generated files and main sources for now
  # this is to prevent breakage in linux-info.eclass that
  # rely on src and build being separated
  tar -c -f - -C ${KBUILD_OUTPUT} -T "${tmpdir}/hdrobjfiles" | tar -xf - -C "${destdir}/build"
  echo "include ../Makefile" > "${destdir}/build/Makefile"

  rm "${tmpdir}/hdrsrcfiles" "${tmpdir}/hdrobjfiles"

  tar -C "${tmpdir}/headers_tmp" -c -z -f "cos-kernel-headers-${version}-${KERNEL_ARCH}.tgz" .
  rm -rf "${tmpdir}"
}

kernel_build() {
  local -r tmproot_dir="$(mktemp -d)"
  local image_target

  case "${KERNEL_ARCH}" in
    x86)   image_target="bzImage" ;;
    arm64) image_target="Image" ;;
    *)
      echo "Unknown kernel architecture: ${KERNEL_ARCH}"
      exit $RETCODE_ERROR
      ;;
  esac

  if [[ "${CLEAN_BEFORE_BUILD}" = "true" ]]; then
    kmake "$@" mrproper
  fi
  kmake "$@" "${KERNEL_CONFIG}"
  local -r version=$(kmake "$@" -s kernelrelease)
  kmake "$@" "${image_target}" modules
  INSTALL_MOD_PATH="${tmproot_dir}" kmake "$@" modules_install

  mkdir -p "${tmproot_dir}/boot/"
  cp -v -- "${KBUILD_OUTPUT}/.config" "${tmproot_dir}/boot/config-${version}"
  cp -v -- "${KBUILD_OUTPUT}/arch/${KERNEL_ARCH}/boot/${image_target}" "${tmproot_dir}/boot/vmlinuz-${version}"

  for module in $(find "$tmproot_dir/lib/modules/" -name "*.ko" -printf '%P\n'); do
    module="lib/modules/$module"
    mkdir -p "$(dirname "$tmproot_dir/usr/lib/debug/$module")"
    # only keep debug symbols in the debug file
    $OBJCOPY --only-keep-debug "$tmproot_dir/$module" "$tmproot_dir/usr/lib/debug/$module"
    # strip original module from debug symbols
    $OBJCOPY --strip-debug "$tmproot_dir/$module"
    # then add a link to those
    $OBJCOPY --add-gnu-debuglink="$tmproot_dir/usr/lib/debug/$module" "$tmproot_dir/$module"
  done

  if [[ "${BUILD_DEBUG_PACKAGE}" = "true" ]]; then
    cp -v -- "${KBUILD_OUTPUT}/vmlinux" "${tmproot_dir}/usr/lib/debug/lib/modules/${version}/"
    # Some other tools expect other locations
    mkdir -p "$tmproot_dir/usr/lib/debug/boot/"
    ln -s "../lib/modules/$version/vmlinux" "$tmproot_dir/usr/lib/debug/boot/vmlinux-$version"
    ln -s "lib/modules/$version/vmlinux" "$tmproot_dir/usr/lib/debug/vmlinux-$version"
    tar -c -J -f "cos-kernel-debug-${version}-${KERNEL_ARCH}.txz" -C "${tmproot_dir}/usr/lib" debug/
  fi

  tar -c -J -f "cos-kernel-${version}-${KERNEL_ARCH}.txz" -C "${tmproot_dir}" boot/ lib/
  rm -rf "${tmproot_dir}"

  if [[ "${BUILD_HEADERS_PACKAGE}" = "true" ]]; then
    tar_kernel_headers
  fi
}

module_build() {
  kmake -C "${KHEADERS}" M="$(pwd)" "$@" clean
  kmake -C "${KHEADERS}" M="$(pwd)" "$@" modules
}

usage() {
cat 1>&2 <<__EOUSAGE__
Usage: $0 [-k | -m | -i] [-cdH] [-A <x86|arm64>]
    [-C <kernelconfig>] [-O  <objdir>]
    [-B <build> -b <board> | -R <release> | -G <bucket>]
    [-t <toolchain_version>] [VAR=value ...] [target ...]

Options:
  -A <arch>     target architecture. Valid values are x86 and arm64.
  -B <build>    seed the toolchain from the COS build <build>.
                Example: R93-16623.0.0 Requires -b option.
  -C <config>   kernel config target. Example: lakitu_defconfig
  -G <bucket>   seed the toolchain and kernel headers from the custom
                GCS bucket <bucket>. Directory structure needs to conform
                to the COS standard.
  -H            create a package with kernel headers for the respective
                kernel package. Should be used only with -k option.
  -O <objdir>   value for KBUILD_OUTPUT to separate obj files from
                sources
  -R <release>  seed the toolchain and kernel headers from the
                specified official COS release. Example: 16442.0.0
  -b <board>    specify board for -B argument. Example: lakitu
  -c            perform mrproper step when building a kernel package.
                Should be used only with -k option.
  -d            create a package with debug symbols for the respective
                kernel package. Should be used only with -k option.
  -h            show this message.
  -i            invoke interactive shell with kernel development
                environment initialized.
  -k            build a kernel package for sources mapped from the host
                to the current working directory.
  -m            build an out-of-tree module for sources mapped from
                the host to the current working directory.
                This mode requires either -R or -B/b options.
  -t            seed the toolchain from the Chromium OS upstream.
                Example: 2021.06.26.094653
__EOUSAGE__

  exit $RETCODE_ERROR
}

main() {
  local build_target=""
  local custom_bucket=""
  get_cos_tools_bucket
  while getopts "A:B:C:G:HO:R:b:cdhikmt:" o; do
    case "${o}" in
      A) KERNEL_ARCH=${OPTARG} ;;
      B) BUILD_ID=${OPTARG} ;;
      C) KERNEL_CONFIG=${OPTARG} ;;
      G) custom_bucket=${OPTARG} ;;
      H) BUILD_HEADERS_PACKAGE="true" ;;
      O) KBUILD_OUTPUT=${OPTARG} ;;
      R) RELEASE_ID=${OPTARG} ;;
      b) BOARD=${OPTARG} ;;
      c) CLEAN_BEFORE_BUILD="true" ;;
      d) BUILD_DEBUG_PACKAGE="true" ;;
      h) usage ;;
      i) build_target="shell" ;;
      k) build_target="kernel" ;;
      m) build_target="module" ;;
      t) CROS_TC_VERSION="${OPTARG}" ;;
      *) usage ;;
    esac
  done
  shift $((OPTIND-1))

  if [[ ! -z "${BOARD}" ]]; then
    case "${BOARD}" in
      lakitu-arm64) KERNEL_ARCH=arm64 ;;
      *) KERNEL_ARCH=x86 ;;
    esac
  fi

  case "${KERNEL_ARCH}" in
    x86)
      TOOLCHAIN_ARCH=x86_64
      ;;
    arm64)
      TOOLCHAIN_ARCH=aarch64
      ;;
    *)
      echo "Invalid -A value: $KERNEL_ARCH"
      usage
      ;;
  esac

  echo "** Kernel architecture: $KERNEL_ARCH"
  echo "** Toolchain architecture: $TOOLCHAIN_ARCH"

  if [[ -n "$RELEASE_ID" ]]; then
    MODE="release"
    BUILD_DIR="/build/${TOOLCHAIN_ARCH}-${RELEASE_ID}"
    echo "** COS release: $RELEASE_ID"
  fi

  if [[ -z "$MODE" && -n "$BOARD" && -n "$BUILD_ID" ]]; then
    MODE="build"
    echo "** COS build: $BOARD-$BUILD_ID"
    BUILD_DIR="/build/${BOARD}-${BUILD_ID}"
  fi

  if [[ -z "$MODE" && -n "$custom_bucket" ]]; then
    MODE="custom"
    BUILD_DIR="/build/$(basename "${custom_bucket}")"
  fi

  if [[ -z "$MODE" ]]; then
    MODE="cross"
    BUILD_DIR="/build/cros-${CROS_TC_VERSION}-${TOOLCHAIN_ARCH}"
  fi
  echo "Mode: $MODE"

  if [[ ! -d ${BUILD_DIR} ]]; then
    mkdir -p "${BUILD_DIR}"
    case "$MODE" in
      cross) install_generic_cross_toolchain ;;
      release)
        install_release_cross_toolchain
        install_release_kernel_headers
        ;;
      build)
        local -r bucket="${COS_CI_DOWNLOAD_GCS}/${BOARD}-release/${BUILD_ID}"
        install_build_cross_toolchain "${bucket}"
        install_build_kernel_headers "${bucket}"
        ;;
      custom)
        install_build_cross_toolchain "${custom_bucket}"
        install_build_kernel_headers "${custom_bucket}"
        ;;
    esac
  fi

  set_compilation_env

  case "${build_target}" in
    kernel) kernel_build -j"$(nproc)" ;;
    module) module_build -j"$(nproc)" ;;
    shell)
      echo "Starting interactive shell for the kernel devenv"
      /bin/bash
      ;;
    *) kmake -j"$(nproc)" "$@" ;;
  esac
}

main "$@"
