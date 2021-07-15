// Package utils defines parsing and run command functions
// that can be used outside nodeprofiler.
package utils

import (
	"fmt"
	"os/exec"
	"strconv"

	log "github.com/sirupsen/logrus"
)

// RunCommand is a wrapper function for exec.Command that will run the command
// specified return its output and/or error.
func RunCommand(cmd string, args ...string) ([]byte, error) {
	log.Printf("running %q", cmd)

	out, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run %q, %v: %v", cmd, args, err)
	}

	log.Printf("finished running %s command successfully", cmd)
	return out, nil
}

// SumAtoi converts all the strings in a slice to integers, sums them up and returns
// the result. A non-nil error is returned if an error occurred.
func SumAtoi(a []string) (int, error) {
	var sum int
	for _, str := range a {
		val, err := strconv.Atoi(str)
		if err != nil {
			return 0, fmt.Errorf("could not convert %q to an int: %v", str, err)
		}
		sum += val
	}
	return sum, nil
}

// SumParseFloat converts all the strings in a slice to floating points, sums them up, and
// returns the result as a floating point. A non-nil error is returned if an error occurred.
func SumParseFloat(a []string) (float64, error) {
	var sum float64
	for _, str := range a {
		val, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return 0, fmt.Errorf("could not convert %q to float64: %v", str, err)
		}
		sum += val
	}
	return sum, nil
}
