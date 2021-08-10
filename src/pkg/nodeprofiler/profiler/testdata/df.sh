#!/bin/bash

main() {
    case "$2" in
    "1")
    cat <<EOF
    Filesystem     1K-blocks   Used Available Use% Mounted on
    /dev/vdb         7864320 5401876   1738492  76% /
    none                 492       0       492   0% /dev
    run              7433436      28   7433408   1% /dev/.host_ip
    /dev/root         176176  173936         0 100% /dev/.ssh/sshd_config
EOF
    ;;
    "2")
    cat <<EOF
    1K-blocks     Filesystem   Used Available Use% Mounted on
    /dev/vdb         7864320 5401876   1738492  76% /
    none                 492       0       492   0% /dev
    devtmpfs         7430944       0   7430944   0% /dev/tty
    /dev/vdb         7864320 5401876   1738492  76% /dev/kvm
    /dev/root         176176  173936         0 100% /dev/.ssh/sshd_config
EOF
    ;;
    esac
}

main "$@"