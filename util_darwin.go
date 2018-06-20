package main

import (
	"bytes"
	"fmt"
	"os/exec"
)

func makeConfigISO(metadataISOFile, metaDataFile, userDataFile string) error {
	cmd := exec.Command("mkisofs", "-output", metadataISOFile, "-volid", "cidata", "-joliet", "-rock", userDataFile, metaDataFile)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s, %s, is 'mkisofs' installed? Install it with 'brew install cdrtools'", stderr.String(), err)
	}
	return nil
}
