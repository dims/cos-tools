package main

import (
	"context"
	"flag"
	"log"

	"cloud.google.com/go/storage"
	"cos.googlesource.com/cos/tools.git/src/cmd/cos_gpu_driver_builder/builder"
	"cos.googlesource.com/cos/tools.git/src/pkg/gpuconfig"
)

var (
	bucket = flag.String("watcher-gcs", "", "GCS bucket to watch for unprocessed configs.")
)

func main() {
	flag.Parse()

	if *bucket == "" {
		log.Fatal("empty watcher gcs dir")
	}

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal("failed to setup client for GCS: %v", err)
	}

	configs, err := gpuconfig.ReadConfigs(ctx, client, *bucket, 7)
	if err != nil {
		log.Fatal("could not read configs: %v", err)
	}

	_ = builder.BuildPrecompiledDrivers(ctx, client, configs)
}
