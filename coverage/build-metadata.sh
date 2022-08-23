#!/bin/bash
# Bash script to build metadata.json, for NorthStar upload
cat <<EOF > metadata.json
{
  "host": "cos",
  "project": "codesearch",
  "trace_type": "GO_COV",
  "trim_prefix": "cos.googlesource.com/cos/tools.git",
  "git_project": "cos/tools",
  "commit_id": "${COMMIT_SHA}",
  "ref": "${REF}",
  "source": "cos-tools-infra:master",
  "owner": "${OWNER}",
  "bug_component": "242766552"
}
EOF
