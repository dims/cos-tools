package cos

import (
	"testing"
)

func TestPackageInfo(t *testing.T) {
	tests := []struct {
		name              string
		inputFile         string
		wantInstalledPkgs []Package
		wantBuildTimePkgs []Package
	}{
		{
			name:      "OnlyInstalledPackages",
			inputFile: "testdata/only_installed_packages.json",
			wantInstalledPkgs: []Package{
				Package{
					Category:      "app-arch",
					Name:          "gzip",
					Version:       "123",
					EbuildVersion: "1.9",
				},
				Package{
					Category:      "dev-libs",
					Name:          "popt",
					Version:       "123",
					EbuildVersion: "1.16-r2",
				},
				Package{
					Category:      "app-emulation",
					Name:          "docker-credential-helpers",
					Version:       "123",
					EbuildVersion: "0.6.3-r1",
				},
			},
		},
		{
			name:      "OnlyBuildTimePackages",
			inputFile: "testdata/only_build_time_packages.json",
			wantBuildTimePkgs: []Package{
				Package{
					Category:      "virtual",
					Name:          "pkgconfig",
					Version:       "123",
					EbuildVersion: "0-r1",
				},
				Package{
					Category:      "dev-go",
					Name:          "protobuf",
					Version:       "123",
					EbuildVersion: "1.3.2",
				},
			},
		},
		{
			name:      "AllPackages",
			inputFile: "testdata/packages.json",
			wantInstalledPkgs: []Package{
				Package{
					Category:      "app-arch",
					Name:          "gzip",
					Version:       "123",
					EbuildVersion: "1.9",
				},
				Package{
					Category:      "dev-libs",
					Name:          "popt",
					Version:       "123",
					EbuildVersion: "1.16-r2",
				},
				Package{
					Category:      "app-emulation",
					Name:          "docker-credential-helpers",
					Version:       "123",
					EbuildVersion: "0.6.3-r1",
				},
			},
			wantBuildTimePkgs: []Package{
				Package{
					Category:      "virtual",
					Name:          "pkgconfig",
					Version:       "123",
					EbuildVersion: "0-r1",
				},
				Package{
					Category:      "dev-go",
					Name:          "protobuf",
					Version:       "123",
					EbuildVersion: "1.3.2",
				},
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			packageInfo, err := GetPackageInfoFromFile(test.inputFile)
			if err != nil {
				t.Fatalf("Failed to read package information: %v", err)
			}

			installedPackages := packageInfo.InstalledPackages
			if len(installedPackages) != len(test.wantInstalledPkgs) {
				t.Errorf("Installed packages length is wrong. want: %d, got: %d", len(test.wantInstalledPkgs), len(installedPackages))
			}

			for index, pkg := range test.wantInstalledPkgs {
				checkPackage(t, installedPackages[index], pkg.Category, pkg.Name, pkg.Version, pkg.EbuildVersion)
			}

			buildTimePackages := packageInfo.BuildTimePackages
			if len(buildTimePackages) != len(test.wantBuildTimePkgs) {
				t.Errorf("Build Time packages length is wrong. want: %d, got: %d", len(test.wantBuildTimePkgs), len(buildTimePackages))
			}

			for index, pkg := range test.wantBuildTimePkgs {
				checkPackage(t, buildTimePackages[index], pkg.Category, pkg.Name, pkg.Version, pkg.EbuildVersion)
			}
		})
	}
}

func checkPackage(t *testing.T, p Package, category string, name string, version string, ebuildVersion string) {
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
	if p.EbuildVersion != ebuildVersion {
		t.Errorf("Wrong package ebuild version in package %v. want: %s, got: %s",
			p, ebuildVersion, p.EbuildVersion)
	}
}
