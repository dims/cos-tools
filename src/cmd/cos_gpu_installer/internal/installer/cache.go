package installer

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/golang/glog"
	"github.com/pkg/errors"

	"cos.googlesource.com/cos/tools.git/src/pkg/utils"
)

const (
	cacheFile        = ".cache"
	buildNumberKey   = "BUILD_ID"
	driverVersionKey = "DRIVER_VERSION"
	kernelOpenKey    = "KERNEL_OPEN"
	kernelOpenValue  = "Y"
)

// Cacher is to cache GPU driver installation info.
type Cacher struct {
	gpuInstallDir string
	buildNumber   string
	driverVersion string
}

// NewCacher returns an instance of Cacher.
func NewCacher(gpuInstallDir, buildNumber, driverVersion string) *Cacher {
	if gpuInstallDir != "" {
		return &Cacher{gpuInstallDir: gpuInstallDir, buildNumber: buildNumber, driverVersion: driverVersion}
	}

	return &Cacher{gpuInstallDir: gpuInstallDirContainer, buildNumber: buildNumber, driverVersion: driverVersion}
}

// Cache writes to fs about the information that a given GPU driver has been installed.
func (c *Cacher) Cache(kernelOpen bool) error {
	cachePath := filepath.Join(c.gpuInstallDir, cacheFile)
	f, err := os.Create(cachePath)
	defer f.Close()
	if err != nil {
		return errors.Wrapf(err, "Failed to create file %s", cachePath)
	}

	cacheMap := map[string]string{
		buildNumberKey:   c.buildNumber,
		driverVersionKey: c.driverVersion}

	if kernelOpen {
		cacheMap[kernelOpenKey] = kernelOpenValue
	}

	var cache string
	for k, v := range cacheMap {
		cache = cache + fmt.Sprintf("%s=%s\n", k, v)
	}

	if _, err = f.WriteString(cache); err != nil {
		return errors.Wrapf(err, "Failed to write to file %s", cachePath)
	}

	log.Info("Updated cached version as")
	for key, value := range cacheMap {
		log.Infof("%s=%s", key, value)
	}
	return nil
}

// IsCached returns a bool pair indicating whether a given GPU driver has been
// installed and if the installation contains open source kernel modules
func (c *Cacher) IsCached() (bool, bool, error) {
	cacheMap, err := utils.LoadEnvFromFile(c.gpuInstallDir, cacheFile)
	if err != nil {
		log.Infof("error: %v", err)
		return false, false, err
	}
	log.Infof("%v", cacheMap)

	return (c.buildNumber == cacheMap[buildNumberKey] &&
			c.driverVersion == cacheMap[driverVersionKey]),
		cacheMap[kernelOpenKey] == kernelOpenValue,
		nil
}
