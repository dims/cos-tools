package gpuconfig

import (
	"context"
	"log"
	"testing"
	"time"

	"cos.googlesource.com/cos/tools.git/src/pkg/fakes"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestReadConfig(t *testing.T) {
	ctx := context.Background()
	gcs := fakes.GCSForTest(t)
	defer gcs.Close()
	gcs.Objects = map[string][]byte{
		"/cos-gpu-configs-test/2022-10-07T01:36:07-4ed7213e/config.textproto": testConfigFileContents,
		"/cos-gpu-configs-test/2022-10-07T01:36:07-4ed7213e/metadata":         testMetadataContents,
	}
	want := testConfig
	got, err := ReadConfig(ctx, gcs.Client, "gs://cos-gpu-configs-test/2022-10-07T01:36:07-4ed7213e/")
	if err != nil {
		log.Fatalf("ReadConfig() failed:%v\n", err)
	}

	if diff := cmp.Diff(got, want, protocmp.Transform()); diff != "" {
		t.Errorf("ReadConfig() returned unexpected difference (-want, got):\n%s", diff)
	}
}

func TestReadConfigs(t *testing.T) {
	ctx := context.Background()
	gcs := fakes.GCSForTest(t)
	defer gcs.Close()
	gcs.Objects = map[string][]byte{
		"/cos-gpu-configs-test/2022-10-05T05:51:44-0bf111fe/config.textproto": []byte(""), // file not read
		"/cos-gpu-configs-test/2022-10-05T05:51:44-0bf111fe/metadata":         []byte(""), // file not read
		"/cos-gpu-configs-test/2022-10-06T13:46:00-269200f5/config.textproto": []byte(""), // file not read
		"/cos-gpu-configs-test/2022-10-06T13:46:00-269200f5/metadata":         []byte(""), // file not read
		"/cos-gpu-configs-test/2022-10-07T01:29:43-e9b4b850/config.textproto": testConfigFileContents,
		"/cos-gpu-configs-test/2022-10-07T01:29:43-e9b4b850/metadata":         testMetadataContents,
	}

	timeNow = func() time.Time { return time.Date(2022, time.October, 10, 0, 0, 0, 0, time.UTC) }

	// read configs from [2022-10-10 to 2022-10-10 minus 3 days]
	got, err := ReadConfigs(ctx, gcs.Client, "cos-gpu-configs-test", 3, "kernel")
	if err != nil {
		log.Fatalf("ReadConfigs() failed:%v\n", err)
	}

	if diff := cmp.Diff(got, []GPUPrecompilationConfig{testConfig}, protocmp.Transform()); diff != "" {
		t.Errorf("ReadConfigs() returned unexpected difference (-want, got):\n%s", diff)
	}
}
