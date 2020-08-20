package utilities

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

const contextTimeOut = time.Second * 50
const base10 = 10

// GcsDowndload calls the GCS client api to download a specified object from
// a GCS bucket.
// Input:
//   (string) bucket - Name of the GCS bucket
//   (string) object - Name of the GCS object
//   (string) destDir - Destination for downloaded GCS object
//   (string) name - Name for the downloaded file
//   (bool) authenticate - Indicates whether the GCS client need to be authenticated.
//                         Use unauthenticated client if you only wish to access public data.
//                         Otherwise, ADC will be used for authorization.
// Output:
//   (string) downloadedFile - Path to downloaded GCS object
func GcsDowndload(bucket, object, destDir, name string, authenticate bool) (string, error) {
	// Call API to download GCS object into tempDir
	var client *storage.Client
	var err error

	ctx := context.Background()
	if authenticate {
		client, err = storage.NewClient(ctx)
	} else {
		client, err = storage.NewClient(ctx, option.WithoutAuthentication())
	}
	if err != nil {
		return "", fmt.Errorf("failed to create new Google Cloud Go storage client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, contextTimeOut)
	defer cancel()

	rc, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to read GCS bucket: %v, and GCS object: %v : %v", bucket, object, err)
	}
	defer rc.Close()

	downloadedFile, err := os.Create(filepath.Join(destDir, name))
	if err != nil {
		return "", fmt.Errorf("failed to create file %v/%v: %v", destDir, object, err)
	}
	defer downloadedFile.Close()

	bytesDownloaded, err := io.Copy(downloadedFile, rc)
	if err != nil {
		return "", fmt.Errorf("failed to copy object into %v file: %v", downloadedFile, err)
	}
	bytesStr := strconv.FormatInt(bytesDownloaded, base10)

	log.Print("GCS object: ", object, " downloaded from GCS bucket: ", bucket, ". Total bytes ", bytesStr)
	return downloadedFile.Name(), nil
}
