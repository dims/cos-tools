package cos

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"pkg/utils"

	log "github.com/golang/glog"
)

var (
	mockCmdStdout     string
	mockCmdExitStatus = 0
)

func fakeExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	es := strconv.Itoa(mockCmdExitStatus)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1",
		"STDOUT=" + mockCmdStdout,
		"EXIT_STATUS=" + es}
	return cmd
}

// TestHelperProcess is not a real test. It is a helper process for faking exec.Command.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, os.Getenv("STDOUT"))
	es, err := strconv.Atoi(os.Getenv("EXIT_STATUS"))
	if err != nil {
		t.Fatalf("Failed to convert EXIT_STATUS to int: %v", err)
	}
	os.Exit(es)
}

func TestCorrectKernelMagicVersionIfNeeded(t *testing.T) {
	execCommand = fakeExecCommand
	defer func() {
		execCommand = exec.Command
		mockCmdExitStatus = 0
	}()
	for _, tc := range []struct {
		testName              string
		kernelVersionUname    string
		utsRelease            string
		expectedNewUTSRelease string
	}{
		{
			"NeedHack",
			"4.19.101+",
			`#define UTS_RELEASE "4.19.100+"`,
			`#define UTS_RELEASE "4.19.101+"`,
		},
		{
			"NoNeedHack",
			"4.19.101+",
			`#define UTS_RELEASE "4.19.101+"`,
			`#define UTS_RELEASE "4.19.101+"`,
		},
	} {

		tmpDir, err := ioutil.TempDir("", "testing")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)
		utsFile := filepath.Join(tmpDir, utsFilepath)
		if err := os.MkdirAll(filepath.Dir(utsFile), 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}
		if err := ioutil.WriteFile(utsFile, []byte(tc.utsRelease), 0644); err != nil {
			t.Fatalf("Failed to write to utsfile: %v", err)
		}
		mockCmdStdout = tc.kernelVersionUname

		if err := correctKernelMagicVersionIfNeeded(tmpDir); err != nil {
			t.Fatalf("Failed to run correctKernelMagicVersionIfNeeded: %v", err)
		}

		gotUTSRelease, err := ioutil.ReadFile(utsFile)
		if err != nil {
			t.Fatalf("Failed to read utsfile: %v", err)
		}
		if string(gotUTSRelease) != tc.expectedNewUTSRelease {
			t.Errorf("%s: Unexpected newUtsRelease, want: %s, got: %s", tc.testName, tc.expectedNewUTSRelease, gotUTSRelease)
		}
	}
}

func TestDownloadKernelSrc(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "testing")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	downloder := fakeDownloader{}
	if err := downloadKernelSrc(&downloder, tmpDir); err != nil {
		t.Fatalf("Failed to run downloadKernelSrc: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "kernel-source")); err != nil {
		t.Errorf("Failed to get kernel source file: %v", err)
	}
}

func TestInstallKernelHeaderPkg(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "testing")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	downloder := fakeDownloader{}
	if err := InstallKernelHeaderPkg(&downloder, tmpDir); err != nil {
		t.Fatalf("Failed to run InstallKernelHeaderPkg: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "kernel-header")); err != nil {
		t.Errorf("Failed to get kernel headers file: %v", err)
	}
}

func TestSetCompilationEnv(t *testing.T) {
	origEnvs := os.Environ()
	defer func() {
		os.Clearenv()
		for _, env := range origEnvs {
			log.Info(env)
			fields := strings.SplitN(env, "=", 2)
			os.Setenv(fields[0], fields[1])
		}
	}()
	tmpDir, err := ioutil.TempDir("", "testing")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	os.Setenv("TMPDIR", tmpDir)

	downloder := fakeDownloader{}
	if err := SetCompilationEnv(&downloder); err != nil {
		t.Fatalf("Failed to run SetCompilationEnv: %v", err)
	}

	for _, tc := range []struct {
		envKey           string
		expectedEnvValue string
	}{
		{"CC", "x86_64-cros-linux-gnu-clang"},
		{"CXX", "x86_64-cros-linux-gnu-clang++"},
	} {
		if os.Getenv(tc.envKey) != tc.expectedEnvValue {
			t.Errorf("Unexpected env %s value: want: %s, got: %s", tc.envKey, tc.expectedEnvValue, os.Getenv(tc.envKey))
		}
	}
}

