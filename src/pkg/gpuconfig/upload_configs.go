package gpuconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"cos.googlesource.com/cos/tools.git/src/pkg/gcs"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
)

func destDir(gcsBucket string) string {
	timestamp := strings.TrimSuffix(time.Now().Format(time.RFC3339), "Z")
	uid := uuid.NewString()[:8]
	return fmt.Sprintf("gs://%s/%s", gcsBucket, timestamp+"-"+uid)
}

func UploadConfigs(ctx context.Context, client *storage.Client, configs []GPUPrecompilationConfig, gcsBucket string) error {
	for _, config := range configs {
		log.Printf("uploading gpu precompilation config for: %s, driver version %s\n", config.Version, config.DriverVersion)
		destDir := destDir(gcsBucket)
		if err := gcs.UploadGCSObjectString(ctx, client, proto.MarshalTextString(config.ProtoConfig), fmt.Sprintf("%s/%s", destDir, "config.textproto")); err != nil {
			return err
		}
		metadata, _ := json.MarshalIndent(config, "", "    ")
		if err := gcs.UploadGCSObjectString(ctx, client, string(metadata), fmt.Sprintf("%s/%s", destDir, "metadata")); err != nil {
			return err
		}
	}
	return nil
}
