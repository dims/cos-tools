package utils

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestLock(t *testing.T) {
	if os.Getenv("TEST_LOCK") == "1" {
		origLockFile := lockFile
		lockFile = os.Getenv("TEST_DIR")
		defer func(origLockFile string) { lockFile = origLockFile }(origLockFile)
		Flock()
		// forever so that the filelock won't be released.
		for {
		}
	}

	tmpfile, err := ioutil.TempFile("", "testing")
	if err != nil {
		t.Fatalf("Failed to create tempfile: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// First time to call Lock(), expect to wait forever
	cmd1 := exec.Command(os.Args[0], "-test.run=TestLock")
	cmd1.Env = append(os.Environ(), "TEST_LOCK=1", "TEST_DIR="+tmpfile.Name())
	if err := cmd1.Start(); err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	// Wait 1 sec for the first process to lock file.
	time.Sleep(time.Second)

	// Second time to call Lock(), expect to exit with status 1
	cmd2 := exec.Command(os.Args[0], "-test.run=TestLock")
	cmd2.Env = append(os.Environ(), "TEST_LOCK=1", "TEST_DIR="+tmpfile.Name())
	if err := cmd2.Start(); err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	waitWithTimeout(t, cmd1, 3, false)
	waitWithTimeout(t, cmd2, 3, true)
}

func waitWithTimeout(t *testing.T, cmd *exec.Cmd, timeout int, expectError bool) {
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			t.Fatalf("Failed to kill process: %v", err)
		}
		if expectError {
			t.Errorf("Process %s didn't exit while expecting to exit with error", cmd.Path)
		}
	case err := <-done:
		e, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatal("Failed to convert error to exec.ExitError")
		}
		if e.Success() == expectError {
			t.Errorf("Process %s exited with unexpected status, want error: %v, got error: %v",
				cmd.Path, expectError, !e.Success())
		}
	}
}

func TestParseVMToken(t *testing.T) {
	token, err := parseVMToken(
		`{"access_token":"ya29.c.Kmi8B89nrn2Esf2e4WEk2MlZp7G8EpMatfxD36UuG3QJpwqePPxLAMvlb-WEi-nnZ7WmFsxyTAhzFMlxBV4AEYfs1tdJqolDay_3BXkwv0cwFe6OO86_dSUWDbiK9gIYQ6bAE_oR9SdVdw","expires_in":3248,"token_type":"Bearer"}`)
	if err != nil {
		t.Fatalf("Failed to run parseVMToken: %v", err)
	}
	expectedToken := serviceAccountToken{
		Token:     "ya29.c.Kmi8B89nrn2Esf2e4WEk2MlZp7G8EpMatfxD36UuG3QJpwqePPxLAMvlb-WEi-nnZ7WmFsxyTAhzFMlxBV4AEYfs1tdJqolDay_3BXkwv0cwFe6OO86_dSUWDbiK9gIYQ6bAE_oR9SdVdw",
		Expire:    3248,
		TokenType: "Bearer",
	}
	if diff := cmp.Diff(*token, expectedToken); diff != "" {
		t.Errorf("Unexpected return\nwant: %v\ngot: %v\ndiff: %v", expectedToken, *token, diff)
	}
}

func TestIsDirEmpty(t *testing.T) {
	emptyDir, err := ioutil.TempDir("", "testing")
	if err != nil {
		t.Fatalf("Failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(emptyDir)

	nonEmptyDir, err := ioutil.TempDir("", "testing")
	if err != nil {
		t.Fatalf("Failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(nonEmptyDir)

	tmpfile, err := ioutil.TempFile(nonEmptyDir, "testing")
	if err != nil {
		t.Fatalf("Failed to create tmp file: %v", err)
	}

	defer os.Remove(tmpfile.Name())

	for _, tc := range []struct {
		testName    string
		dir         string
		expectEmpty bool
	}{
		{"EmptyDir", emptyDir, true},
		{"NonEmptyDir", nonEmptyDir, false},
	} {
		ret, _ := IsDirEmpty(tc.dir)
		if ret != tc.expectEmpty {
			t.Errorf("%v: Unexpected return, want: %v, got: %v", tc.testName, tc.expectEmpty, ret)
		}
	}
}

func TestLoadEnvFromFile(t *testing.T) {
	testDir, err := ioutil.TempDir("", "testing")
	if err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}
	defer os.RemoveAll(testDir)
	envStr := `key1=value1
key2=value2`
	if err := ioutil.WriteFile(filepath.Join(testDir, "env"), []byte(envStr), 0644); err != nil {
		t.Fatalf("Failed to write to env file: %v", err)
	}

	envs, err := LoadEnvFromFile(testDir, "env")
	if err != nil {
		t.Fatalf("Failed to read from env file: %v", err)
	}

	expectedEnvs := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	if diff := cmp.Diff(envs, expectedEnvs); diff != "" {
		t.Errorf("Unexpected envs, want: %v, got: %v, diff: %v", expectedEnvs, envs, diff)
	}
}

func TestCut(t *testing.T) {
	for _, tt := range []struct {
		s, sep        string
		before, after string
		found         bool
	}{
		{"abc", "b", "a", "c", true},
		{"abc", "a", "", "bc", true},
		{"abc", "c", "ab", "", true},
		{"abc", "abc", "", "", true},
		{"abc", "", "", "abc", true},
		{"abc", "d", "abc", "", false},
		{"", "d", "", "", false},
		{"", "", "", "", true},
	} {
		if before, after, found := Cut(tt.s, tt.sep); before != tt.before || after != tt.after || found != tt.found {
			t.Errorf("Cut(%q, %q) = %q, %q, %v, want %q, %q, %v", tt.s, tt.sep, before, after, found, tt.before, tt.after, tt.found)
		}
	}
}
