#!/bin/bash

main() {
    case "$1" in 
    "-dxz")
    cat <<EOF
    Linux 5.4.109-26092-g9d947a4eeb73 (penguin)     07/09/2021      _x86_64_        (8 CPU)

    Device            r/s     w/s     rkB/s     wkB/s   rrqm/s   wrqm/s  %rrqm  %wrqm r_await w_await aqu-sz rareq-sz wareq-sz  svctm  %util
    vdb              0.01    0.60      0.86     21.39     0.00     0.20   0.24  25.16    8.82 1503.09   0.90    95.89    35.81  92.21   5.59
    vda              0.00    0.00      0.04      0.00     0.00     0.00   2.73   0.00    3.08    0.00   0.00    62.55     0.00   2.20   0.00

    Device            r/s     w/s     rkB/s     wkB/s   rrqm/s   wrqm/s  %rrqm  %wrqm r_await w_await aqu-sz rareq-sz wareq-sz  svctm  %util

    Device            r/s     w/s     rkB/s     wkB/s   rrqm/s   wrqm/s  %rrqm  %wrqm r_await w_await aqu-sz rareq-sz wareq-sz  svctm  %util

    Device            r/s     w/s     rkB/s     wkB/s   rrqm/s   wrqm/s  %rrqm  %wrqm r_await w_await aqu-sz rareq-sz wareq-sz  svctm  %util

    Device            r/s     w/s     rkB/s     wkB/s   rrqm/s   wrqm/s  %rrqm  %wrqm r_await w_await aqu-sz rareq-sz wareq-sz  svctm  %util
EOF
    ;;
    "-dxyz")
    cat <<EOF
    Linux 5.10.40-1rodete2-amd64 (muutuhm.c.googlers.com)   08/11/2021      _x86_64_        (16 CPU)


    Device            r/s      w/s    dkB/s   drqm/s  %drqm d_await dareq-sz     f/s f_await  aqu-sz  %util
    dm-0             0.00  17838.00     0.00     0.00   0.00    0.00     0.00    0.00    0.00    1.08 100.00
    sda              0.00  47420.00     0.00     0.00   0.00    0.00     0.00 31004.00    0.02    2.94 100.00


    Device            r/s      w/s    dkB/s   drqm/s  %drqm d_await dareq-sz     f/s f_await  aqu-sz  %util
    dm-0             3.00  16308.00     0.00     0.00   0.00    0.00     0.00    0.00    0.00    1.93 100.00
    sda              3.00  43083.00     0.00     0.00   0.00    0.00     0.00 27551.00    0.03    3.33 100.00


    Device            r/s      w/s    dkB/s   drqm/s  %drqm d_await dareq-sz     f/s f_await  aqu-sz  %util
    dm-0             1.00  19197.00     0.00     0.00   0.00    0.00     0.00    0.00    0.00    0.88 100.00
    sda              1.00  51192.00     0.00     0.00   0.00    0.00     0.00 33485.00    0.02    2.86 100.00
EOF
    ;;
    esac
}

main "$@"