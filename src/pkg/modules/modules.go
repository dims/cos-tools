// Package modules provides fucntionality to install and sign Linux kernel modules.
package modules

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/golang/glog"
	"github.com/pkg/errors"

	"pkg/utils"
)

const (
	// PKEYIDPKCS7 is a constant defined in https://github.com/torvalds/linux/blob/master/scripts/sign-file.c
	PKEYIDPKCS7 = byte(2)
	// magicNumber is a constant defined in https://github.com/torvalds/linux/blob/master/scripts/sign-file.c
	magicNumber = "~Module signature appended~\n"
)

var (
	execCommand = exec.Command
)

// LoadModule loads a given kernel module to kernel.
func LoadModule(moduleName, modulePath string) error {
	loaded, err := isModuleLoaded(moduleName)
	if err != nil {
		return errors.Wrapf(err, "failed to load module %s (%s)", moduleName, modulePath)
	}
	if loaded {
		return nil
	}
	if err := loadModule(modulePath); err != nil {
		return errors.Wrapf(err, "failed to load module %s (%s)", moduleName, modulePath)
	}
	return nil
}

// UpdateHostLdCache updates the ld cache on host.
func UpdateHostLdCache(hostRootDir, moduleLibDir string) error {
	log.Info("Updating host's ld cache")
	ldPath := filepath.Join(hostRootDir, "/etc/ld.so.conf")
	f, err := os.OpenFile(ldPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to open %s", ldPath)
	}
	defer f.Close()

	if _, err := f.WriteString(moduleLibDir); err != nil {
		return errors.Wrapf(err, "failed to write \"%s\" to %s", moduleLibDir, ldPath)
	}

	if err := execCommand("ldconfig", "-r", hostRootDir).Run(); err != nil {
		return errors.Wrapf(err, "failed to run `ldconfig -r %s`", hostRootDir)
	}

	return nil
}

// LoadPublicKey loads the given public key to the secondary keyring.
func LoadPublicKey(keyName, keyPath string) error {
	log.Infof("Loading %s to secondary system keyring", keyName)

	keyBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return errors.Wrapf(err, "failed to read key %s", keyPath)
	}

	cmd := execCommand("/bin/keyctl", "padd", "asymmetric", keyName, "%keyring:.secondary_trusted_keys")
	cmd.Stdin = bytes.NewBuffer(keyBytes)
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "failed to load %s to system keyring", keyName)
	}
	log.Infof("Successfully load key %s into secondary system keyring.", keyName)
	return nil
}

// AppendSignature appends a raw PKCS#7 signature to the end of a given kernel module.
// This is basically the Go implementation of `scripts/sign-file -s` in Linux upstream.
func AppendSignature(outfilePath, modulefilePath, sigfilePath string) error {
	tempFile, err := ioutil.TempFile("", "tempFile")
	if err != nil {
		return errors.Wrap(err, "failed to create temp file")
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Copy bytes of kernel module into the temp file.
	modulefile, err := os.Open(modulefilePath)
	if err != nil {
		return errors.Wrapf(err, "failed to open file %s", modulefilePath)
	}
	defer modulefile.Close()

	_, err = io.Copy(tempFile, modulefile)
	if err != nil {
		return errors.Wrap(err, "failed to copy file")
	}

	// Append bytes of module signature into the temp file.
	sigfile, err := os.Open(sigfilePath)
	if err != nil {
		return errors.Wrapf(err, "failed to open file %s", sigfilePath)
	}
	defer sigfile.Close()

	sigSize, err := io.Copy(tempFile, sigfile)
	if err != nil {
		return errors.Wrap(err, "failed to copy file")
	}

	// Append the marker and the PKCS#7 message.
	// moduleSignature is the struct module_signature defined in
	// https://github.com/torvalds/linux/blob/master/scripts/sign-file.c
	moduleSignature := [12]byte{}
	// moduleSignature[2] is the id_type of struct module_signature
	moduleSignature[2] = PKEYIDPKCS7
	// moduleSignature[8:12] is the sig_len of struct module_signature.
	// Using BigEndian as the sig_len should be in network byte order.
	binary.BigEndian.PutUint32(moduleSignature[8:12], uint32(sigSize))
	_, err = tempFile.Write(moduleSignature[:])
	if err != nil {
		return errors.Wrapf(err, "failed to write to file %s", tempFile.Name())
	}

	_, err = tempFile.Write([]byte(magicNumber))
	if err != nil {
		return errors.Wrapf(err, "failed to write to file %s", tempFile.Name())
	}

	if err := tempFile.Close(); err != nil {
		return errors.Wrapf(err, "failed to close file %s", tempFile.Name())
	}

	// Finally, move the outfile to specified location.
	// It overwrites the original module file if we are appending in place.
	if err := utils.MoveFile(tempFile.Name(), outfilePath); err != nil {
		return errors.Wrapf(err, "failed to rename file from %s to %s", tempFile.Name(), outfilePath)
	}

	return nil
}

func isModuleLoaded(moduleName string) (bool, error) {
	out, err := execCommand("lsmod").Output()
	if err != nil {
		return false, errors.Wrap(err, "failed to run command `lsmod`")
	}

	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == moduleName {
			return true, nil
		}
	}
	return false, nil
}

func loadModule(modulePath string) error {
	if err := execCommand("insmod", modulePath).Run(); err != nil {
		return errors.Wrapf(err, "failed to run command `insmod %s`", modulePath)
	}
	return nil
}
