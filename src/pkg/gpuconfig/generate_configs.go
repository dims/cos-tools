// Package gpuconfig implements routines for manipulating proto based
// GPU build configuration files.
//
// It also implements the construction of these configs for
// the COS Image and the COS Kernel CI.
package gpuconfig

//go:generate protoc --go_out=:./pb -I. proto/config.proto

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"cloud.google.com/go/storage"
	"cos.googlesource.com/cos/tools.git/src/pkg/gpuconfig/pb"
)

const (
	kernelGCSPrefix                  string = "gs://cos-kernel-artifacts/builds"
	kernelSrcTarballPathTemplate     string = "%s/%[2]s/cos-kernel-src-%[2]s.tgz"
	kernelHeadersTarballPathTemplate string = "%s/%[2]s/cos-kernel-headers-%[2]s-x86_64.tgz"
	toolchainTarballPathTemplate     string = "builds/%s/toolchain_url.x86_64"
	toolchainEnvPathTemplate         string = "%s/%s/toolchain_env.x86_64"
	driverOutputGcsDirTemplate       string = "gs://nvidia-drivers-us-public/nvidia-cos-project/%s/"
	nvidiaRunfileAddressTemplate     string = "https://us.download.nvidia.com/tesla/%[1]s/NVIDIA-Linux-x86_64-%[1]s.run"
	timeFormatTemplate               string = "2006-01-02-15:04:05"
)

type GPUPrecompilationConfig struct {
	ProtoConfig   *pb.COSGPUBuildRequest `json:"-"`
	DriverVersion string                 `json:"driver_version"`
	Milestone     string                 `json:"milestone"`
	Version       string                 `json:"version"`
	VersionType   string                 `json:"version_type"`
}

func kernelVersionToMilestone(kernelVersion string) string {
	milestone := ""
	for _, sep := range []string{"m", "r"} { // release branch or main branch check
		if split := strings.Split(kernelVersion, sep); len(split) == 2 {
			milestone = split[1]
			break
		}
	}
	return milestone
}

// Generates and GPU precompilation build configs(and metadata) for a given
// tuple of kernelVersion and driver versions
func GenerateKernelCIConfigs(ctx context.Context, client *storage.Client, kernelVersion string, driverVersions []string) ([]GPUPrecompilationConfig, error) {
	configs := []GPUPrecompilationConfig{}
	for _, driverVersion := range driverVersions {
		config, err := constructKernelCIConfig(ctx, client, kernelVersion, driverVersion)
		if err != nil {
			return nil, err
		}
		milestone := kernelVersionToMilestone(kernelVersion)
		configs = append(configs, GPUPrecompilationConfig{config, driverVersion, milestone, kernelVersion, "Kernel"})
	}
	return configs, nil
}

func constructKernelCIConfig(ctx context.Context, client *storage.Client, kernelVersion, driverVersion string) (*pb.COSGPUBuildRequest, error) {
	config := pb.COSGPUBuildRequest{
		KernelSrcTarballGcs:     stringPtr(fmt.Sprintf(kernelSrcTarballPathTemplate, kernelGCSPrefix, kernelVersion)),
		KernelHeadersTarballGcs: stringPtr(fmt.Sprintf(kernelHeadersTarballPathTemplate, kernelGCSPrefix, kernelVersion)),
		NvidiaRunfileAddress:    stringPtr(fmt.Sprintf(nvidiaRunfileAddressTemplate, driverVersion)),
		ToolchainEnvGcs:         stringPtr(fmt.Sprintf(toolchainEnvPathTemplate, kernelGCSPrefix, kernelVersion)),
		DriverOutputGcsDir:      stringPtr(fmt.Sprintf(driverOutputGcsDirTemplate, kernelVersion)),
	}

	toolchainTarballPath, err := fetchToolchainTarballPath(ctx, client, kernelVersion)
	if err != nil {
		return nil, err
	}
	config.ToolchainTarballGcs = &toolchainTarballPath

	return &config, nil
}

func fetchToolchainTarballPath(ctx context.Context, client *storage.Client, kernelVersion string) (string, error) {
	toolchainTarballPathURL := fmt.Sprintf(toolchainTarballPathTemplate, kernelVersion)
	reader, err := client.Bucket("cos-kernel-artifacts").Object(toolchainTarballPathURL).NewReader(ctx)
	if err != nil {
		return "", fmt.Errorf("Could not fetch the toolchain tarball path: %w", err)
	}
	var toolchainTarballPath []byte
	if toolchainTarballPath, err = ioutil.ReadAll(reader); err != nil {
		return "", fmt.Errorf("Could not read file contents of toolchain tarball path: %w", err)
	}
	return string(toolchainTarballPath), nil
}

// stringPtr returns a pointer to a string.
func stringPtr(s string) *string {
	return &s
}
