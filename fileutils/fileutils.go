package fileutils

import (
	"io"
	"os"
	"path/filepath"
)

// Function checks if file exists, creating it if not exists, including the whole path needed for that fie
// The function returns true if file has been created, or false if file already had existed.
func CreateFile(filefullpath string) (bool, error) {

	_, err := os.Stat(filefullpath)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(filefullpath), 0770); err != nil {
			return false, err
		}
		os.Create(filefullpath)
		return true, nil
	}

	return false, nil
}

// CopyDir copies the contents of a source directory to a destination directory recursively.
func CopyDir(src, dest string) error {
	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(dest, os.ModePerm); err != nil {
		return err
	}

	// Open source directory
	dir, err := os.Open(src)
	if err != nil {
		return err
	}
	defer dir.Close()

	// Read the contents of the source directory
	fileInfos, err := dir.Readdir(-1)
	if err != nil {
		return err
	}

	// Iterate over the files in the source directory
	for _, fileInfo := range fileInfos {
		srcFilePath := filepath.Join(src, fileInfo.Name())
		destFilePath := filepath.Join(dest, fileInfo.Name())

		if fileInfo.IsDir() {
			// Recursively copy subdirectories
			if err := CopyDir(srcFilePath, destFilePath); err != nil {
				return err
			}
		} else {
			// Copy regular files
			if err := CopyFile(srcFilePath, destFilePath); err != nil {
				return err
			}
		}
	}

	return nil
}

// CopyFile copies a file from source to destination.
func CopyFile(src, dest string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create destination file
	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy data from source to destination
	_, err = io.Copy(destFile, srcFile)
	return err
}

// Removes all files and directories from given location.
// The most parent directory (given in path) remains untouched
func WipeDir(dir string) error {

	// Open the directory
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()

	// Read the directory contents
	files, err := d.Readdir(-1)
	if err != nil {
		return err
	}

	// Remove each file or directory in the directory
	for _, file := range files {

		fullPath := filepath.Join(dir, file.Name())

		if file.IsDir() {
			// Remove the subdirectory incl its content
			if err := os.RemoveAll(fullPath); err != nil {
				return err
			}

		} else {
			// Remove the file
			if err := os.Remove(fullPath); err != nil {
				return err
			}
		}
	}

	return nil
}
