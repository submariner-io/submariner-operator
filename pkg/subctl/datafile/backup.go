package datafile

import (
	"os"
	"strings"
	"time"
)

func BackupIfExists(filename string) (string, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return "", nil
	}
	now := time.Now()
	nowStr := strings.Replace(now.Format(time.RFC3339), ":", "_", -1)
	newFilename := filename + "." + nowStr
	return newFilename, os.Rename(filename, newFilename)
}
