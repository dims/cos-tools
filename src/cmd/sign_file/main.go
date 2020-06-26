package main

import (
	"flag"
	"fmt"
	"os"

	log "github.com/golang/glog"

	"pkg/modules"
)

var (
	rawSigPath = flag.String("rawsig", "", "Path of the raw signature to append to the kernel module. Required.")
	modulePath = flag.String("module", "", "Path of the kernel module needs to be signed. Required.")
	outPath    = flag.String("outpath", "", "Path of the signed module output destination. The default is to append signature in-place.")
)

func main() {
	flag.Parse()
	if err := checkFlags(); err != nil {
		log.Errorf("failed to parse flags: %v", err)
		os.Exit(1)
	}
	if err := modules.AppendSignature(*outPath, *modulePath, *rawSigPath); err != nil {
		log.Errorf("failed to append signature: %v", err)
		os.Exit(1)
	}
}

func checkFlags() error {
	if *rawSigPath == "" {
		return fmt.Errorf("flag -rawsig is required")
	}
	if *modulePath == "" {
		return fmt.Errorf("flag -module is required")
	}
	if *outPath == "" {
		outPath = modulePath
	}
	return nil
}
