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
	"os"
	"time"

	"cos.googlesource.com/cos/tools/pkg/changelog"

	"github.com/google/martian/log"
	"github.com/urfave/cli/v2"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/api/gerrit"
	"go.chromium.org/luci/hardcoded/chromeinfra"
)

// Default Manifest location
const (
	cosInstanceURL      string = "cos.googlesource.com"
	defaultManifestRepo string = "cos/manifest-snapshots"
)

func getAuthenticator() *auth.Authenticator {
	opts := chromeinfra.DefaultAuthOptions()
	opts.Scopes = []string{gerrit.OAuthScope, auth.OAuthScopeEmail}
	return auth.NewAuthenticator(context.Background(), auth.InteractiveLogin, opts)
}

func writeChangelogAsJSON(source string, target string, changes map[string][]*changelog.Commit) error {
	jsonData, err := json.MarshalIndent(changes, "", "    ")
	if err != nil {
		return fmt.Errorf("writeChangelogAsJSON: Error marshalling changelog from: %s to: %s\n%v", source, target, err)
	}
	fileName := fmt.Sprintf("%s -> %s.json", source, target)
	if err = ioutil.WriteFile(fileName, jsonData, 0644); err != nil {
		return fmt.Errorf("writeChangelogAsJSON: Error writing changelog to file: %s\n%v", fileName, err)
	}
	return nil
}

func generateChangelog(source, target, instance, manifestRepo string) {
	start := time.Now()
	authenticator := getAuthenticator()
	sourceToTargetChanges, targetToSourceChanges, err := changelog.Changelog(authenticator, source, target, instance, manifestRepo)
	if err != nil {
		log.Infof("generateChangelog: error retrieving changelog between builds %s and %s on GoB instance: %s with manifest repository: %s\n%v\n",
			source, target, instance, manifestRepo, err)
		os.Exit(1)
	}
	if err := writeChangelogAsJSON(source, target, sourceToTargetChanges); err != nil {
		log.Infof("generateChangelog: error writing first changelog with source: %s and target: %s\n%v\n",
			source, target, err)
	}
	if err := writeChangelogAsJSON(target, source, targetToSourceChanges); err != nil {
		log.Infof("generateChangelog: Error writing second changelog with source: %s and target: %s\n%v\n",
			target, source, err)
	}
	log.Infof("Retrieved changelog in %s\n", time.Since(start))
}

func main() {
	var instance, manifestRepo string
	app := &cli.App{
		Name:  "changelog",
		Usage: "get commits between builds",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "instance",
				Value:       cosInstanceURL,
				Aliases:     []string{"i"},
				Usage:       "GoB `INSTANCE` to use as client",
				Destination: &instance,
			},
			&cli.StringFlag{
				Name:        "repo",
				Value:       defaultManifestRepo,
				Aliases:     []string{"r"},
				Usage:       "`REPO` containing Manifest file",
				Destination: &manifestRepo,
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() < 2 {
				return errors.New("Must specify source and target build number")
			}
			source := c.Args().Get(0)
			target := c.Args().Get(1)
			generateChangelog(source, target, instance, manifestRepo)
			return nil
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Infof("main: error running app with arguments: %v:\n%v", os.Args, err)
		os.Exit(1)
	}
}
