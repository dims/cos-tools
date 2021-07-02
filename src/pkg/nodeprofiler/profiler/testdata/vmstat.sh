#!/bin/bash

main () {
    if [[ "$#" -ne 3 ]]; then
     echo "command not called with 3 arguments" >&2 return 1
    fi
    if [[ $3 -lt 0 ]]; then
        echo "$3 is not a valid argument" >&2 return 1
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
        "0")
        cat <<EOF
        procs -----------memory---------- ---swap-- -----io---- -system-- ------cpu-----
        r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st
        9  0      0 14828152      0  25740    0    0     5     5   69   98  1  0 90  9  0
EOF
        ;;
        "+5")
        cat <<EOF
        procs -----------memory---------- ---swap-- -----io---- -system-- ------cpu-----
        r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st
        3  0      0 14828112      0  25740    0    0     3     6   85  121  1  0 93  6  0
        1  0      0 14828112      0  25740    0    0     0     0  854 1098  1  1 98  0  0
        1  0      0 14828112      0  25740    0    0     0     0 1012 1399  2  1 98  0  0
        1  0      0 14828112      0  25740    0    0     0     0 2991 4478  5  2 92  0  0
        2  0      0 14828112      0  25740    0    0     0     0 7698 8623 18  6 75  0  1
EOF
        ;;
        "1")
        cat <<EOF
        procs -----------memory---------- ---swap-- -----io---- -system-- ------cpu-----
        r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st
        2  0      0 14827724      0  25608    0    0     1     6   10   37  1  0 96  2  0
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


