package builder

import (
	"context"
	"log"

	"cloud.google.com/go/storage"
	"cos.googlesource.com/cos/tools.git/src/pkg/gpuconfig"
)

func BuildPrecompiledDrivers(ctx context.Context, client *storage.Client, configs []gpuconfig.GPUPrecompilationConfig) error {
	for _, config := range configs {
		log.Printf("building precompiled GPU driver for: %s, driver version %s\n", config.Version, config.DriverVersion)
	}
	return nil
}
