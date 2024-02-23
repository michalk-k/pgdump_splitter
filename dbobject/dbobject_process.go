package dbobject

import (
	"bufio"
	"fmt"
	"os"
	fu "pgdump_splitter/fileutils"
	"regexp"
)

// Prepare input streams into scanner and pass it to for processing
func StartProcessing(args *Config) error {

	var err error
	var scanner *bufio.Scanner
	var finalizeFunc func()

	fmt.Println("Destination location: " + args.Dest)
	if err = os.RemoveAll(args.Dest); err != nil {
		return err
	}

	scanner, finalizeFunc, err = GetScannerWithData(args)
	defer finalizeFunc()

	if err != nil {
		return err
	}

	if err = ProcessStream(args, scanner); err != nil {
		return err
	}

	// Optionally remove - subdirectory (if exists).
	// it contains roles data created from pgdumpall
	if args.MvRl {
		if err = os.RemoveAll(args.Dest + "/-"); err != nil {
			return err
		}
	}

	return nil

}

// Creates scanner object
// The scanner takes data from the stdin pipe or from the input file
// depending on passed arguments
func GetScannerWithData(args *Config) (*bufio.Scanner, func(), error) {
	var scanner *bufio.Scanner
	var err error
	var closeStreamFn func()

	// Open the file or pipe
	if args.File != "" {

		fmt.Println("Loading dump data from a file: " + args.File)
		scanner, closeStreamFn, err = getScannerFromFile(args.File)
		if err != nil {
			return nil, closeStreamFn, err
		}

	} else {

		fmt.Println("Loading dump data from stdin (pipe)")
		if scanner, err = getScannerFromPipe(); err != nil {
			return nil, closeStreamFn, err
		}

	}

	// Set the scanner to preserve original line endings
	scanner.Split(preserveNewlines)

	// Dynamically resize the buffer based on input size
	maxCapacity := args.BufS // Maximum capacity for the buffer
	buf := make([]byte, 0, bufio.MaxScanTokenSize)
	scanner.Buffer(buf, maxCapacity)

	return scanner, closeStreamFn, nil
}

func getScannerFromFile(filename string) (*bufio.Scanner, func(), error) {

	var file *os.File
	var err error
	var closeFileFn = func() {
		if file != nil {
			defer file.Close()
		}
	}

	if file, err = os.Open(filename); err != nil {
		return nil, closeFileFn, err
	}

	// Create a scanner.
	scanner := bufio.NewScanner(file)

	return scanner, closeFileFn, nil
}

func getScannerFromPipe() (*bufio.Scanner, error) {

	// Check if anything is attached to stdin
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}

	if !((stat.Mode() & os.ModeCharDevice) == 0) {
		return nil, fmt.Errorf("no data piped to stdin")
	}

	// Create a scanner.
	return bufio.NewScanner(os.Stdin), nil

}

