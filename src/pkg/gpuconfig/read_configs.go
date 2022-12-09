package gpuconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"cos.googlesource.com/cos/tools.git/src/pkg/gcs"
	"cos.googlesource.com/cos/tools.git/src/pkg/gpuconfig/pb"
	"github.com/golang/protobuf/proto"
	"google.golang.org/api/iterator"
)

func listConfigDirs(ctx context.Context, client *storage.Client, bucketName string, start string) ([]string, error) {
	query := &storage.Query{
		StartOffset: start, // Only list objects lexicographically >=
		Delimiter:   "/",   // Only list dirs
	}
	query.SetAttrSelection([]string{"Prefix"})

	bkt := client.Bucket(bucketName)
	var dirNames []string
	it := bkt.Objects(ctx, query)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		dirNames = append(dirNames, fmt.Sprintf("gs://%s/%s", bucketName, attrs.Prefix))
	}

	return dirNames, nil
}

// Reads precompilation config from GCS bucket into GPUPrecompilationConfig struct.
func ReadConfig(ctx context.Context, client *storage.Client, dirName string) (GPUPrecompilationConfig, error) {
	var config GPUPrecompilationConfig
	if len(dirName) > 1 && dirName[len(dirName)-1] != '/' {
		dirName += "/"
	}
	metadata, err := gcs.DownloadGCSObjectString(ctx, client, dirName+"metadata")
	if err != nil {
		return config, err
	}

	if err := json.Unmarshal([]byte(metadata), &config); err != nil {
		return config, err
	}

	textproto, err := gcs.DownloadGCSObjectString(ctx, client, dirName+"config.textproto")
	if err != nil {
		return config, err
	}

	config.ProtoConfig = &pb.COSGPUBuildRequest{}
	if err := proto.UnmarshalText(textproto, config.ProtoConfig); err != nil {
		return config, err
	}

	return config, nil
}

// Reads all config dirs published within <lookBackDays> of current date into a list of GPUPrecompilationConfig struct
func ReadConfigs(ctx context.Context, client *storage.Client, bucketName string, lookBackDays int, versionType string) ([]GPUPrecompilationConfig, error) {
	startDay := strings.TrimSuffix(timeNow().AddDate(0, 0, -lookBackDays).Format(time.RFC3339), "Z")
	dirNames, err := listConfigDirs(ctx, client, bucketName, startDay)
	if err != nil {
		return nil, err
	}

	configs := []GPUPrecompilationConfig{}
	for _, dir := range dirNames {
		config, err := ReadConfig(ctx, client, dir)
		if err != nil {
			return nil, err
		}
		if matchVersionType(versionType, config.VersionType) {
			configs = append(configs, config)
		}
	}
	return configs, nil
}

func matchVersionType(mode string, versionType string) bool {
	if strings.EqualFold(mode, "both") {
		return true
	}
	return strings.EqualFold(mode, versionType)
}
