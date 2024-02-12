package fileutils

import (
	"log"
	"os"
	"path/filepath"
)

// Function checks if file exists, creating it if not exists, including the whole path needed for that fie
// The function returns true if file has been created, or false if file already had existed.
func CreateFile(filefullpath string) bool {

	_, err := os.Stat(filefullpath)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(filefullpath), 0770); err != nil {
			log.Fatal(err)
		}
		os.Create(filefullpath)
		return true
	}

	return false
}
