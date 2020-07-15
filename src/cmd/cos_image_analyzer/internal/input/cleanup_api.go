package input

import (
	"os"
	"os/exec"
)

// Cleanup is called to remove a mounted directory and its loop device
//   (string) mountDir - Active mount directory ready to close
//   (string) loopDevice - Active loop device ready to close
// Output: nil on success, else error
func Cleanup(mountDir, loopDevice string) error {
	_, err := exec.Command("sudo", "umount", mountDir).Output()
	if err != nil {
		return err
	}
	_, err1 := exec.Command("sudo", "losetup", "-d", loopDevice).Output()
	if err1 != nil {
		return err1
	}
	os.Remove(mountDir)
	return nil
}
