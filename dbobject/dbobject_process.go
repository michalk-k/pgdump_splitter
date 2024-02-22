package dbobject

import (
	"bufio"
	"fmt"
	"os"
	fu "pgdump_splitter/fileutils"
	"regexp"
)

// Most outer processing function.
// It initializes a stream either from a file or pgdump, and processes it line by line.
func ProcessDump(args *Config) error {

	err := os.RemoveAll(args.Dest)
	if err != nil {
		return err
	}

	var scanner *bufio.Scanner
	var dbname string
	var clusterphase = true
	var curObj DbObject

	// Open the file or pipe
	if args.File != "" {

		fmt.Println("Loading dump data from a file: " + args.File)

		file, err := os.Open(args.File)
		if err != nil {
			return err
		}
		defer file.Close()

		// Create a scanner.
		scanner = bufio.NewScanner(file)

	} else {

		fmt.Println("Loading dump data from stdin (pipe)")

		// Check if anything is attached to stdin
		stat, _ := os.Stdin.Stat()
		if !((stat.Mode() & os.ModeCharDevice) == 0) {
			return fmt.Errorf("no data is being piped to stdin")
		}

		// Create a scanner.
		scanner = bufio.NewScanner(os.Stdin)
	}

	// Set the scanner to preserve original line endings
	scanner.Split(preserveNewlines)

	// Dynamically resize the buffer based on input size
	maxCapacity := args.BufS // Maximum capacity for the buffer
	buf := make([]byte, 0, bufio.MaxScanTokenSize)
	scanner.Buffer(buf, maxCapacity)

	rgx_conn := regexp.MustCompile(`^\\connect (.*)`)
	rgx_users := regexp.MustCompile(`^-- (User Configurations|Databases)[\s]*$`)
	rgx_dbdump := regexp.MustCompile(`^-- PostgreSQL database dump[\s]*(complete)?[\s]*$`)
	rgx_roles := regexp.MustCompile(`(^-- (?P<Type1>Roles|Role memberships)[\s]*$)|(^-- (?P<Type2>User Config) \".*\"[\s]*$)`)
	rgx_common := regexp.MustCompile(`^-- (Data for )?Name: (?P<Name>.*); Type: (?P<Type>.*); Schema: (?P<Schema>.*);`)

	lineno := 0
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

	// Optionally remove - subdirectory (if exists).
	// it contains roles data created from pgdumpall
	if args.MvRl {
		err := os.RemoveAll(args.Dest + "/-")
		if err != nil {
			return err
		}
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
