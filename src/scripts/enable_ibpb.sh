# Copyright 2023 Google LLC
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Run this script to enable IBPB mitigations on COS VMs. Secure boot must be
# disabled.
#
# WARNING: While this mitigation will improve the security of your nodes, it is
# known to have a _dramatic_ performance impact on some workloads - on the order
# of ~50% worse performance on AMD processors. Be careful when deploying this
# script.

set -o errexit
set -o pipefail
set -o nounset

function check_not_secure_boot() {
  if [[ ! -d "/sys/firmware/efi" ]]; then
    return
  fi
  efi="$(mktemp -d)"
  mount -t efivarfs none "${efi}"
  secure_boot="$(cat "${efi}"/SecureBoot-* | python -c 'import sys; print(sys.stdin.buffer.read() == b"\x06\x00\x00\x00\x01")')"
  umount "${efi}"
  rmdir "${efi}"
  if [[ "${secure_boot}" == "True" ]]; then
    echo "Secure Boot is enabled. Boot options cannot be changed. You must disable secure boot to enable IBPB mitigations."
    exit 1
  fi
}

function main() {
  if grep " retbleed=ibpb " /proc/cmdline > /dev/null; then
    echo "'retbleed=ibpb' already present on the kernel command line. Nothing to do."
    return
  fi
  echo "Attempting to set 'retbleed=ibpb' on the kernel command line."
  if [[ "${EUID}" -ne 0 ]]; then
    echo "This script must be run as root."
    return 1
  fi
  check_not_secure_boot

  dir="$(mktemp -d)"
  mount /dev/disk/by-partlabel/EFI-SYSTEM "${dir}"
  sed -i -e "s|cros_efi|cros_efi retbleed=ibpb|g" "${dir}/efi/boot/grub.cfg"
  umount "${dir}"
  rmdir "${dir}"
  echo "Rebooting."
  reboot
}

main
