package cos

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestEnvReader(t *testing.T) {
	testDir, err := ioutil.TempDir("", "testing")
	if err != nil {
		t.Fatalf("Failed to create tempdir: %v", err)
	}
	defer os.RemoveAll(testDir)

	osReleaseString := `BUILD_ID=12688.0.0
NAME="Container-Optimized OS"
KERNEL_COMMIT_ID=5d8615d1e135275cbfdf9522517a3b198e7199ee
GOOGLE_CRASH_ID=Lakitu
VERSION_ID=80
BUG_REPORT_URL="https://cloud.google.com/container-optimized-os/docs/resources/support-policy#contact_us"
PRETTY_NAME="Container-Optimized OS from Google"
VERSION=80
GOOGLE_METRICS_PRODUCT_ID=26
HOME_URL="https://cloud.google.com/container-optimized-os/docs"
ID=cos`
	if err := createConfigFile(osReleaseString, osReleasePath, testDir); err != nil {
		t.Fatalf("Failed to create osRelease file: %v", err)
	}
	toolchainPathString := `2019/11/x86_64-cros-linux-gnu-2019.11.16.041937.tar.xz`
	if err := createConfigFile(toolchainPathString, toolchainPathFile, testDir); err != nil {
		t.Fatalf("Failed to create toolchain path file: %v", err)
	}

	envReader, err := NewEnvReader(testDir)
	if err != nil {
		t.Fatalf("Failed to create EnvReader: %v", err)
	}

	for _, tc := range []struct {
		testName string
		got      interface{}
		expect   interface{}
	}{
		{
			"OsRelease",
			envReader.OsRelease(),
			map[string]string{
				"BUILD_ID":                  "12688.0.0",
				"NAME":                      "Container-Optimized OS",
				"KERNEL_COMMIT_ID":          "5d8615d1e135275cbfdf9522517a3b198e7199ee",
				"GOOGLE_CRASH_ID":           "Lakitu",
				"VERSION_ID":                "80",
				"BUG_REPORT_URL":            "https://cloud.google.com/container-optimized-os/docs/resources/support-policy#contact_us",
				"PRETTY_NAME":               "Container-Optimized OS from Google",
				"VERSION":                   "80",
				"GOOGLE_METRICS_PRODUCT_ID": "26",
				"HOME_URL":                  "https://cloud.google.com/container-optimized-os/docs",
				"ID":                        "cos",
			},
		},
		{"BuildNumber", envReader.BuildNumber(), "12688.0.0"},
		{"Milestone", envReader.Milestone(), "80"},
		{"Milestone", envReader.KernelCommit(), "5d8615d1e135275cbfdf9522517a3b198e7199ee"},
		{"ToolchainPath", envReader.ToolchainPath(), "2019/11/x86_64-cros-linux-gnu-2019.11.16.041937.tar.xz"},
	} {
		if !reflect.DeepEqual(tc.expect, tc.got) {
			t.Errorf("Unexpected %s,\nwant: %v\n got: %v", tc.testName, tc.testName, tc.expect)
		}
	}
}

func createConfigFile(configStr, configFileName, testDir string) error {
	path := filepath.Join(testDir, configFileName)
	if err := os.MkdirAll(filepath.Dir(path), 0744); err != nil {
		return fmt.Errorf("Failed to create dir: %v", err)
	}
	configFile, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("Failed to create file: %v", err)
	}
	defer configFile.Close()

	if _, err = configFile.WriteString(configStr); err != nil {
		return fmt.Errorf("Failed to write to file %s: %v", configFile.Name(), err)
	}
	if err = configFile.Close(); err != nil {
		return fmt.Errorf("Failed to close file %s: %v", configFile.Name(), err)
	}
	return nil
}
