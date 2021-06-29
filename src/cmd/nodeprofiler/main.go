// Package main is the entry point for the cos_node_profiler application that
// imports the cloudlogger, profiler, and utils packages that respectively
// write logs to Google Cloud Logging backend, fetch debugging information from
// a given Linux system and provide the interface between cloudlogger and
// profiler packages.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/logging"
	"cos.googlesource.com/cos/tools.git/src/pkg/nodeprofiler/cloudlogger"
	"cos.googlesource.com/cos/tools.git/src/pkg/nodeprofiler/utils"
)

const cloudLoggerName = "cos_node_profiler"

func main() {
	projID := flag.String("project", "cos-interns-playground", "Specifies the GCP project where logs will be added.")
	flag.Parse()
	// [START client setup]
	ctx := context.Background()
	client, err := logging.NewClient(ctx, *projID)
	if err != nil {
		log.Fatalf("Failed to create logging client: %v", err)
	}
	defer client.Close()
	client.OnError = func(err error) {
		// Print an error to the local log.
		// For example, if Flush() failed.
		log.Printf("client.OnError: %v", err)
	}
	// [END client setup]

	// [BEGIN write entry]
	log.Print("Writing some log entries.")
	logger := client.Logger(cloudLoggerName)
	// Ensure the entry is written
	defer logger.Flush()
	cloudlogger.LogUSEReport(logger, GenerateUSEReport())
	// [END write entry]
}

// func returns mocked out useReport data
func GenerateUSEReport() *utils.USEReport {
	// [Begin Making of mocked out instances]
	useMetrics := []utils.USEMetrics{

		{
			Timestamp:   time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
			Interval:    time.Duration(100) * time.Millisecond,
			Utilization: 4,
			Saturation:  false,
			Errors:      int64(0),
		},
		{
			Timestamp:   time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
			Interval:    time.Duration(100) * time.Millisecond,
			Utilization: 7,
			Saturation:  false,
			Errors:      int64(0),
		},
	}
	analysis := "No action required"
	useReport := &utils.USEReport{Components: useMetrics, Analysis: analysis}
	// [End Making of mocked out instances]
	return useReport

}

func usage(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(2)
}
