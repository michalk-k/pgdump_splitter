package dbobject

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	fu "pgdump_splitter/fileutils"
	"pgdump_splitter/output"
	"regexp"
)

var rgx_conn *regexp.Regexp
var rgx_restrict *regexp.Regexp
var rgx_users *regexp.Regexp
var rgx_dbdump *regexp.Regexp
var rgx_roles *regexp.Regexp
var rgx_common *regexp.Regexp
var rgx_ExclDb *regexp.Regexp
var rgx_WhiteListDb *regexp.Regexp

var rgx_ExcludeObjType *regexp.Regexp

func init() {
	rgx_conn = regexp.MustCompile(`^\\connect( -reuse-previous=on)? (("dbname='(.*?)'")|(.*))`)
	rgx_users = regexp.MustCompile(`^-- (User Configurations|Databases)[\s]*$`)
	rgx_dbdump = regexp.MustCompile(`^-- PostgreSQL database dump[\s]*(complete)?[\s]*$`)
	rgx_roles = regexp.MustCompile(`(^-- (?P<Type1>Roles|Role memberships)[\s]*$)|(^-- (?P<Type2>User Config) \".*\"[\s]*$)`)
	rgx_common = regexp.MustCompile(`^-- (Data for )?Name: "?(?P<Name>.*)"?; Type: (?P<Type>.*); Schema: "?(?P<Schema>.*)"?;`)

}

// Prepare input streams into scanner and pass it to for processing
func StartProcessing(args *Config) error {

	var err error
	var dataprov ScanerProvider

	output.Println("Destination location: " + args.Dest)
	output.Println(fmt.Sprintf("Clean destination location: %t", args.Cln))

	// Check regular expression before any processing
	if err := IsExclObjTypeOk(args.ExOT); err != nil {
		return err
	}

	// wipe destination directory if requested.
	// leaving data might result in appending DDLs to existing files
	if args.Cln {
		if err = fu.WipeDir(args.Dest); err != nil {
			return err
		}
	}

	// Create scanner, either from system pipe or given file
	if err := dataprov.CreateScanner(args); err != nil {
		return err
	}

	// Execute processing
	if err = ProcessStream(args, dataprov.scanner); err != nil {
		return err
	}

	// Remove cluster subdirectory (if exists), if Move Cluster Data has been selected
	if args.MvRl {
		if err = os.RemoveAll(filepath.Join(args.Dest, "-")); err != nil {
			return err
		}
	}

	return nil

}

// Check wether regular expression (given by a user) is compilable
// If so, assign to the rgx_ExcludeObjType
func IsExclObjTypeOk(rgx string) error {

	if rgx == "" {
		return nil
	}

	var err error

	if rgx_ExcludeObjType, err = regexp.Compile(rgx); err != nil {
		return err
	}

	return nil

}

// Decide whethere currently scanned database is selected/blacklisted
func enableCurrentDb(dbname string) bool {

	if rgx_WhiteListDb != nil {
		matches := rgx_WhiteListDb.FindStringSubmatch(dbname)
		if len(matches) > 0 {
			return true
		} else {
			return false
		}
	}

	if rgx_ExclDb != nil {
		matches := rgx_ExclDb.FindStringSubmatch(dbname)
		if len(matches) > 0 {
			return false
		}
	}

	return true
}

// Decide whether collected object should be stored into file or not.
// Decision is made based on database it belongs to and the fact it's whitelisted/blacklisted
func allowObject(dbo *DbObject) bool {

	// Exclude objects by type given by regular expression passed with prog arguments
	if rgx_ExcludeObjType != nil {
		if rgx_ExcludeObjType.MatchString(dbo.ObjType) || rgx_ExcludeObjType.MatchString(dbo.ObjSubtype) {
			return false
		}
	}

	if dbo.ObjType == "DATABASE" {
		return enableCurrentDb(dbo.Name)
	}

	if dbo.Database != "" && dbo.Database != "-" {
		return enableCurrentDb(dbo.Database)
	}

	return true

}

