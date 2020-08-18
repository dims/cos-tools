package packagediff

import (
	"testing"

	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/input"
	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/utilities"
)

// test searchPackageList function
func TestSearchPackageList(t *testing.T) {
	testPackageList1 := []Package{
		{Category: "sys-boot", Name: "shim"},
		{Category: "chromeos-launch", Name: "cloud-network-boot", Version: "1.0.0", Revision: "4"},
		{Revision: "533"}}
	testPackageList2 := []Package{}
	for _, tc := range []struct {
		packageName string
		packageList []Package
		wantPackage Package
		wantOk      bool
	}{
		{packageName: "shim", packageList: testPackageList1, wantPackage: Package{Category: "sys-boot", Name: "shim"}, wantOk: true},
		{packageName: "sys-boot", packageList: testPackageList1, wantPackage: Package{}, wantOk: false},
		{packageName: "", packageList: testPackageList1, wantPackage: Package{Revision: "533"}, wantOk: true},
		{packageName: "cloud-network-boot", packageList: testPackageList2, wantPackage: Package{}, wantOk: false},
	} {
		gotPackage, gotOk := searchPackageList(tc.packageName, tc.packageList)
		if tc.wantOk != gotOk {
			t.Fatalf("searchPackageList call expected: %v, got: %v", tc.wantOk, gotOk)
		}
		if tc.wantPackage.Name != gotPackage.Name {
			t.Fatalf("searchPackageList expected: %v, got: %v", tc.wantPackage, gotPackage)
		}
		if tc.wantPackage.Category != gotPackage.Category {
			t.Fatalf("searchPackageList expected: %v, got: %v", tc.wantPackage, gotPackage)
		}
		if tc.wantPackage.Version != gotPackage.Version {
			t.Fatalf("searchPackageList expected: %v, got: %v", tc.wantPackage, gotPackage)
		}
		if tc.wantPackage.Revision != gotPackage.Revision {
			t.Fatalf("searchPackageList expected: %v, got: %v", tc.wantPackage, gotPackage)
		}
	}
}

// test Diff function
func TestDiff(t *testing.T) {
	testPackageList1 := []Package{
		{Category: "sys-boot", Name: "shim", Version: "14.0.20180308", Revision: "4"},
		{Category: "chromeos-launch", Name: "cloud-network-boot", Version: "1.0.0", Revision: "4"},
		{Category: "sys-kernel", Name: "lakitu-kernel-4_19", Version: "4.20.127", Revision: "533"},
		{Category: "sys-apps", Name: "findutils", Version: "4.9.10", Revision: "1"},
		{Category: "app-emulation", Name: "runc", Version: "1.0.0_rc10", Revision: "1"}}
	testPackageList2 := []Package{}
	testPackageList3 := []Package{
		{Category: "sys-boot", Name: "shim", Version: "14.0.20180308", Revision: "4"},
		{Category: "chromeos-base", Name: "cloud-network-init", Version: "1.0.0", Revision: "4"},
		{Category: "sys-kernel", Name: "lakitu-kernel-4_19", Version: "4.19.127", Revision: "535"},
		{Category: "app-shells", Name: "dash", Version: "0.5.9.1", Revision: "7"}}

	testPackageDiff := []PkgDiff{
		{category: []string{"chromeos-launch"}, name: []string{"cloud-network-boot"}, version: []string{"1.0.0"}, revision: []string{"4"}, typeOFDiff: "image1"},
		{name: []string{"lakitu-kernel-4_19", "lakitu-kernel-4_19"}, version: []string{"4.20.127", "4.19.127"}, revision: []string{"533", "535"}, typeOFDiff: "shared"},
		{category: []string{"sys-apps"}, name: []string{"findutils"}, version: []string{"4.9.10"}, revision: []string{"1"}, typeOFDiff: "image1"},
		{category: []string{"app-emulation"}, name: []string{"runc"}, version: []string{"1.0.0_rc10"}, revision: []string{"1"}, typeOFDiff: "image1"},
		{category: []string{"chromeos-base"}, name: []string{"cloud-network-init"}, version: []string{"1.0.0"}, revision: []string{"4"}, typeOFDiff: "image2"},
		{category: []string{"app-shells"}, name: []string{"dash"}, version: []string{"0.5.9.1"}, revision: []string{"7"}, typeOFDiff: "image2"}}
	testPkgDiffOneImage := &Differences{PackageList: testPackageList1}
	testPkgDiffTwoImages := &Differences{PackageDiff: testPackageDiff}

	for _, tc := range []struct {
		packagesImage1 []Package
		packagesImage2 []Package
		flagInfo       *input.FlagInfo
		want           *Differences
	}{ // One Image Test
		{packagesImage1: testPackageList1,
			packagesImage2: testPackageList2,
			flagInfo:       &input.FlagInfo{Image2: "", PackageSelected: true},
			want:           testPkgDiffOneImage},
		// TWo Image Test
		{packagesImage1: testPackageList1,
			packagesImage2: testPackageList3,
			flagInfo:       &input.FlagInfo{Image2: "../testdata/image2", PackageSelected: true},
			want:           testPkgDiffTwoImages},
	} {
		got, _ := Diff(tc.packagesImage1, tc.packagesImage2, tc.flagInfo)

		for _, pl := range tc.want.PackageList {
			pg, ok := searchPackageList(pl.Name, got.PackageList)
			if !ok {
				t.Fatalf("GetPackageInfo expected: %v, got: %v", tc.want, got)
			}
			if pl.Name != pg.Name {
				t.Fatalf("GetPackageInfo expected: %v, got: %v", tc.want, got)
			}
			if pl.Category != pg.Category {
				t.Fatalf("GetPackageInfo expected: %v, got: %v", tc.want, got)
			}
			if pl.Version != pg.Version {
				t.Fatalf("GetPackageInfo expected: %v, got: %v", tc.want, got)
			}
			if pl.Revision != pg.Revision {
				t.Fatalf("GetPackageInfo expected: %v, got: %v", tc.want, got)
			}
		}

		for _, pd := range tc.want.PackageDiff {
			foundPkg := false
			for _, pg := range got.PackageDiff {
				if utilities.EqualArrays(pd.name, pg.name) {
					if !utilities.EqualArrays(pd.category, pg.category) {
						t.Fatalf("Package diff expected: %v, got: %v", tc.want, got)
					}
					if !utilities.EqualArrays(pd.version, pg.version) {
						t.Fatalf("Package diff expected: %v, got: %v", tc.want, got)
					}
					if !utilities.EqualArrays(pd.revision, pg.revision) {
						t.Fatalf("Package diff expected: %v, got: %v", tc.want, got)
					}
					foundPkg = true
				}
			}
			if !foundPkg {
				t.Fatalf("Package diff expected: %v, got: %v", tc.want, got)
			}
		}
	}
}