// Most outer processing function.
// It initializes a stream either from a file or pgdump, and processes it line by line.
func ProcessStream(args *Config, scanner *bufio.Scanner) error {

	rgx_conn := regexp.MustCompile(`^\\connect (.*)`)
	rgx_users := regexp.MustCompile(`^-- (User Configurations|Databases)[\s]*$`)
	rgx_dbdump := regexp.MustCompile(`^-- PostgreSQL database dump[\s]*(complete)?[\s]*$`)
	rgx_roles := regexp.MustCompile(`(^-- (?P<Type1>Roles|Role memberships)[\s]*$)|(^-- (?P<Type2>User Config) \".*\"[\s]*$)`)
	rgx_common := regexp.MustCompile(`^-- (Data for )?Name: (?P<Name>.*); Type: (?P<Type>.*); Schema: (?P<Schema>.*);`)

	lineno := 0

	var dbname string
	var clusterphase = true
	var curObj DbObject
	var err error

	// Iterate over each line
	for scanner.Scan() {
		lineno = lineno + 1
		line := scanner.Text()

		// Reacts on row:
		// \connect database_name
		matches := rgx_conn.FindStringSubmatch(line)
		if len(matches) > 0 {

			err = Save(&curObj, args.ExDb)
			if err != nil {
				return err
			}

			curObj = DbObject{}
			curObj.Paths = DbObjPath{}
			dbname = matches[1]
			continue
		}

		if clusterphase {

			// Reacts on rows:
			// -- User Configurations
			// -- User Databases
			matches = rgx_users.FindStringSubmatch(line)
			if len(matches) > 0 {

				if matches[1] == "Databases" {
					clusterphase = false
				}

				err = Save(&curObj, args.ExDb)
				if err != nil {
					return err
				}

				continue
			}
		}

		// Reacts on rows:
		// -- PostgreSQL database dump
		// -- PostgreSQL database dump complete
		//
		// When completion of database is recognized, we copy user roles into it (if enabled by configuration)
		matches = rgx_dbdump.FindStringSubmatch(line)
		if len(matches) > 0 {

			if matches[1] == "complete" {
				if args.MvRl {
					RelocateClusterRoles(args.Dest+"-", args.Dest+dbname+"/-")
				}
			} else {
				clusterphase = false
			}

			err = Save(&curObj, args.ExDb)
			if err != nil {
				return err
			}

			curObj = DbObject{}
			curObj.Paths = DbObjPath{}
			continue
		}

		// Reacts on rows:
		// -- Roles
		// -- Role memberships
		// -- User Config "user_name"
		// Starts collecting data for obj type ROLE
		if clusterphase {

			matches = rgx_roles.FindStringSubmatch(line)
			if len(matches) > 0 {

				err = Save(&curObj, args.ExDb)
				if err != nil {
					return err
				}

				// Iterate over each match
				result := make(map[string]string)
				for i, name := range rgx_roles.SubexpNames() {
					if i != 0 && name != "" {
						result[name] = matches[i]
					}
				}

				var objtype string
				if result["Type1"] != "" {
					objtype = result["Type1"]
				} else if result["Type2"] != "" {
					objtype = result["Type2"]
				}

				curObj = DbObject{

					Name:     objtype,
					ObjType:  "ROLE",
					Schema:   "-",
					Database: dbname,
					DocuRgx:  args.Docu,
					Paths: DbObjPath{
						Rootpath:   args.Dest,
						IsCustom:   args.Mode == "custom",
						NoDbInPath: args.NoDb,
					},
				}

				continue
			}
		}

		// Reacts on rows:
		// -- Name: some name; Type: some type; Schema: some_schema;
		// -- Data for Name: some name; Type: some type; Schema: some_schema;
		//
		// Starts collecting data for obj type ROLE
		// If type is TABLE DATA, data are not being added to the object (for performance reasons)
		matches = rgx_common.FindStringSubmatch(line)

		if len(matches) > 0 {

			err = Save(&curObj, args.ExDb)
			if err != nil {
				return err
			}

			// Iterate over each match
			result := make(map[string]string)
			for i, name := range rgx_common.SubexpNames() {
				if i != 0 && name != "" {
					result[name] = matches[i]
				}
			}
			curObj = DbObject{

				Name:     result["Name"],
				ObjType:  result["Type"],
				Schema:   result["Schema"],
				Database: dbname,
				DocuRgx:  args.Docu,
				Paths: DbObjPath{
					Rootpath:   args.Dest,
					IsCustom:   args.Mode == "custom",
					NoDbInPath: args.NoDb,
				},
			}

			continue
		}

		if curObj.ObjType != "" && curObj.ObjType != "TABLE DATA" {
			curObj.Content = curObj.Content + line
		}

	}

	// save the last row remaining in the buffer
	err = Save(&curObj, args.ExDb)
	if err != nil {
		return err
	}

	// Check for any errors that may have occurred during scanning
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("%s. Fails on line: %d.\nConsider setting buffer size to higher value", err.Error(), lineno+1)
	}

	return nil
}

// custom function for the Scanner.
// While default Scanner function strips EOL characters from the stream, this version maintains them untouched.
func preserveNewlines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			// Include the newline character in the token
			return i + 1, data[0 : i+1], nil
		}
	}
	// If at end of file and no newline found, return the entire data
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func Save(dbo *DbObject, exdb_rgx string) error {

	if exdb_rgx != "" {
		rgx, err := regexp.Compile(exdb_rgx)
		if err != nil {
			return fmt.Errorf("invalid regular expression for excluding databases")
		}

		matches := rgx.FindStringSubmatch(dbo.Database)
		if len(matches) > 0 {
			return nil
		}
	}

	if dbo.Content != "" {
		return dbo.StoreObj()
	}

	return nil
}

func RelocateClusterRoles(srcDir string, destDir string) error {
	if err := fu.CopyDir(srcDir, destDir); err != nil {
		return err
	}

	return nil
}
