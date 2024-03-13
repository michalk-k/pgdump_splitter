package dbobject

import (
	"bufio"
	"fmt"
	"os"
	"pgdump_splitter/output"
)

type ScanerProvider struct {
	file    *os.File
	scanner *bufio.Scanner
}

func (obj *ScanerProvider) Finalize() {
	if obj.file != nil {
		defer obj.file.Close()
	}
}

// Creates scanner object
// The scanner takes data from the stdin pipe or from the input file
// depending on passed arguments
func (obj *ScanerProvider) CreateScanner(args *Config) error {
	var err error

	// Open the file or pipe
	if args.File != "" {

		output.Println("Loading dump data from a file: " + args.File)
		if err = obj.getScannerFromFile(args.File); err != nil {
			return err
		}

	} else {

		output.Println("Loading dump data from stdin (pipe)")
		if err = obj.getScannerFromPipe(); err != nil {
			return err
		}

	}

	// Set the scanner to preserve original line endings
	obj.scanner.Split(preserveNewlines)

	// Dynamically resize the buffer based on input size
	maxCapacity := args.BufS // Maximum capacity for the buffer
	buf := make([]byte, 0, bufio.MaxScanTokenSize)
	obj.scanner.Buffer(buf, maxCapacity)

	return nil
}

func (obj *ScanerProvider) getScannerFromFile(filename string) error {

	var err error

	if obj.file, err = os.Open(filename); err != nil {
		return err
	}

	// Create a scanner.
	obj.scanner = bufio.NewScanner(obj.file)

	return nil
}

func (obj *ScanerProvider) getScannerFromPipe() error {

	// Check if anything is attached to stdin
	stat, err := os.Stdin.Stat()
	if err != nil {
		return err
	}

	if !((stat.Mode() & os.ModeCharDevice) == 0) {
		return fmt.Errorf("no data piped to stdin")
	}

	// Create a scanner.
	obj.scanner = bufio.NewScanner(os.Stdin)

	return nil

}
