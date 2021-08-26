# Release process

## Validating release config - TODO (rnv) - convert to presubmit cloud build
task

After making changes to the release
[state](https://cos.googlesource.com/cos/tools/+/refs/heads/master/release/release-versions.yaml)
the validity of the changes can be checked by running the following command from
the root of the repo:

`gcloud builds submit --config=presubmit.yaml`

where presubmit.yaml could look like the following:

```
steps:
- name: 'gcr.io/google.com/cloudsdktool/cloud-sdk:latest'
  entrypoint: 'bash'
  args: ['-c',
  'pip3 install -r release/requirements.txt && python3 release/release.py --verify'
  ]
```
