// Copyright 2020 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This file contains the CLI application for COS Changelog.
// Project information available at go/cos-changelog
//
// This application is responsible for:
// 1. Accepting user input and creating the authenticator object used for
//    queries
// 2. Calling function in the Changelog package, converting the output
//    to json, and writing the result into a file

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"cos.googlesource.com/cos/tools/src/pkg/changelog"
	"cos.googlesource.com/cos/tools/src/pkg/findbuild"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"go.chromium.org/luci/common/api/gerrit"
)

const (
	externalGerritURL    = "https://cos-review.googlesource.com"
	fallbackGerritURL    = "https://chromium-review.googlesource.com"
	externalGoBURL       = "cos.googlesource.com"
	externalManifestRepo = "cos/manifest-snapshots"
	fallbackRepoPrefix   = "mirrors/cros/"
)

func unwrappedError(err error) error {
	innerErr := err
	for errors.Unwrap(innerErr) != nil {
		innerErr = errors.Unwrap(innerErr)
	}
	return innerErr
}

func getHTTPClient() (*http.Client, error) {
	log.Debug("Creating HTTP client")
	creds, err := google.FindDefaultCredentials(context.Background(), gerrit.OAuthScope)
	if err != nil || len(creds.JSON) == 0 {
		return nil, fmt.Errorf("no application default credentials found - run `gcloud auth application-default login` and try again")
	}
	return oauth2.NewClient(oauth2.NoContext, creds.TokenSource), nil
}

func writeChangelogAsJSON(source string, target string, changes map[string]*changelog.RepoLog) error {
	fileName := fmt.Sprintf("%s -> %s.json", source, target)
	log.Infof("Writing changelog to %s\n", fileName)
	jsonData, err := json.MarshalIndent(changes, "", "    ")
	if err != nil {
		return fmt.Errorf("writeChangelogAsJSON: error marshalling changelog from: %s to: %s\n%v", source, target, err)
	}
	if err = ioutil.WriteFile(fileName, jsonData, 0644); err != nil {
		return fmt.Errorf("writeChangelogAsJSON: error writing changelog to file: %s\n%v", fileName, err)
	}
	return nil
}

func generateChangelog(source, target, instance, manifestRepo string) error {
	start := time.Now()
	httpClient, err := getHTTPClient()
	if err != nil {
		return fmt.Errorf("generateChangelog: failed to create http client: \n%v", err)
	}
	sourceToTargetChanges, targetToSourceChanges, err := changelog.Changelog(httpClient, source, target, instance, manifestRepo, -1)
	if err != nil {
		return fmt.Errorf("generateChangelog: error retrieving changelog between builds %s and %s on GoB instance: %s with manifest repository: %s\n%v",
			source, target, instance, manifestRepo, err)
	}
	if err := writeChangelogAsJSON(source, target, sourceToTargetChanges); err != nil {
		log.Errorf("generateChangelog: error writing first changelog with source: %s and target: %s\n%v\n",
			source, target, err)
	}
	if err := writeChangelogAsJSON(target, source, targetToSourceChanges); err != nil {
		log.Errorf("generateChangelog: Error writing second changelog with source: %s and target: %s\n%v\n",
			target, source, err)
	}
	log.Infof("Retrieved changelog in %s\n", time.Since(start))
	return nil
}

func getBuildForCL(gerrit, fallback, gob, manifestRepo, fallbackPrefix, targetCL string) error {
	httpClient, err := getHTTPClient()
	if err != nil {
		return fmt.Errorf("Error creating http client: %v", err)
	}
	req := &findbuild.BuildRequest{
		HTTPClient:   httpClient,
		GerritHost:   gerrit,
		GitilesHost:  gob,
		ManifestRepo: manifestRepo,
		RepoPrefix:   "",
		CL:           targetCL,
	}
	buildData, err := findbuild.FindBuild(req)
	if unwrappedError(err) == findbuild.ErrorCLNotFound {
		log.Debugf("Query failed on Gerrit url %s and Gitiles url %s, retrying with fallback urls", externalGerritURL, externalGoBURL)
		fallbackReq := &findbuild.BuildRequest{
			HTTPClient:   httpClient,
			GerritHost:   fallback,
			GitilesHost:  gob,
			ManifestRepo: manifestRepo,
			RepoPrefix:   fallbackPrefix,
			CL:           targetCL,
		}
		buildData, err = findbuild.FindBuild(fallbackReq)
	}
	if err != nil {
		return err
	}
	log.Infof("Build: %s", buildData.BuildNum)
	return nil
}

func main() {
	var mode, gobURL, gerritURL, fallbackURL, manifestRepo, fallbackPrefix string
	var debug bool
	app := &cli.App{
		Name:  "changelog",
		Usage: "get commits between builds or first build containing CL",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "mode",
				Value:       "",
				Aliases:     []string{"m"},
				Usage:       "Specify query mode. Acceptable values: changelog | findbuild",
				Destination: &mode,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "gerrit",
				Value:       externalGerritURL,
				Usage:       "Gerrit `URL` to query from",
				Destination: &gerritURL,
			},
			&cli.StringFlag{
				Name:        "fallback",
				Value:       fallbackGerritURL,
				Usage:       "Fallback Gerrit `URL` to query from",
				Destination: &fallbackURL,
			},
			&cli.StringFlag{
				Name:        "gob",
				Value:       externalGoBURL,
				Usage:       "Git on Borg `URL` to query from",
				Destination: &gobURL,
			},
			&cli.StringFlag{
				Name:        "repo",
				Value:       externalManifestRepo,
				Aliases:     []string{"r"},
				Usage:       "`REPO` containing Manifest file",
				Destination: &manifestRepo,
			},
			&cli.StringFlag{
				Name:        "prefix",
				Value:       fallbackRepoPrefix,
				Aliases:     []string{"p"},
				Usage:       "`PREFIX` prepended to repo when querying GoB using fallback Gerrit results",
				Destination: &fallbackPrefix,
			},
			&cli.BoolFlag{
				Name:        "debug",
				Value:       false,
				Aliases:     []string{"d"},
				Usage:       "Toggle debug messages",
				Destination: &debug,
			},
		},
		Action: func(c *cli.Context) error {
			if debug {
				log.SetLevel(log.DebugLevel)
			}
			switch mode {
			case "findbuild":
				if c.NArg() != 1 {
					return errors.New("Must specify CL number or change ID")
				}
				targetCL := c.Args().Get(0)
				return getBuildForCL(gerritURL, fallbackURL, gobURL, manifestRepo, fallbackPrefix, targetCL)
			case "changelog":
				if c.NArg() != 2 {
					return errors.New("Must specify source and target build number")
				}
				source := c.Args().Get(0)
				target := c.Args().Get(1)
				return generateChangelog(source, target, gobURL, manifestRepo)
			default:
				return fmt.Errorf("Please specify either \"findbuild\" or \"changelog\" mode")
			}
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("main: error running app with arguments: %v:\n%v", os.Args, err)
		os.Exit(1)
	}
}
