package test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"testing"
	"text/template"

	"cloud.google.com/go/storage"
	pb_version "cos.googlesource.com/cos/tools.git/src/cmd/cos_gpu_installer/versions"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/encoding/prototext"
)

const (
	protoConfigPath       = "../versions/config/versions.textproto"
	driverPublicGcsBucket = "nvidia-drivers-us-public"
)

// Test to check whether precompiled drivers exist in public GCS bucket.
//
// Note: The test uses Application Default Credentials for authentication.
//       If not already done, install the gcloud CLI from
//       https://cloud.google.com/sdk/ and run
//       `gcloud auth application-default login`. For more information, see
//       https://developers.google.com/identity/protocols/application-default-credentials
func TestCheckDrivers(t *testing.T) {
	ctx := context.Background()

	c, err := google.DefaultClient(ctx, compute.CloudPlatformScope)
	if err != nil {
		log.Fatal(err)
	}

	computeService, err := compute.New(c)
	if err != nil {
		log.Fatal(err)
	}

	storageClient, err := storage.NewClient(ctx, option.WithoutAuthentication())
	if err != nil {
		log.Fatal(err)
	}
	defer storageClient.Close()

	versionsMap, err := readVersionMap()
	if err != nil {
		log.Fatal(err)
	}

	for _, entry := range versionsMap.GetEntry() {
		testCase := fmt.Sprintf("[cos=%s,driver=%s]", entry.GetCosImageFamily(), entry.GetGpuDriverVersion())
		t.Run(testCase, testCheckDriver(entry.GetCosImageFamily(), entry.GetGpuDriverVersion(), computeService, storageClient, ctx))
	}
}

// Reads GpuVersionMap from protocal buffer.
// The definition and data of protobuf should be found at cos.googlesource.com/cos/tools.git/src/cmd/cos_gpu_installer/versions
func readVersionMap() (*pb_version.GpuVersionMap, error) {
	configPath, err := filepath.Abs(protoConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get abspath of proto config file: %v", err)
	}

	versionMap := &pb_version.GpuVersionMap{}
	in, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read proto config file: %v", err)
	}
	if err := prototext.Unmarshal(in, versionMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal proto config: %v", err)
	}
	return versionMap, nil
}

// Gets the COS image name from a COS image family.
func getImageFromFamily(imageFamily string, computeService *compute.Service, ctx context.Context) (string, error) {
	resp, err := computeService.Images.GetFromFamily("cos-cloud", imageFamily).Context(ctx).Do()
	if err != nil {
		return "", err
	}
	return resp.Name, nil
}

// Checks whether a given GCS object exisit in the given GCS bucket.
func gcsObjectExist(bucket string, object string, storageClient *storage.Client, ctx context.Context) (bool, error) {
	_, err := storageClient.Bucket(bucket).Object(object).Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get GCS object %s from bukect %s: %v", object, bucket, err)
	}
	return true, nil
}

// Composes the GCS path of a Nvidia precompiled drivers based on COS image name and GPU driver version.
func getPrecompiledDriverGcsPath(cosImage string, gpuDriverVersion string) (string, error) {
	const temp = `nvidia-cos-project/{{.milestone}}/tesla/{{.driverBranch}}_00/{{.driverVersion}}/NVIDIA-Linux-x86_64-{{.driverVersion}}_{{.cosVersion}}.cos`

	re, err := regexp.Compile(`^cos-(dev-|beta-|stable-)?([\d]+)-([\d-]+)$`)
	if err != nil {
		return "", fmt.Errorf("failed to compile regular expression: %v", err)
	}
	if !re.MatchString(cosImage) {
		return "", fmt.Errorf("failed to parse COS image name %s", cosImage)
	}
	cosVersion := re.FindStringSubmatch(cosImage)

	re, err = regexp.Compile(`^([\d]+)\.[\d\.]+$`)
	if err != nil {
		return "", fmt.Errorf("failed to compile regular expression: %v", err)
	}
	if !re.MatchString(gpuDriverVersion) {
		return "", fmt.Errorf("failed to parse GPU driver version %s", gpuDriverVersion)
	}
	driverBranch := re.FindStringSubmatch(gpuDriverVersion)[1]

	m := map[string]string{
		"milestone":     cosVersion[2],
		"cosVersion":    cosVersion[2] + "-" + cosVersion[3],
		"driverBranch":  driverBranch,
		"driverVersion": gpuDriverVersion,
	}
	var buffer bytes.Buffer
	if err := template.Must(template.New("").Parse(temp)).Execute(&buffer, m); err != nil {
		return "", fmt.Errorf("failed to generate GCS object path from template: %v", err)
	}
	return buffer.String(), nil
}

// Testcase to check whether the precompiled driver of a [cosImageFamily, gpuDriverVersion] combination exists.
func testCheckDriver(cosImageFamily string, gpuDriverVersion string, computeService *compute.Service, storageClient *storage.Client, ctx context.Context) func(*testing.T) {
	return func(t *testing.T) {
		imageName, err := getImageFromFamily(cosImageFamily, computeService, ctx)
		if err != nil {
			t.Errorf("failed to get image from image family %s: %v", cosImageFamily, err)
		}
		driverObject, err := getPrecompiledDriverGcsPath(imageName, gpuDriverVersion)
		if err != nil {
			t.Errorf("failed to get GCS path of precompiled driver [image=%s,driver=%s]: %v", imageName, gpuDriverVersion, err)
		}

		exist, err := gcsObjectExist(driverPublicGcsBucket, driverObject, storageClient, ctx)
		if err != nil {
			t.Errorf("failed to check existence: %v", err)
		}
		if !exist {
			t.Errorf("Precompiled drivers gs://%s/%s doesn't exist", driverPublicGcsBucket, driverObject)
		}
	}
}
