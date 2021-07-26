#!/bin/bash

main() {
    cat <<EOF
    Filesystem      Size  Used Avail Use% Mounted on
    /dev/vdb        7.5G  4.6G  2.2G  68% /    
    tmpfs           100K     0  100K   0% /dev/lxd
    tmpfs           7.1G  121M  7.0G   2% /dev/shm
    run             7.1G   28K  7.1G   1% /dev/.host_ip
    /dev/root       173M  170M     0 100% /dev/.ssh/sshd_config
EOF
}
main "$#"