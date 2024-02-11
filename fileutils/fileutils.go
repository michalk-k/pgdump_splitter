package fileutils

import (
	"os"
	"path/filepath"
	"log"
   )

func CreateFile(filefullpath string) {

	_, err := os.Stat(filefullpath)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(filefullpath), 0770); err != nil {
			log.Fatal(err)
		}
		os.Create(filefullpath)

	}
}