package config

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/storage"
	"cos.googlesource.com/cos/tools.git/src/cmd/cos_gpu_driver_builder/internal/builder"
	"cos.googlesource.com/cos/tools.git/src/pkg/gcs"
	"cos.googlesource.com/cos/tools.git/src/pkg/gpuconfig"
)

func outputDriverFile(config gpuconfig.GPUPrecompilationConfig) string {
	driverRunfile := fmt.Sprintf("NVIDIA-Linux-x86_64-%s-custom.run", config.DriverVersion)
	return fmt.Sprintf("%s/%s", config.ProtoConfig.GetDriverOutputGcsDir(), driverRunfile)
}

func ProcessConfigs(ctx context.Context, client *storage.Client, configs []gpuconfig.GPUPrecompilationConfig) error {
	for _, config := range configs {
		log.Printf("building precompiled GPU driver for: %s, driver version %s\n", config.Version, config.DriverVersion)
		if processed, _ := gcs.GCSObjectExists(ctx, client, outputDriverFile(config)); processed {
			continue
		}

		precompiledDriver, err := builder.BuildPrecompiledDriver(ctx, client, config)
		if err != nil {
			log.Printf("precompilation failed for: %s, driver version %s\n", config.Version, config.DriverVersion)
			continue
		}

		if err := gcs.UploadGCSObject(ctx, client, precompiledDriver, outputDriverFile(config)); err != nil {
			log.Printf("export failed for: %s, driver version %s\n", config.Version, config.DriverVersion)
			continue
		}
	}
	return nil
}
