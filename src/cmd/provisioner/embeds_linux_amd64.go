// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import _ "embed"

//go:embed _handle_disk_layout_amd64.bin
var handleDiskLayoutBin []byte

//go:embed _veritysetup_amd64.img
var veritySetupImage []byte

//go:embed docker-credential-gcr_amd64
var dockerCredentialGCR []byte
