"""
Release script invoked from git trigger upon submission of changes to release-versions.yaml config file to the cos/tools GoB repo

Parses contents of release-versions.yaml file and copies release candidates to gcr.io/cos-tools
"""

import sys
import yaml
import subprocess
import os

_SBOM_TAG = "sbom"
_COS_GPU_INSTALLER_STAGING_NAME = "cos-gpu-installer"

def validate_config(release_config):
  for release_container in release_config:
    for key in ["staging_container_name", "release_container_name", "build_commit", "release_tags"]:
      assert key in release_container, "missing {} in entry {}".format(key, release_container)

def validate_src_gcr_path(path):
  # path format: gcr.io/cos-infra-prod
  return len(path) > len("gcr.io/") and path[:len("gcr.io/")] == "gcr.io/"

def validate_dst_gcr_path(path):
  # path format: us-docker.pkg.dev/cos-cloud/us.gcr.io
  path = path.split('/')
  return len(path) == 3 and len(path[0]) > len("docker.pkg.dev/") and len(path[1]) != 0 and path[2][-len("gcr.io"):] == "gcr.io"

def copy_container_image(src_bucket, dst_bucket, staging_container_name, release_container_name, build_tag, release_tags):
  assert validate_src_gcr_path(src_bucket), "cannot use address {}, only gcr.io/ addresses are supported".format(src_bucket)
  assert validate_dst_gcr_path(dst_bucket), "cannot use address {}, only <location>-docker.pkg.dev/<project-name>/<location(optional)>gcr.io/ addresses are supported".format(dst_bucket)

  src_path = os.path.join(src_bucket, staging_container_name)
  dst_path = os.path.join(dst_bucket, release_container_name)

  for release_tag in release_tags:
    subprocess.run(["gcloud", "container", "images", "add-tag", src_path + ":" + build_tag, dst_path + ":" + release_tag, "-q"])

# Add tag for generating and uploading SBOM for cos-gpu-installer via louhi workflow.
def add_tag_for_sbom(src_bucket, staging_container_name, release_container_name, build_tag):
  if staging_container_name != _COS_GPU_INSTALLER_STAGING_NAME:
    return

  assert validate_src_gcr_path(src_bucket), "cannot use address {}, only gcr.io/ addresses are supported".format(src_bucket)

  src_path = os.path.join(src_bucket, staging_container_name)
  dst_path = os.path.join(src_bucket, release_container_name)

  subprocess.run(["gcloud", "container", "images", "add-tag", src_path + ":" + build_tag, dst_path + ":" + _SBOM_TAG, "-q"])

def verify_and_release(src_bucket, dst_buckets, release):
  with open('release/release-versions.yaml', 'r') as file:
    try:
      release_config = yaml.safe_load(file)
      validate_config(release_config)

      if release:
        dst_buckets = dst_buckets.split('^')
        for release_container in release_config:
          staging_container_name = release_container["staging_container_name"]
          release_container_name = release_container["release_container_name"]
          build_tag = release_container["build_commit"]
          release_tags = release_container["release_tags"]
          for dst_bucket in dst_buckets:
            copy_container_image(src_bucket, dst_bucket, staging_container_name, release_container_name, build_tag, release_tags)
          add_tag_for_sbom(src_bucket, staging_container_name, release_container_name, build_tag)

    except yaml.YAMLError as ex:
      raise Exception("Invalid YAML config: %s" % str(ex))

def main():
  if len(sys.argv) == 2 and sys.argv[1] == "--verify":
    verify_and_release("", "", False)
  elif len(sys.argv) == 3:
    src_bucket = sys.argv[1]
    dst_buckets = sys.argv[2]

    verify_and_release(src_bucket, dst_buckets, True)
  else:
    sys.exit("sample use: ./release_script <source_gcr_path> <destination_gcr_paths> \n \
              example use: ./release_script gcr.io/cos-infra-prod us-docker.pkg.dev/cos-cloud/us.gcr.io^europe-docker.pkg.dev/cos-cloud/eu.gcr.io")

if __name__ == '__main__':
  main()
