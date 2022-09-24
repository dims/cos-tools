package gpuconfig

import (
	"context"
	"log"
	"strings"
	"testing"

	"cos.googlesource.com/cos/tools.git/src/pkg/fakes"
	"cos.googlesource.com/cos/tools.git/src/pkg/gpuconfig/pb"
	"github.com/google/go-cmp/cmp"
)

var (
	testConfigFileContents = []byte("kernel_src_tarball_gcs: \"gs://cos-kernel-artifacts/builds/5.10.133-43.r97/cos-kernel-src-5.10.133-43.r97.tgz\"\nkernel_headers_tarball_gcs: \"gs://cos-kernel-artifacts/builds/5.10.133-43.r97/cos-kernel-headers-5.10.133-43.r97-x86_64.tgz\"\nnvidia_runfile_address: \"https://us.download.nvidia.com/tesla/510.47.03/NVIDIA-Linux-x86_64-510.47.03.run\"\ntoolchain_tarball_gcs: \"gs://chromiumos-sdk/2021/06/x86_64-cros-linux-gnu-2021.06.26.094653.tar.xz\"\ntoolchain_env_gcs: \"gs://cos-kernel-artifacts/builds/5.10.133-43.r97/toolchain_env.x86_64\"\ndriver_output_gcs_dir: \"gs://nvidia-drivers-us-public/nvidia-cos-project/5.10.133-43.r97/\"\n")
	testMetadataContents   = []byte("{\n    \"driver_version\": \"510.47.03\",\n    \"milestone\": \"97\",\n    \"version\": \"5.10.133-43.r97\",\n    \"version_type\": \"Kernel\"\n}")
	testConfig             = GPUPrecompilationConfig{
		ProtoConfig: &pb.COSGPUBuildRequest{
			KernelSrcTarballGcs:     stringPtr("gs://cos-kernel-artifacts/builds/5.10.133-43.r97/cos-kernel-src-5.10.133-43.r97.tgz"),
			KernelHeadersTarballGcs: stringPtr("gs://cos-kernel-artifacts/builds/5.10.133-43.r97/cos-kernel-headers-5.10.133-43.r97-x86_64.tgz"),
			NvidiaRunfileAddress:    stringPtr("https://us.download.nvidia.com/tesla/510.47.03/NVIDIA-Linux-x86_64-510.47.03.run"),
			ToolchainTarballGcs:     stringPtr("gs://chromiumos-sdk/2021/06/x86_64-cros-linux-gnu-2021.06.26.094653.tar.xz"),
			ToolchainEnvGcs:         stringPtr("gs://cos-kernel-artifacts/builds/5.10.133-43.r97/toolchain_env.x86_64"),
			DriverOutputGcsDir:      stringPtr("gs://nvidia-drivers-us-public/nvidia-cos-project/5.10.133-43.r97/"),
		},
		DriverVersion: "510.47.03",
		Milestone:     "97",
		Version:       "5.10.133-43.r97",
		VersionType:   "Kernel",
	}
)

func TestUploadConfig(t *testing.T) {
	ctx := context.Background()
	gcs := fakes.GCSForTest(t)
	defer gcs.Close()
	err := UploadConfigs(ctx, gcs.Client, []GPUPrecompilationConfig{testConfig}, "cos-gpu-configs-test")
	if err != nil {
		log.Fatalf("UploadConfig() failed:%v\n", err)
	}

	// verify contents of files uploaded
	for filename, content := range gcs.Objects {
		if strings.Contains(filename, "metadata") {
			if !cmp.Equal(content, testMetadataContents) {
				t.Errorf("bucket 'cos-gpu-configs-test', object has %s; want %s\n", content, testMetadataContents)
			}
		} else if strings.Contains(filename, "config.textproto") {
			if !cmp.Equal(content, testConfigFileContents) {
				t.Errorf("bucket 'cos-gpu-configs-test', object has %s; want %s\n", content, testConfigFileContents)
			}
		} else {
			t.Errorf("bucket 'cos-gpu-configs-test' has unknown object %s with data %s\n", filename, content)
		}
	}
}
