// Package utils defines parsing and run command functions
// that can be used outside nodeprofiler.
package utils

import (
	"fmt"
	"os/exec"
)

// RunCommand is a wrapper function for exec.Command that will run the command
// specified return its output and/or error.
func RunCommand(cmd string, args ...string) ([]byte, error) {
	out, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run %s, %v: %v", cmd, args, err)
	}
	return out, nil
}
