package cos

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestPackageInfo(t *testing.T) {
	testDataJSON := `{
    "installedPackages": [
	{
	    "category": "app-arch",
	    "name": "gzip",
	    "version": "1.9"
	},
	{
	    "category": "dev-libs",
	    "name": "popt",
	    "version": "1.16",
	    "revision": "2"
	},
	{
	    "category": "app-emulation",
	    "name": "docker-credential-helpers",
	    "version": "0.6.3",
	    "revision": "1"
	},
	{
	    "category": "_not.real-category1+",
	    "name": "_not-real_package1",
	    "version": "12.34.56.78"
	},
	{
	    "category": "_not.real-category1+",
	    "name": "_not-real_package2",
	    "version": "12.34.56.78",
	    "revision": "26"
	},
	{
	    "category": "_not.real-category1+",
	    "name": "_not-real_package3",
	    "version": "12.34.56.78_rc3"
	},
	{
	    "category": "_not.real-category1+",
	    "name": "_not-real_package4",
	    "version": "12.34.56.78_rc3",
	    "revision": "26"
	},
	{
	    "category": "_not.real-category1+",
	    "name": "_not-real_package5",
	    "version": "12.34.56.78_pre2_rc3",
	    "revision": "26"
	},
	{
	    "category": "_not.real-category2+",
	    "name": "_not-real_package1",
	    "version": "12.34.56.78q"
	},
	{
	    "category": "_not.real-category2+",
	    "name": "_not-real_package2",
	    "version": "12.34.56.78q",
	    "revision": "26"
	},
	{
	    "category": "_not.real-category2+",
	    "name": "_not-real_package3",
	    "version": "12.34.56.78q_rc3"
	},
	{
	    "category": "_not.real-category2+",
	    "name": "_not-real_package4",
	    "version": "12.34.56.78q_rc3",
	    "revision": "26"
	},
	{
	    "category": "_not.real-category2+",
	    "name": "_not-real_package5",
	    "version": "12.34.56.78q_pre2_rc3",
	    "revision": "26"
	}
    ]
}`

	testFile, err := ioutil.TempFile("", "pkg_info_test")
	if err != nil {
		t.Fatalf("Failed to create tempfile: %v", err)
	}
	defer os.Remove(testFile.Name())
	_, err = testFile.WriteString(testDataJSON)
	if err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	err = testFile.Close()
	if err != nil {
		t.Fatalf("Failed to close test file: %v", err)
	}

	packageInfo, err := GetPackageInfoFromFile(testFile.Name())
	if err != nil {
		t.Fatalf("Failed to read package information: %v", err)
	}

	installedPackages := packageInfo.InstalledPackages
	if len(installedPackages) != 13 {
		t.Errorf("Installed packages length is wrong. want: 13, got: %d",
			len(installedPackages))

	}

	checkPackage(t, installedPackages[0], "app-arch", "gzip", "1.9", 0)
	checkPackage(t, installedPackages[1], "dev-libs", "popt", "1.16", 2)
	checkPackage(t, installedPackages[2], "app-emulation", "docker-credential-helpers", "0.6.3", 1)
	checkPackage(t, installedPackages[3], "_not.real-category1+", "_not-real_package1", "12.34.56.78", 0)
	checkPackage(t, installedPackages[4], "_not.real-category1+", "_not-real_package2", "12.34.56.78", 26)
	checkPackage(t, installedPackages[5], "_not.real-category1+", "_not-real_package3", "12.34.56.78_rc3", 0)
	checkPackage(t, installedPackages[6], "_not.real-category1+", "_not-real_package4", "12.34.56.78_rc3", 26)
	checkPackage(t, installedPackages[7], "_not.real-category1+", "_not-real_package5", "12.34.56.78_pre2_rc3", 26)
	checkPackage(t, installedPackages[8], "_not.real-category2+", "_not-real_package1", "12.34.56.78q", 0)
	checkPackage(t, installedPackages[9], "_not.real-category2+", "_not-real_package2", "12.34.56.78q", 26)
	checkPackage(t, installedPackages[10], "_not.real-category2+", "_not-real_package3", "12.34.56.78q_rc3", 0)
	checkPackage(t, installedPackages[11], "_not.real-category2+", "_not-real_package4", "12.34.56.78q_rc3", 26)
	checkPackage(t, installedPackages[12], "_not.real-category2+", "_not-real_package5", "12.34.56.78q_pre2_rc3", 26)
}

func checkPackage(t *testing.T, p Package, category string, name string, version string, revision int) {
	if p.Category != category {
		t.Errorf("Wrong package category in package %v. want: %s, got: %s",
			p, category, p.Category)
	}
	if p.Name != name {
		t.Errorf("Wrong package name in package %v. want: %s, got: %s",
			p, name, p.Name)
	}
	if p.Version != version {
		t.Errorf("Wrong package version in package %v. want: %s, got: %s",
			p, version, p.Version)
	}
	if p.Revision != revision {
		t.Errorf("Wrong package revision in package %v. want: %d, got: %d",
			p, revision, p.Revision)
	}
}
