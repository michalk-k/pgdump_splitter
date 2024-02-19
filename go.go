package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"pgdump_splitter/dbobject"
	fu "pgdump_splitter/fileutils"
	"regexp"
)

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

func Save(dbo *dbobject.DbObject, exdb_rgx string) {

	if exdb_rgx != "" {
		rgx := regexp.MustCompile(exdb_rgx)
		matches := rgx.FindStringSubmatch(dbo.Database)
		if len(matches) > 0 {
			return
		}
	}

	if dbo.Content != "" {
		dbo.StoreObj()
	}

}

func RelocateClusterRoles(srcDir string, destDir string) {
	if err := fu.CopyDir(srcDir, destDir); err != nil {
		fmt.Println("Error copying directory:", err)
		return
	}
}

// Most outer processing function.
// It initializes a stream either from a file or pgdump, and processes it line by line.
func ProcessDump(args *Args) {

	err := os.RemoveAll(args.Dest)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	var scanner *bufio.Scanner
	var dbname string
	var clusterphase = true
	var curObj dbobject.DbObject

	// Open the file or pipe
	if args.File != "" {

		fmt.Println("Loading dump data from a file: " + args.File)

		file, err := os.Open(args.File)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer file.Close()
		scanner = bufio.NewScanner(file)
	} else {

		fmt.Println("Loading dump data from stdin (pipe)")
		// Check if anything is attached to stdin
		stat, _ := os.Stdin.Stat()
		if !((stat.Mode() & os.ModeCharDevice) == 0) {
			fmt.Println("No data is being piped to stdin")
			return
		}

		scanner = bufio.NewScanner(os.Stdin)
	}

	// Create a scanner.
	// Set the scanner to preserve original line endings

	scanner.Split(preserveNewlines)

	// Iterate over each line
	for scanner.Scan() {

		line := scanner.Text()

		if !clusterphase && dbname == "" {
			//		dbname = args.DNme
		}

		// Reacts on row:
		// \connect database_name
		rgx := regexp.MustCompile("^\\\\connect (.*)")
		matches := rgx.FindStringSubmatch(line)
		if len(matches) > 0 {
			Save(&curObj, args.ExDb)
			curObj = dbobject.DbObject{}
			dbname = matches[1]
			continue
		}

		if clusterphase {

			// Reacts on rows:
			// -- User Configurations
			// -- User Databases
			rgx = regexp.MustCompile("^-- (User Configurations|Databases)[\\s]*$")
			matches = rgx.FindStringSubmatch(line)
			if len(matches) > 0 {

				if matches[1] == "Databases" {
					clusterphase = false
				}

				Save(&curObj, args.ExDb)
				continue
			}
		}

		// Reacts on rows:
		// -- PostgreSQL database dump
		// -- PostgreSQL database dump complete
		//
		// When completion of database is recognized, we copy user roles into it (if enabled by configuration)
		rgx = regexp.MustCompile("^-- PostgreSQL database dump[\\s]*(complete)?[\\s]*$")
		matches = rgx.FindStringSubmatch(line)
		if len(matches) > 0 {

			if matches[1] == "complete" {
				if args.MvRl {
					RelocateClusterRoles(args.Dest+"-", args.Dest+dbname+"/-")
				}
			} else {
				clusterphase = false
			}

			Save(&curObj, args.ExDb)
			curObj = dbobject.DbObject{}
			continue
		}

		// Reacts on rows:
		// -- Roles
		// -- Role memberships
		// -- User Config "user_name"
		// Starts collecting data for obj type ROLE
		if clusterphase {
			rgx = regexp.MustCompile("(^-- (?P<Type1>Roles|Role memberships)[\\s]*$)|(^-- (?P<Type2>User Config) \".*\"[\\s]*$)")
			matches = rgx.FindStringSubmatch(line)
			if len(matches) > 0 {

				Save(&curObj, args.ExDb)

				// Iterate over each match
				result := make(map[string]string)
				for i, name := range rgx.SubexpNames() {
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

				curObj = dbobject.DbObject{
					Rootpath:   args.Dest,
					Name:       objtype,
					ObjType:    "ROLE",
					Schema:     "-",
					Database:   dbname,
					IsCustom:   args.Mode == "custom",
					NoDbInPath: args.NoDb,
					DocuRgx:    args.Docu,
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
		rgx = regexp.MustCompile("^-- (Data for )?Name: (?P<Name>.*); Type: (?P<Type>.*); Schema: (?P<Schema>.*);")
		matches = rgx.FindStringSubmatch(line)

		if len(matches) > 0 {

			Save(&curObj, args.ExDb)

			// Iterate over each match
			result := make(map[string]string)
			for i, name := range rgx.SubexpNames() {
				if i != 0 && name != "" {
					result[name] = matches[i]
				}
			}
			curObj = dbobject.DbObject{
				Rootpath:   args.Dest,
				Name:       result["Name"],
				ObjType:    result["Type"],
				Schema:     result["Schema"],
				Database:   dbname,
				IsCustom:   args.Mode == "custom",
				NoDbInPath: args.NoDb,
				DocuRgx:    args.Docu,
			}

			continue
		}

		if curObj.ObjType != "" && curObj.ObjType != "TABLE DATA" {
			curObj.Content = curObj.Content + line
		}

	}

	// save the last row remaining in the buffer
	Save(&curObj, args.ExDb)

	// Optionally remove - subdirectory (if exists).
	// it contains roles data created from pgdumpall
	if args.MvRl {
		err := os.RemoveAll(args.Dest + "/-")
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
	}

	// Check for any errors that may have occurred during scanning
	if err := scanner.Err(); err != nil {
		fmt.Println("Error:", err)
	}

}

// Structure handling program runtime configuration.
// Values are comming from command line arguments.
type Args struct {
	Mode string
	Dest string
	NoDb bool
	ExDb string
	MvRl bool
	File string
	Docu string
}

func main() {

	var args Args
	//var file string
	//file = "/home/kozusznikm/gitrepo/pgdump_splitter_orig/dumpall.sql"

	flag.StringVar(&args.File, "f", "", "path to dump generated by pg_dump or pg_dumpall. If omited the program will expect data on stdin via system pipe.")
	flag.StringVar(&args.Mode, "mode", "custom", "The mode of dumping db objects. origin - for file organization as present in the database dump. custom - reorganizes db objects storing related ones into single file")
	flag.StringVar(&args.Dest, "dst", "", "Location where structures will be dumped to")
	flag.BoolVar(&args.NoDb, "ndb", false, "No db name in destination path. It should not be set to true if multiple databases are dumped at once")
	flag.StringVar(&args.ExDb, "exdb", "^(template|postgres)", "Regular expression pattern allowing to skip extraction of matching databases. Usefull in case of processing dump files. In case of using a pipe from pg_dumpall, exclude them using pd_dumpall switch.")
	flag.BoolVar(&args.MvRl, "mc", true, "Move dump of roles into each database subdirectory")
	flag.StringVar(&args.Docu, "docu", "/\\*DOCU(.*)DOCU\\*/", "Move dump of roles into each database subdirectory.")

	flag.Parse()

	if !(args.Mode == "" || args.Mode == "custom" || args.Mode == "origin") {
		fmt.Println("Invalid value passed to `mode` modifier")
		flag.PrintDefaults()
		return
	}

	ProcessDump(&args)
	// Print the output
	fmt.Println("Finished")
}
