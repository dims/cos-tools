// Copyright 2021 Google Inc. All Rights Reserved.
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

package gcs

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"cloud.google.com/go/storage"
)

const schemeGCS = "gs"

// DownloadGCSObject downloads the object at inputURL and saves it at destinationPath
func DownloadGCSObject(ctx context.Context,
	gcsClient *storage.Client, inputURL, destinationPath string) error {
	gcsBucket, name, err := getGCSVariables(inputURL)
	if err != nil {
		return err
	}
	r, err := gcsClient.Bucket(gcsBucket).Object(name).NewReader(ctx)
	if err != nil {
		return err
	}
	defer r.Close()

	f, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("error copying file from gcs bucket: %v", err)
	}
	return nil
}

// UploadGCSObject uploads an object at inputPath to destination URL
func UploadGCSObject(ctx context.Context,
	gcsClient *storage.Client, inputPath, destinationURL string) error {

	gcsBucket, name, err := getGCSVariables(destinationURL)
	if err != nil {
		return fmt.Errorf("error parsing destination URL: %v", err)
	}
	fileReader, err := os.Open(inputPath)
	if err != nil {
		return err
	}

	w := gcsClient.Bucket(gcsBucket).Object(name).NewWriter(ctx)
	defer w.Close()

	if _, err := io.Copy(w, fileReader); err != nil {
		return err
	}
	return nil
}

// DeleteGCSObject deletes an object at the input URL
func DeleteGCSObject(ctx context.Context,
	gcsClient *storage.Client, inputURL string) error {
	gcsBucket, name, err := getGCSVariables(inputURL)
	if err != nil {
		return fmt.Errorf("error parsing input URL: %v", err)
	}
	return gcsClient.Bucket(gcsBucket).Object(name).Delete(ctx)
}

// Returns the getGCSVariables(GCSBucket, GCSPath, fileName) based on the input.
func getGCSVariables(gcsPath string) (string, string, error) {
	url, err := url.Parse(gcsPath)
	if err != nil || url.Scheme != schemeGCS {
		return "", "", fmt.Errorf("error parsing the input GCS path: %s", gcsPath)
	}
	// url.EscapedPath returns with the leading /.
	return url.Hostname(), strings.TrimLeft(url.EscapedPath(), "/"), nil
}
