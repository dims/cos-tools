# COS Ova Converter

COS Ova Converter is a simple interface to convert the OVA images to GCE image in
a GCP project and also exports the GCE image to OVA. It is available as a Docker image.

The main motivation is to provide a simple tool for handling the OVA images
during the COS preloading process using the COS Customizer as it exclusively
deals with the GCE images.

## How to get started

### Compile COS OVA Converter

``` shell
make
```
will build the COS OVA Converter application

### Try COS OVA Converter

COS OVA Converter is available as a docker image (cos_ova_converter). It can be
run as one of the steps in the Cloud build workflow for converting the OVA images
to GCE and back to OVA from GCE images.


To convert the OVA to GCE image,

```shell
steps:
- name: 'cos-ova-converter'
  args: ['to-gce',
         '-image-name=cos-ova-converted-image',
         '-image-project=${PROJECT_ID}',
         '-gcs-bucket=${PROJECT_ID}_cloudbuild',
         '-input-url=gs://sample-gcs/cos.ova']
```

This will download the image specified in the `input-url` and creates and image with
name `image-name` in `image-project`. `gcs-bucket` here is a workspace.

To convert the OVA from the GCE image

```shell
steps:
- name: 'cos-ova-converter'
  args: ['from-gce',
         '-source-image=cos-preloaded-image',
         '-image-project=${PROJECT_ID}',
         '-gcs-bucket=${PROJECT_ID}_cloudbuild',
         '-destination-path=gs://sample-gcs/cos.ova']
```

This will export the image with name `source-image` in `image-project` to `destination-path`
as OVA. `gcs-bucket` here is a workspace.