// Most outer processing function.
// It initializes a stream either from a file or pgdump, and processes it line by line.
func ProcessStream(args *Config, scanner *bufio.Scanner) error {

	lineno := 0

	var dbname string
	var clusterphase = true
	var curObj DbObject
	var err error
	var processdb bool = true

	if args.ExDb != "" {
		rgx_ExclDb, err = regexp.Compile(args.ExDb)
		if err != nil {
			return fmt.Errorf("invalid regular expression for databases exclusion")
		}
	}

	if args.WlDb != "" {
		rgx_WhiteListDb, err = regexp.Compile(args.WlDb)
		if err != nil {
			return fmt.Errorf("invalid regular expression for databases whitelisting")
		}
	}

	rgx := `^\\(un)?restrict `
	if args.Restrict != "" {
		rgx = `^\\(un)?restrict ` + args.Restrict + `[\n\r]*$`
	}

	rgx_restrict, err = regexp.Compile(rgx)
	if err != nil {
		return fmt.Errorf("invalid Restrict argument; breaks regular expression compilation")
	}

	// Iterate over each line
	for scanner.Scan() {
		lineno = lineno + 1
		line := scanner.Text()

		// Skip restrict/unrestrict commands
		if rgx_restrict.MatchString(line) {
			continue
		}

		// Reacts on row:
		// \connect database_name
		if db := InitDatabaseFromLine(&line); db != "" {

			dbname = db

			if err := Save(&curObj); err != nil {
				return err
			}

			// init of the obj
			curObj.init(args.AclFiles)

			if !clusterphase {
				processdb = enableCurrentDb(dbname)
			}
			continue
		}

		if clusterphase {

			// Reacts on rows:
			// -- Roles
			// -- Role memberships
			// -- User Config "user_name"
			// Starts collecting data for obj type ROLE

			if obj := InitRoleObjFromLine(&line, args, dbname); obj != nil {

				if err := Save(&curObj); err != nil {
					return err
				}

				curObj = *obj
				continue
			}
		}

		// Reacts on rows:
		// -- User Configurations
		// -- User Databases
		// -- PostgreSQL database dump
		// -- PostgreSQL database dump complete
		//
		// Database init ends cluster data (retmode = 2)
		// Completion of database (retmode = 2), triggers copying user roles into the database structure (if enabled by configuration)
		// New users (retmode = 1) - still in cluster part
		// Any other found token (retmode = 0)
		// no token found (retmode = -1)
		//
		retmode := EndOfCluster(&line, args, dbname)

		if retmode == 2 {
			clusterphase = false
		}

		if retmode == 2 {

			if args.MvRl && dbname != "" && enableCurrentDb(dbname) {
				if err := RelocateClusterRoles(args.Dest, dbname); err != nil {
					return err
				}
			}
		}

		if retmode >= 0 {

			if err := Save(&curObj); err != nil {
				return err
			}

			curObj.init(args.AclFiles)

			continue
		}

		if !clusterphase && !processdb {
			continue
		}

		// Reacts on rows:
		// -- Name: some name; Type: some type; Schema: some_schema;
		// -- Data for Name: some name; Type: some type; Schema: some_schema;
		//
		// Starts collecting data for obj type ROLE
		// If type is TABLE DATA, data are not being added to the object (for performance reasons)
		obj := InitCommonObjFromLine(&line, args, dbname)
		if obj != nil {

			if err := Save(&curObj); err != nil {
				return err
			}

			curObj = *obj
			continue
		}

		if curObj.ObjType != "" && curObj.ObjType != "TABLE DATA" {

			curObj.appendContent(&line)
		}

	}

	// save the last row remaining in the buffer
	if err := Save(&curObj); err != nil {
		return err
	}

	// at end of the file, move roles to db location if requested
	if args.MvRl && dbname != "" && enableCurrentDb(dbname) {
		if err := RelocateClusterRoles(args.Dest, dbname); err != nil {
			return err
		}
	}

	// Check for any errors that may have occurred during scanning
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("%s. Fails on line: %d.\nConsider setting buffer size to higher value", err.Error(), lineno+1)
	}

	return nil
}

func InitDatabaseFromLine(line *string) string {

	matches := rgx_conn.FindStringSubmatch(*line)

	if len(matches) > 0 {
		if matches[4] != "" {
			return matches[4]
		} else {
			return matches[2]
		}
	}

	return ""
}

// When completion of database is recognized, we copy user roles into it (if enabled by configuration)

func EndOfCluster(line *string, args *Config, dbname string) int {

	if retmode := MatchDbStartEnd(line); retmode != -1 {
		return retmode
	}

	return MatchUsersAndDatabasesStart(line)

}

// Regognize beginning or the end of database dump
// returned values (int)
// 0: database end
// 2: database start
// -1: unmatched
func MatchDbStartEnd(line *string) int {
	matches := rgx_dbdump.FindStringSubmatch(*line)

	if len(matches) > 0 {
		if matches[1] == "complete" {
			return 0
		} else {
			return 2 // begining of database. Means end of cluster data
		}
	}

	return -1
}

func MatchUsersAndDatabasesStart(line *string) int {

	matches := rgx_users.FindStringSubmatch(*line)

	if len(matches) > 0 {

		if matches[1] == "Databases" {
			return 2
		}

		return 1

	}

	return -1
}

func InitRoleObjFromLine(line *string, args *Config, dbname string) *DbObject {

	matches := rgx_roles.FindStringSubmatch(*line)

	if len(matches) == 0 {
		return nil
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

	return &DbObject{
		Name:     objtype,
		ObjType:  "ROLE",
		Schema:   "-",
		Database: dbname,
		AclFiles: args.AclFiles,
		Paths: DbObjPath{
			Rootpath:   args.Dest,
			IsCustom:   args.Mode == "custom",
			NoDbInPath: args.NoDb,
		},
	}

}

func InitCommonObjFromLine(line *string, args *Config, dbname string) *DbObject {

	matches := rgx_common.FindStringSubmatch(*line)

	if len(matches) == 0 {
		return nil
	}

	// Iterate over each match
	//result := remapMatches(matches)

	result := make(map[string]string)
	for i, name := range rgx_common.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = matches[i]
		}
	}

	obj := &DbObject{

		Name:     result["Name"],
		ObjType:  result["Type"],
		Schema:   result["Schema"],
		Database: dbname,
		AclFiles: args.AclFiles,
		Paths: DbObjPath{
			Rootpath:   args.Dest,
			IsCustom:   args.Mode == "custom",
			NoDbInPath: args.NoDb,
		},
	}

	return obj

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

func Save(dbo *DbObject) error {

	if !allowObject(dbo) {
		return nil
	}

	if dbo.Content.Len() > 0 {
		return dbo.StoreObj()
	}

	return nil
}

// Moves roles from root, to each database location.
func RelocateClusterRoles(destpath string, dbname string) error {

	var srcloc = filepath.Join(destpath, "-")
	var dstloc = filepath.Join(destpath, dbname, "-")

	if err := fu.CopyDir(srcloc, dstloc); err != nil {
		return err
	}

	return nil
}
