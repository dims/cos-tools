package modules

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
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

func TestHasInstalled(t *testing.T) {
	execCommand = fakeExecCommand
	defer func() {
		execCommand = exec.Command
		mockCmdExitStatus = 0
	}()

	for _, tc := range []struct {
		testName      string
		moduleName    string
		cmdStdout     string
		cmdExitStatus int
		expectOutput  bool
	}{
		{"TestModuleInstalled", "nf_nat",
			"Module\tSize\tUsed by\nnf_nat_ipv4\t16384\t2 ipt_MASQUERADE,iptable_nat\nnf_nat\t53248\t1 nf_nat_ipv4\n",
			0, true,
		},
		{"TestModuleNotInstalled", "fat",
			"Module\tSize\tUsed by\nnf_nat_ipv4\t16384\t2 ipt_MASQUERADE,iptable_nat\nnf_nat\t53248\t1 nf_nat_ipv4\n",
			0, false,
		},
	} {
		t.Run(tc.testName, func(t *testing.T) {
			mockCmdStdout = tc.cmdStdout
			mockCmdExitStatus = tc.cmdExitStatus
			out, err := isModuleLoaded(tc.moduleName)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if out != tc.expectOutput {
				t.Errorf("Unexpected return value, want %v, got %v", tc.expectOutput, out)
			}
		})
	}
}

func TestAppendSignature(t *testing.T) {
	modulefile, err := ioutil.TempFile("", "modulefile")
	if err != nil {
		t.Fatalf("AppendSignature: failed to create temp file: %v", err)
	}
	defer os.Remove(modulefile.Name())
	sigfile, err := ioutil.TempFile("", "sigfile")
	if err != nil {
		t.Fatalf("AppendSignature: failed to create temp file: %v", err)
	}
	defer os.Remove(sigfile.Name())

	_, err = modulefile.Write([]byte("module"))
	if err != nil {
		t.Fatalf("AppendSignature: failed to write to file %s: %v", modulefile.Name(), err)
	}
	if err := modulefile.Close(); err != nil {
		t.Fatalf("AppendSignature: failed to close file %s: %v", modulefile.Name(), err)
	}

	_, err = sigfile.Write([]byte("signature"))
	if err != nil {
		t.Fatalf("AppendSignature: failed to write to file %s: %v", sigfile.Name(), err)
	}
	if err := sigfile.Close(); err != nil {
		t.Fatalf("AppendSignature: failed to close file %s: %v", sigfile.Name(), err)
	}

	if err := AppendSignature(modulefile.Name(), modulefile.Name(), sigfile.Name()); err != nil {
		t.Fatalf("AppendSignature: failed to run with error: %v", err)
	}
	signedModuleBytes, err := ioutil.ReadFile(modulefile.Name())
	if err != nil {
		t.Fatalf("AppendSignature: failed to read signed module file: %v", err)
	}
	expectedBytes := [...]byte{
		// The following line is the bytes of the original module: "module"
		0x6D, 0x6F, 0x64, 0x75, 0x6c, 0x65,
		// The following line is the bytes of the signature: "signature"
		0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65,
		// The following lines are the bytes of module_signature struct
		0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x09,
		// The following lines are the bytes of PKCS7 message: "~Module signature appended~\n"
		0x7e, 0x4d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x20, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x20, 0x61,
		0x70, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x64, 0x7e, 0xa,
	}

	if diff := cmp.Diff(expectedBytes[:], signedModuleBytes); diff != "" {
		t.Errorf("AppendSignature: signedModuleBytes doesn't match,\nwant: %v\ngot: %v\ndiff: %v",
			expectedBytes, signedModuleBytes, diff)
	}
}
