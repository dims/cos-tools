package installer

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestIsCached(t *testing.T) {
	testDir, err := ioutil.TempDir("", "testing")
	if err != nil {
		t.Fatalf("Failed to create tempdir: %v", err)
	}
	defer os.RemoveAll(testDir)

	cacher := NewCacher(testDir, "12688.0.0", "418.67")
	if err := cacher.Cache(); err != nil {
		t.Fatalf("Failed to cache: %v", err)
	}

	for _, tc := range []struct {
		testName      string
		buildNumber   string
		driverVersion string
		expectOut     bool
	}{
		{"TestIsCachedTrue", "12688.0.0", "418.67", true},
		{"TestIsCachedWrongBuild", "12670.0.0", "418.67", false},
		{"TestIsCachedWrongDriver", "12688.0.0", "418.00", false},
	} {
		t.Run(tc.testName, func(t *testing.T) {
			testCacher := NewCacher(testDir, tc.buildNumber, tc.driverVersion)
			out, err := testCacher.IsCached()
			if err != nil {
				t.Fatalf("Failed to check cache result: %v", err)
			}
			if out != tc.expectOut {
				t.Errorf("Unexpected cache result: want :%v, got: %v", tc.expectOut, out)
			}
		})
	}
}
