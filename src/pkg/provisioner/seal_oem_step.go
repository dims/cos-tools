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

package provisioner

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"cos.googlesource.com/cos/tools.git/src/pkg/tools"
	"cos.googlesource.com/cos/tools.git/src/pkg/utils"
)

type SealOEMStep struct{}

func (s *SealOEMStep) run(ctx context.Context, runState *state, deps *stepDeps) error {
	log.Println("Sealing the OEM partition with dm-verity")
	veritysetupImgPath := filepath.Join(runState.dir, "veritysetup.img")
	if _, err := os.Stat(veritysetupImgPath); os.IsNotExist(err) {
		if err := utils.CopyFile(deps.VeritySetupImage, veritysetupImgPath); err != nil {
			return err
		}
	}
	if err := tools.SealOEMPartition(veritysetupImgPath, runState.data.Config.BootDisk.OEMFSSize4K); err != nil {
		return err
	}
	if err := tools.DisableSystemdService("update-engine.service"); err != nil {
		return err
	}
	if err := tools.DisableSystemdService("usr-share-oem.mount"); err != nil {
		return err
	}
	log.Println("Done sealing the OEM partition with dm-verity")
	return nil
}
