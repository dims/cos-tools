#!/bin/bash

main() {
    cat <<EOF
                  total        used        free      shared  buff/cache   available
    Mem:          14520          13       14481           0          25       14506
    Swap:             0           0           0
EOF
}

main "$#"