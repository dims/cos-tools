package installer

import (
	"testing"
)

func TestGetInstallerDownloadLocation(t *testing.T) {
	for _, tc := range []struct {
		testName         string
		metadataZone     string
		expectedLocation string
	}{
		{
			"us-west1-b",
			"projects/123456789/zones/us-west1-b",
			"us",
		},
		{
			"asia-east1-a",
			"projects/123456789/zones/asia-east1-a",
			"asia",
		},
		{
			"europe-west1-b",
			"projects/123456789/zones/europe-west1-b",
			"eu",
		},
		{
			"australia-southeast1-a",
			"projects/123456789/zones/australia-southeast1-a",
			"us",
		},
	} {
		location := getInstallerDownloadLocation(tc.metadataZone)
		if location != tc.expectedLocation {
			t.Errorf("%s: expect location: %s, got: %s", tc.testName, tc.expectedLocation, location)
		}
	}
}

func TestGetPrecompiledInstallerURL(t *testing.T) {
	ret := getPrecompiledInstallerURL("418.116.00", "73", "11647.415.0", "us")
	expectedRet := "https://storage.googleapis.com/nvidia-drivers-us-public/nvidia-cos-project/73/tesla/418_00/418.116.00/NVIDIA-Linux-x86_64-418.116.00_73-11647-415-0.cos"
	if ret != expectedRet {
		t.Errorf("Unexpected return, want: %s, got: %s", expectedRet, ret)
	}
}

func TestGetGenericDriverInstallerURL(t *testing.T) {
	ret, err := getGenericDriverInstallerURL("525.125.06")
	if err != nil {
		t.Errorf("Unexpected err, want: nil, got: %v", err)
	}
	expectedRet := "https://storage.googleapis.com/nvidia-drivers-us-public/tesla/525.125.06/NVIDIA-Linux-x86_64-525.125.06.run"
	if ret != expectedRet {
		t.Errorf("Unexpected return, want: %s, got: %s", expectedRet, ret)
	}
}
