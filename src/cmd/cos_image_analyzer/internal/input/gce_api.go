package input

import (
	"bytes"
	"encoding/json"

	// "fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

const timeOut = "7200"
const imageFormat = "vmdk"
const name = "gcr.io/compute-image-tools/gce_vm_image_export:release"

type Steps struct {
	Args [6]string `json:"args"`
	Name string    `json:"name"`
	Env  [1]string `json:"env"`
}

type GcePayload struct {
	Timeout string    `json:"timeout"`
	Steps   [1]Steps  `json:"steps"`
	Tags    [2]string `json:"tags"`
}

// gceExport calls the cloud build REST api that exports a public compute
// image to a specfic GCS bucket.
// Input:
//   (string) projectID - project ID of the cloud project holding the image
//   (string) bucket - name of the GCS bucket holding the COS Image
//   (string) image - name of the source image to be exported
// Output: None
func gceExport(projectID, bucket, image string) error {
	// API Variables
	gceURL := "https://cloudbuild.googleapis.com/v1/projects/" + projectID + "/builds"
	destURI := "gs://" + bucket + "/" + image + "." + imageFormat
	args := [6]string{"-oauth=/usr/local/google/home/acueva/cos-googlesource/tools/src/cmd/cos_image_analyzer/internal/utilities/oauth.json", "-timeout=" + timeOut, "-source_image=" + image, "-client_id=api", "-format=" + imageFormat, "-destination_uri=" + destURI}
	env := [1]string{"BUILD_ID=$BUILD_ID"}
	tags := [2]string{"gce-daisy", "gce-daisy-image-export"}

	// Build API bodies
	steps := [1]Steps{Steps{Args: args, Name: name, Env: env}}
	payload := &GcePayload{
		Timeout: timeOut,
		Steps:   steps,
		Tags:    tags}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	log.Println(string(requestBody))

	resp, err := http.Post(gceURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	log.Println(string(body))
	return nil
}

// GetCosImage calls the cloud build api to export a public COS image to a
// a GCS bucket and then calls GetGcsImage() to download that image from GCS.
// ADC is used for authorization.
// Input:
//   (string) cosCloudPath - The "projectID/gcs-bucket/image" path of the
//   source image to be exported
// Output:
//   (string) imageDir - Path to the mounted directory of the  COS Image
func GetCosImage(cosCloudPath string) (string, error) {
	spiltPath := strings.Split(cosCloudPath, "/")
	projectID, bucket, image := spiltPath[0], spiltPath[1], spiltPath[2]

	if err := gceExport(projectID, bucket, image); err != nil {
		return "", err
	}

	gcsPath := filepath.Join(bucket, image)
	imageDir, err := GetGcsImage(gcsPath, 1)
	if err != nil {
		return "", err
	}

	return imageDir, nil
}
