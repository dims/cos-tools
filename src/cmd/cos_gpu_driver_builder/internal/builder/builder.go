package builder

import (
	"context"

	"cloud.google.com/go/storage"
	"cos.googlesource.com/cos/tools.git/src/pkg/gpuconfig"
)

func BuildPrecompiledDriver(ctx context.Context, client *storage.Client, config gpuconfig.GPUPrecompilationConfig) (string, error) {
	return "", nil
}
