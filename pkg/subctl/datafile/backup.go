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
	nowStr := strings.ReplaceAll(now.Format(time.RFC3339), ":", "_")
	newFilename := filename + "." + nowStr
	return newFilename, os.Rename(filename, newFilename)
}
