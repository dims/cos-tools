package config

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
	"cos.googlesource.com/cos/tools.git/src/cmd/cos_gpu_driver_builder/internal/builder"
	"cos.googlesource.com/cos/tools.git/src/pkg/gcs"
	"cos.googlesource.com/cos/tools.git/src/pkg/gpuconfig"
)

func outputDriverFile(config gpuconfig.GPUPrecompilationConfig) string {
	driverRunfile := fmt.Sprintf("NVIDIA-Linux-x86_64-%s-custom.run", config.DriverVersion)
	return fmt.Sprintf("%s/%s", config.ProtoConfig.GetDriverOutputGcsDir(), driverRunfile)
}

func ProcessConfigs(ctx context.Context, client *storage.Client, configs []gpuconfig.GPUPrecompilationConfig, dryRun bool) error {
	for _, config := range configs {
		log.Printf("building precompiled GPU driver for %s:%s, driver version %s\n", config.VersionType, config.Version, config.DriverVersion)
		if processed, _ := gcs.GCSObjectExists(ctx, client, outputDriverFile(config)); processed {
			continue
		}
		dir, precompiledDriver, err := builder.BuildPrecompiledDriver(ctx, client, config)
		defer os.RemoveAll(dir)
		if err != nil {
			log.Printf("precompilation failed for: %s, driver version %s: %v\n", config.Version, config.DriverVersion, err)
			continue
		}
		outputURL, err := url.Parse(config.ProtoConfig.GetDriverOutputGcsDir())
		if err != nil {
			log.Printf("failed to parse driver output gcs dir: %v\n", err)
		}
		outputURL.Path = filepath.Join(outputURL.Path, precompiledDriver)
		outputDriverFile := outputURL.String()
		if !dryRun {

			if err := gcs.UploadGCSObject(ctx, client, filepath.Join(dir, precompiledDriver), outputDriverFile); err != nil {
				log.Printf("export failed for: %s, driver version %s: %v\n", config.Version, config.DriverVersion, err)
				continue
			}
		}
	}
	return nil
}