func TestInstallCrossToolchain(t *testing.T) {
	origEnvs := os.Environ()
	defer func() {
		os.Clearenv()
		for _, env := range origEnvs {
			log.Info(env)
			fields := strings.SplitN(env, "=", 2)
			os.Setenv(fields[0], fields[1])
		}
	}()
	tmpDir, err := ioutil.TempDir("", "testing")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	origPath := os.Getenv("PATH")

	downloder := fakeDownloader{}
	if err := InstallCrossToolchain(&downloder, tmpDir); err != nil {
		t.Fatalf("Failed to run InstallCrossToolchain: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "x86_64-cros-linux-gnu-clang")); err != nil {
		t.Errorf("Failed to check file in toolchain: %v", err)
	}
	for _, tc := range []struct {
		envKey           string
		expectedEnvValue string
	}{
		{"PATH", tmpDir + "/bin:" + origPath},
		{"SYSROOT", filepath.Join(tmpDir, "usr/x86_64-cros-linux-gnu")},
	} {
		if os.Getenv(tc.envKey) != tc.expectedEnvValue {
			t.Errorf("Unexpected env %s value: want: %s, got: %s", tc.envKey, tc.expectedEnvValue, os.Getenv(tc.envKey))
		}
	}
}

func TestDisableKernelOptionFromGrubCfg(t *testing.T) {
	for _, tc := range []struct {
		testName           string
		kernelOption       string
		grubCfg            string
		expectedNewGrubCfg string
		expectedNeedReboot bool
	}{
		{
			"LoadPin",
			"loadpin.enabled",

			`BOOT_IMAGE=/syslinux/vmlinuz.A init=/usr/lib/systemd/systemd boot=local rootwait ro noresume noswap ` +
				`loglevel=7 noinitrd console=ttyS0 security=apparmor virtio_net.napi_tx=1 ` +
				`systemd.unified_cgroup_hierarchy=false systemd.legacy_systemd_cgroup_controller=false csm.disabled=1 ` +
				`dm_verity.error_behavior=3 dm_verity.max_bios=-1 dm_verity.dev_wait=1 i915.modeset=1 cros_efi root=/dev/dm-0 ` +
				`"dm=1 vroot none ro 1,0 2539520 verity payload=PARTUUID=36547742-9356-EF4E-B9AD-F8DED2F6D087 ` +
				`hashtree=PARTUUID=36547742-9356-EF4E-B9AD-F8DED2F6D087 hashstart=2539520 alg=sha256 ` +
				`root_hexdigest=0ff80250bd97ad47a65e7cd330ab70bcf5013d7a86817dca59fcac77f0ba1a8f ` +
				`salt=414038a6ed9b1f528c327aff4eac16ad5ca4a6699d142ae096e90374af907c34`,

			`BOOT_IMAGE=/syslinux/vmlinuz.A init=/usr/lib/systemd/systemd boot=local rootwait ro noresume noswap ` +
				`loglevel=7 noinitrd console=ttyS0 security=apparmor virtio_net.napi_tx=1 ` +
				`systemd.unified_cgroup_hierarchy=false systemd.legacy_systemd_cgroup_controller=false csm.disabled=1 ` +
				`dm_verity.error_behavior=3 dm_verity.max_bios=-1 dm_verity.dev_wait=1 i915.modeset=1 cros_efi loadpin.enabled=0 root=/dev/dm-0 ` +
				`"dm=1 vroot none ro 1,0 2539520 verity payload=PARTUUID=36547742-9356-EF4E-B9AD-F8DED2F6D087 ` +
				`hashtree=PARTUUID=36547742-9356-EF4E-B9AD-F8DED2F6D087 hashstart=2539520 alg=sha256 ` +
				`root_hexdigest=0ff80250bd97ad47a65e7cd330ab70bcf5013d7a86817dca59fcac77f0ba1a8f ` +
				`salt=414038a6ed9b1f528c327aff4eac16ad5ca4a6699d142ae096e90374af907c34`,
			true,
		},
		{
			"LoadPinEnabled",
			"loadpin.enabled",
			"cros_efi loadpin.enabled=1",
			"cros_efi loadpin.enabled=0",
			true,
		},
		{
			"LoadPinDisabled",
			"loadpin.enabled",
			"cros_efi loadpin.enabled=0",
			"cros_efi loadpin.enabled=0",
			false,
		},
	} {
		newGrubCfg, needReboot := disableKernelOptionFromGrubCfg(tc.kernelOption, tc.grubCfg)
		if newGrubCfg != tc.expectedNewGrubCfg || needReboot != tc.expectedNeedReboot {
			t.Errorf("%v: Unexpected output:\nexpect grubcfg: %v\ngot grubcfg: %v\nexpect needReboot: %v, got needReboot: %v",
				tc.testName, tc.expectedNewGrubCfg, newGrubCfg, tc.expectedNeedReboot, needReboot)
		}
	}
}

type fakeDownloader struct {
}

func (*fakeDownloader) DownloadKernelSrc(destDir string) error {
	var archive = map[string][]byte{
		"kernel-source": []byte("foo"),
	}
	if err := utils.CreateTarFile(filepath.Join(destDir, kernelSrcArchive), archive); err != nil {
		return fmt.Errorf("Failed to download kernel source: %v", err)
	}
	return nil
}

func (*fakeDownloader) DownloadToolchainEnv(destDir string) error {
	toolchainEnvStr := `CC=x86_64-cros-linux-gnu-clang
CXX=x86_64-cros-linux-gnu-clang++
`
	if err := ioutil.WriteFile(filepath.Join(destDir, toolchainEnv), []byte(toolchainEnvStr), 0644); err != nil {
		return fmt.Errorf("Failed to download toolchain env file: %v", err)
	}
	return nil
}

func (*fakeDownloader) DownloadToolchain(destDir string) error {
	var archive = map[string][]byte{
		"x86_64-cros-linux-gnu-clang": []byte("foo"),
	}
	if err := utils.CreateTarFile(filepath.Join(destDir, toolchainArchive), archive); err != nil {
		return fmt.Errorf("Failed to download toolchain archive: %v", err)
	}
	return nil
}

func (*fakeDownloader) DownloadKernelHeaders(destDir string) error {
	var archive = map[string][]byte{
		"kernel-header": []byte("bar"),
	}
	if err := utils.CreateTarFile(filepath.Join(destDir, kernelHeaders), archive); err != nil {
		return fmt.Errorf("Failed to download kernel headers: %v", err)
	}
	return nil
}

func (*fakeDownloader) DownloadArtifact(string, string) error { return nil }

func (*fakeDownloader) GetArtifact(string) ([]byte, error) { return nil, nil }
