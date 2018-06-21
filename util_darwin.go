package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

func makeConfigISO(metadataISOFile, metaDataFile, userDataFile string) error {
	dir, err := ioutil.TempDir("", "ci-tmp-data")
	defer os.RemoveAll(dir) // clean up
	if err != nil {
		return err
	}
	tmpMetaData := filepath.Join(dir, "meta-data")
	if err := copyFile(metaDataFile, tmpMetaData); err != nil {
		return err
	}
	tmpUserData := filepath.Join(dir, "user-data")
	if err := copyFile(userDataFile, tmpUserData); err != nil {
		return err
	}
	cmd := exec.Command("mkisofs", "-output", metadataISOFile, "-volid", "cidata", "-joliet", "-rock", tmpUserData, tmpMetaData)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s, %s, is 'mkisofs' installed? Install it with 'brew install cdrtools'", stderr.String(), err)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}
