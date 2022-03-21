#!/bin/sh

# Copyright 2022 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -eu

echo "Getting auth token."
AUTH_DATA="$(curl -s -f -m 10 "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token" -H "Metadata-Flavor: Google")"
R=$?
if [ ${R} -ne 0 ]; then
  echo "Getting auth token error, exited with status ${R}" >&2
  exit ${R}
fi

AUTH="$(echo "${AUTH_DATA}" \
| tr -d '{}' \
| sed 's/,/\n/g' \
| awk -F ':' '/access_token/ { print $2 }' \
| tr -d '"\n')"

if [ -z "${AUTH}" ]; then
  echo "Auth token not found in AUTH_DATA ${AUTH_DATA}" >&2
  exit 1
fi

echo "Getting instance project and zone."
PROJ_ZONE="$(curl -s -f -m 10 "http://metadata.google.internal/computeMetadata/v1/instance/zone" -H "Metadata-Flavor: Google")"

R=$?
if [ ${R} -ne 0 ]; then
  echo "Getting project and zone error, exited with status ${R}" >&2
  exit ${R}
fi

echo "Getting instance name."
INSTANCE_NAME="$(curl -s -f -m 10 "http://metadata.google.internal/computeMetadata/v1/instance/name" -H "Metadata-Flavor: Google")"
R=$?
if [ ${R} -ne 0 ]; then
  echo "Getting instance name error, exited with status ${R}" >&2
  exit ${R}
fi

# Save the output of the instance detail to a local destination
curl "https://www.googleapis.com/compute/v1/${PROJ_ZONE}/instances/${INSTANCE_NAME}" -H "Authorization":"Bearer ${AUTH}" --header 'Accept: application/json'   --compressed >  /var/lib/instance_info.json
