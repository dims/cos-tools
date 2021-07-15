#!/bin/bash

main() {
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
}

main "$#"