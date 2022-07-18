package main

import (
	"context"
	"flag"
	"log"
	"strings"

	"cloud.google.com/go/storage"
	"cos.googlesource.com/cos/tools.git/src/pkg/gpuconfig"
)

var (
	bucket        = flag.String("gcs-bucket", "cos-gpu-configs", "GCS bucket to upload GPU configs to.")
	kernelVersion = flag.String("kernel-version", "", "Kernel version for COS GPU precompilation build request, example: 5.10.105-23.m97")

	driverVersions = flag.String("driver-versions", "", "Driver version/ (Comma separated if multiple driver versions) for COS GPU precompilation build request, example 450.119.04 / 450.119.04,470.150.03")
)

func main() {
	flag.Parse()

	if *kernelVersion == "" || *driverVersions == "" {
		log.Fatal("empty kernel version: %s or driver version:%s specified", kernelVersion, driverVersions)
	}

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal("failed to setup client for GCS: %v", err)
	}

	configs, err := gpuconfig.GenerateKernelCIConfigs(ctx, client, *kernelVersion, strings.Split(*driverVersions, ","))
	if err != nil {
		log.Fatal("gpu config generation failed: %v", err)
	}

	if err := gpuconfig.UploadConfigs(ctx, client, configs, *bucket); err != nil {
		log.Fatal("uploading gpu config failed: %v", err)
	}
}
