#!/bin/bash

main () {
    if [[ $3 -lt 0 ]]; then
        echo "$3 is not a valid argument" 1>&2 
        return 1
    fi
    case "$3" in
        "3")
        cat << EOF
        procs -----------memory---------- ---swap-- -----io---- -system-- ------cpu-----
        r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st
        5  0      0 14827096      0  25608    0    0     2     5   57    2  1  0 99  0  0
        2  0      0 14827096      0  25608    0    0     0     0 1131 1594  2  1 97  0  0
        2  0      0 14827096      0  25608    0    0     0     0 5283 8037  7  3 90  0  0
EOF
        ;;
        "4")
        cat <<EOF
        procs -----------memory---------- ---swap-- -----io---- -system-- ------cpu-----
        r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st


        1  0      0 14827712      0  25608    0    0     0     0 2561 3731  7  2 91  0  0
        0  0      0 14827712      0  25608    0    0     0     0 1885 2684  3  2 95  0  0

        0  0      0 14827712      0  25608    0    0     0     0  827  894  1  2 98  0  0

        5  0      0 14827780      0  25740    0    0     0     5    3    7  1  0 96  3  0
EOF
        ;;
        "2")
        cat <<EOF
        procs--sys--cpu--
        r  us sy id wa st
        3   1  0 96  3  0
        1   2  1 98  0  0
EOF
        ;;
    esac
}

main "$@"


