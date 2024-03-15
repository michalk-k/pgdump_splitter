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
var rgx_users *regexp.Regexp
var rgx_dbdump *regexp.Regexp
var rgx_roles *regexp.Regexp
var rgx_common *regexp.Regexp
var rgx_ExclDb *regexp.Regexp
var rgx_WhiteListDb *regexp.Regexp

var allow_current_db bool = false

func init() {
	rgx_conn = regexp.MustCompile(`^\\connect (.*)`)
	rgx_users = regexp.MustCompile(`^-- (User Configurations|Databases)[\s]*$`)
	rgx_dbdump = regexp.MustCompile(`^-- PostgreSQL database dump[\s]*(complete)?[\s]*$`)
	rgx_roles = regexp.MustCompile(`(^-- (?P<Type1>Roles|Role memberships)[\s]*$)|(^-- (?P<Type2>User Config) \".*\"[\s]*$)`)
	rgx_common = regexp.MustCompile(`^-- (Data for )?Name: (?P<Name>.*); Type: (?P<Type>.*); Schema: (?P<Schema>.*);`)
}

// Prepare input streams into scanner and pass it to for processing
func StartProcessing(args *Config) error {

	var err error
	var dataprov ScanerProvider

	output.Println("Destination location: " + args.Dest)
	output.Println(fmt.Sprintf("Clean destination location: %t", args.Cln))

	if args.Cln {
		if err = os.RemoveAll(args.Dest); err != nil {
			return err
		}
	}

	if err := dataprov.CreateScanner(args); err != nil {
		return err
	}

	if err = ProcessStream(args, dataprov.scanner); err != nil {
		return err
	}

	// Optionally remove - subdirectory (if exists).
	// it contains roles data created from pgdumpall
	if args.MvRl {
		if err = os.RemoveAll(filepath.Join(args.Dest, "-")); err != nil {
			return err
		}
	}

	return nil

}

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

func allowObject(dbo *DbObject) bool {

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

	// Iterate over each line
	for scanner.Scan() {
		lineno = lineno + 1
		line := scanner.Text()

		// Reacts on row:
		// \connect database_name
		if db := InitDatabaseFromLine(line); db != "" {

			dbname = db

			if err := Save(&curObj, rgx_ExclDb, rgx_WhiteListDb); err != nil {
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

			if obj := InitRoleObjFromLine(line, *args, dbname); obj != nil {

				if err := Save(&curObj, rgx_ExclDb, rgx_WhiteListDb); err != nil {
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
		// When completion of database is recognized, we copy user roles into it (if enabled by configuration)
		//
		// POSSIBLY NO REINIT OBJECT IN EndOfCluster for rgx_users
		//

		retmode, err := EndOfCluster(line, *args, dbname)

		if err != nil {
			return err
		}

		if retmode == 2 {
			clusterphase = false
		}

		if retmode > 0 {

			if err := Save(&curObj, rgx_ExclDb, rgx_WhiteListDb); err != nil {
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
		obj := InitCommonObjFromLine(line, *args, dbname)
		if obj != nil {

			if err := Save(&curObj, rgx_ExclDb, rgx_WhiteListDb); err != nil {
				return err
			}

			curObj = *obj
			continue
		}

		if curObj.ObjType != "" && curObj.ObjType != "TABLE DATA" {
			curObj.Content = curObj.Content + line
		}

	}

	// save the last row remaining in the buffer
	if err := Save(&curObj, rgx_ExclDb, rgx_WhiteListDb); err != nil {
		return err
	}

	// Check for any errors that may have occurred during scanning
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("%s. Fails on line: %d.\nConsider setting buffer size to higher value", err.Error(), lineno+1)
	}

	return nil
}

func InitDatabaseFromLine(line string) string {

	matches := rgx_conn.FindStringSubmatch(line)

	if len(matches) > 0 {
		return matches[1]
	}

	return ""
}

func EndOfCluster(line string, args Config, dbname string) (int, error) {

	matches := rgx_dbdump.FindStringSubmatch(line)

	if len(matches) > 0 {

		if matches[1] == "complete" {

			if args.MvRl && enableCurrentDb(dbname) {
				if err := RelocateClusterRoles(filepath.Join(args.Dest, "-"), filepath.Join(args.Dest, dbname, "-")); err != nil {
					return 0, err
				}
			}

		} else {
			return 2, nil // cluster is finished. db is starting
		}

		return 1, nil
	}

	matches = rgx_users.FindStringSubmatch(line)

	if len(matches) > 0 {

		if matches[1] == "Databases" {
			return 2, nil
		}

		return 1, nil

	}

	return 0, nil // cluster is continuing

}

func InitRoleObjFromLine(line string, args Config, dbname string) *DbObject {

	matches := rgx_roles.FindStringSubmatch(line)

	if len(matches) == 0 {
		return nil
	}

	// Iterate over each match
	result := make(map[string]string)
	for i, name := range rgx_common.SubexpNames() {
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
		DocuRgx:  args.Docu,
		AclFiles: args.AclFiles,
		Paths: DbObjPath{
			Rootpath:   args.Dest,
			IsCustom:   args.Mode == "custom",
			NoDbInPath: args.NoDb,
		},
	}

}

func InitCommonObjFromLine(line string, args Config, dbname string) *DbObject {

	rgx_common := regexp.MustCompile(`^-- (Data for )?Name: (?P<Name>.*); Type: (?P<Type>.*); Schema: (?P<Schema>.*);`)
	matches := rgx_common.FindStringSubmatch(line)

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
		DocuRgx:  args.Docu,
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

func Save(dbo *DbObject, exdb_rgx *regexp.Regexp, wldb_rgx *regexp.Regexp) error {

	if !allowObject(dbo) {
		return nil
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
