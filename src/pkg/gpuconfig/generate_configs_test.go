package gpuconfig

import (
	"context"
	"testing"

	"cos.googlesource.com/cos/tools.git/src/pkg/fakes"
	"cos.googlesource.com/cos/tools.git/src/pkg/gpuconfig/pb"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

const toolchainTarballPath = "gs://chromiumos-sdk/2021/06/x86_64-cros-linux-gnu-2021.06.26.094653.tar.xz"

var testGCSObjects = map[string][]byte{
	"/cos-kernel-artifacts/builds/5.15.55-34.m101/toolchain_url.x86_64": []byte(toolchainTarballPath),
}

func TestGenerateKernelCIConfigs(t *testing.T) {
	gcs := fakes.GCSForTest(t)
	defer gcs.Close()
	gcs.Objects = testGCSObjects
	client := gcs.Client
	for _, tc := range []struct {
		kernelVersion  string
		driverVersions []string
		expected       []GPUPrecompilationConfig
	}{
		{
			"5.15.55-34.m101",
			[]string{"470.82.01"},
			[]GPUPrecompilationConfig{
				GPUPrecompilationConfig{
					ProtoConfig: &pb.COSGPUBuildRequest{
						KernelSrcTarballGcs:     stringPtr("gs://cos-kernel-artifacts/builds/5.15.55-34.m101/cos-kernel-src-5.15.55-34.m101.tgz"),
						KernelHeadersTarballGcs: stringPtr("gs://cos-kernel-artifacts/builds/5.15.55-34.m101/cos-kernel-headers-5.15.55-34.m101-x86_64.tgz"),
						NvidiaRunfileAddress:    stringPtr("https://us.download.nvidia.com/tesla/470.82.01/NVIDIA-Linux-x86_64-470.82.01.run"),
						ToolchainTarballGcs:     stringPtr("gs://chromiumos-sdk/2021/06/x86_64-cros-linux-gnu-2021.06.26.094653.tar.xz"),
						ToolchainEnvGcs:         stringPtr("gs://cos-kernel-artifacts/builds/5.15.55-34.m101/toolchain_env.x86_64"),
						DriverOutputGcsDir:      stringPtr("gs://nvidia-drivers-us-public/nvidia-cos-project/5.15.55-34.m101/"),
					},
					DriverVersion: "470.82.01",
					Milestone:     "101",
					Version:       "5.15.55-34.m101",
					VersionType:   "Kernel",
				},
			},
		},
	} {
		ctx := context.Background()
		got, err := GenerateKernelCIConfigs(ctx, client, tc.kernelVersion, tc.driverVersions)
		if err != nil {
			t.Fatalf("GenerateKernelCIConfig() failed: %s", err)
		}
		if diff := cmp.Diff(got, tc.expected, protocmp.Transform()); diff != "" {
			t.Errorf("GenerateKernelCIConfig() returned unexpected difference (-want +got):\n%s", diff)
		}
	}
}

func TestKernelVersionToMilestone(t *testing.T) {
	for _, tc := range []struct {
		kernelVersion     string
		milestoneExpected string
	}{
		{"5.10.100-14.m97", "97"},
		{"5.10.107-10.r97", "97"},
		{"5.10.100-14", ""},
	} {
		if got := kernelVersionToMilestone(tc.kernelVersion); got != tc.milestoneExpected {
			t.Errorf("kernelVersionToMilestone() = %+v, want %+v", got, tc.milestoneExpected)
		}
	}
}

func TestFetchToolchainTarballPath(t *testing.T) {
	gcs := fakes.GCSForTest(t)
	defer gcs.Close()
	gcs.Objects = testGCSObjects
	client := gcs.Client
	kernelVersion := "5.15.55-34.m101"
	got, err := fetchToolchainTarballPath(context.Background(), client, kernelVersion)
	if err != nil {
		t.Fatalf("fetchToolchainTarballPath() failed: %s", err)
	}
	if got != toolchainTarballPath {
		t.Errorf("fetchToolchainTarballPath() = %+v, want %+v", got, toolchainTarballPath)
	}
}
