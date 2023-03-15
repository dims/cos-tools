package main

import (
	"context"
	"flag"

	log "github.com/golang/glog"

	"cloud.google.com/go/storage"

	"cos.googlesource.com/cos/tools.git/src/cmd/cos_gpu_driver_builder/internal/config"
	"cos.googlesource.com/cos/tools.git/src/pkg/gpuconfig"
)

var (
	configDir = flag.String("config-dir", "", "Directory containing config.textproto and metadata file that needs to be processed.")
	bucket    = flag.String("watcher-gcs", "", "GCS bucket to watch for unprocessed configs.")
	lookBack  = flag.Int("lookBackDays", 7, "read configs produced within the past <lookBack> days.")
	// default to only building image CI precompiled drivers
	mode = flag.String("mode", "image", "image, kernel, or both for processing image CI/kernel CI configs. Works only with watcher-gcs arg")
)

func main() {
	flag.Parse()

	if *bucket == "" && *configDir == "" {
		log.Fatal("empty watcher gcs dir and config file dir")
	}

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal("failed to setup client for GCS:", err)
	}

	var configs []gpuconfig.GPUPrecompilationConfig
	if *bucket != "" { // cos_gpu_driver_builder --watcher-gcs="cos-gpu-configs"
		configs, err = gpuconfig.ReadConfigs(ctx, client, *bucket, *lookBack, *mode)
		if err != nil {
			log.Fatal("could not read configs:", err)
		}
	} else { // cos_gpu_driver_builder --config="gs://cos-gpu-configs/2022-09-26T16:03:17-dc65ba40/"
		config, err := gpuconfig.ReadConfig(ctx, client, *configDir)
		if err != nil {
			log.Fatal("could not read config:", err)
		}
		configs = append(configs, config)
	}

	config.ProcessConfigs(ctx, client, configs)
}
