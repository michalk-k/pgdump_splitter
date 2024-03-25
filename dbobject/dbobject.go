package dbobject

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	fu "pgdump_splitter/fileutils"
	"regexp"
	"strings"
)

var rgx_normalize_index *regexp.Regexp
var rgx_normalize_subtypes_a *regexp.Regexp
var rgx_normalize_subtypes_b *regexp.Regexp
var rgx_normalize_subtypes2 *regexp.Regexp
var rgx_genFunctionName *regexp.Regexp
var rgx_fncNormArgNames_b *regexp.Regexp
var rgx_fncNormArgNames_c *regexp.Regexp

var Rgx_fncDocu *regexp.Regexp

func init() {

	rgx_normalize_index = regexp.MustCompile(` ON ([\w]+)\.([\w]+)`)
	rgx_normalize_subtypes_a = regexp.MustCompile(`^([A-Z ]+) (.*)$`)
	rgx_normalize_subtypes_b = regexp.MustCompile(`^([\S]+)\.([\S]+)$`)
	rgx_normalize_subtypes2 = regexp.MustCompile(`^(.*) (.*)$`)
	rgx_genFunctionName = regexp.MustCompile(`^(FUNCTION )?(.*)\((.*)\)$`)

	rgx_fncNormArgNames_b = regexp.MustCompile(`.*( DEFAULT.*)$`)
	rgx_fncNormArgNames_c = regexp.MustCompile(`(.*?)(((double precision|character varying|time without time zone|timestamp without time zone|timestamp with time zone|time with time zone|bit varying)|([\S]+))(\[\])?)`)

}

type DbObjPath struct {
	Rootpath    string
	NameForFile string
	FullPath    string
	NoDbInPath  bool
	IsCustom    bool
}

type DbObject struct {
	Schema     string
	Name       string
	ObjType    string
	ObjSubtype string
	ObjSubName string
	Content    strings.Builder
	Database   string
	DocuRgx    string
	AclFiles   bool
	Paths      DbObjPath
}

func (obj *DbObject) init(aclfiles bool) {
	*obj = DbObject{Paths: DbObjPath{}, AclFiles: aclfiles}
}

func (obj *DbObject) appendContent(line *string) {
	obj.Content.WriteString(*line)
}

// Check wether regular expression (given by a user) is compilable
func IsDocuRegexOk(rgx string) error {

	if rgx == "" {
		return nil
	}

	var err error

	if Rgx_fncDocu, err = regexp.Compile(`(?s)` + rgx); err != nil {
		return err
	}

	return nil

}

// Extracts documentation (DOCU section) from the contect.
func (obj *DbObject) extractDocu() error {

	if !(obj.ObjType == "FUNCTION" && obj.DocuRgx != "") {
		return nil
	}

	// Defer a function that recovers from MustCompile panic
	defer func() error {
		if r := recover(); r != nil {
			return fmt.Errorf("invalid regular expression for extracting documentation")
		}
		return nil
	}()

	newfile := filepath.Join(
		filepath.Dir(obj.Paths.FullPath),
		strings.TrimSuffix(filepath.Base(obj.Paths.FullPath), filepath.Ext(obj.Paths.FullPath))+".md")

	content := "# " + obj.Schema + "." + obj.Name

	matches := Rgx_fncDocu.FindSubmatch([]byte(obj.Content.String()))
	if len(matches) > 1 {
		content += "\n" + string(matches[1])
	}

	content += "\n [Back to function list](../readme.md)\n"

	if err := os.WriteFile(newfile, []byte(content), 0777); err != nil {
		return fmt.Errorf("error while writing file: %s", newfile)
	}

	return nil
}

// Stores objects to the file.
// It makes some minor formatting (mainly adds/removes EOLs)
// It also extracts documentation for function code if available
func (obj *DbObject) StoreObj() error {

	obj.normalizeDbObject()
	obj.generateDestinationPath()

	if obj.Paths.FullPath == "" {
		//		fmt.Println("Emtpy path")
		//		fmt.Println(obj.Content)
		return nil
	}

	newlycreated, err := fu.CreateFile(obj.Paths.FullPath)
	if err != nil {
		return fmt.Errorf("Could not create the file:" + obj.Paths.FullPath)
	}

	newfile, err := os.OpenFile(obj.Paths.FullPath, os.O_APPEND|os.O_WRONLY|os.O_APPEND, 0770)

	if err != nil {
		return fmt.Errorf("Could not open the file:" + obj.Paths.FullPath)
	}
	defer newfile.Close()

	var prefix string
	if !newlycreated {
		prefix = "\n"
	}

	content := obj.Content.String()
	_, err = newfile.WriteString(prefix + strings.Trim(content, " -\n") + "\n")

	if err != nil {
		return fmt.Errorf("Could not write text to:" + obj.Paths.FullPath)
	}

	if err = obj.extractDocu(); err != nil {
		return err
	}

	return nil

}

// In some cases pgdump generates function identifiers containing argument names, incl OUT keyword
// this function stripes unwanted parts our from the identifier.
// Note, it's naive, condidering the input string is in requested format.
// There is no validation for that, so passing proper identifier will brake the result.
func NormalizeFunctionIdentArgs(funcargs string) string {

	argsarr := strings.Split(funcargs, ", ")

	for i := 0; i < len(argsarr); i++ {

		argsarr[i] = rgx_fncNormArgNames_b.ReplaceAllString(argsarr[i], "")
		argmatches := rgx_fncNormArgNames_c.FindAllString(argsarr[i], -1)

		if len(argmatches) > 0 {
			argsarr[i] = strings.Trim(argmatches[len(argmatches)-1], " ")
		}
	}

	funcargs = strings.Join(argsarr, ", ")

	return funcargs
}

func getFuncIdentParts(funcident string) (string, string) {

	parts := rgx_genFunctionName.FindStringSubmatch(funcident)

	if len(parts) == 0 {
		return "", ""
	}

	if parts[3] != "" {
		return parts[2], parts[3]
	}

	return parts[2], ""
}

// Modifies meta information of object, of some of their data are stored name of the object
// It applies to indexes, triggers and similar objects which have no parent object type stored in object name
func (dbo *DbObject) normalizeSubtypes2(newtype string) error {

	matches := rgx_normalize_subtypes2.FindStringSubmatch(dbo.Name)

	if len(matches) > 0 {
		dbo.Name = matches[2]
		dbo.ObjSubName = matches[1]
		dbo.ObjSubtype = newtype
	}

	return nil
}

// Modifies meta information of object, of some of their data are stored name of the object
// It applies to comments or ACLs
func (dbo *DbObject) normalizeSubtypes() error {

	matches := rgx_normalize_subtypes_a.FindStringSubmatch(dbo.Name)

	if len(matches) > 0 {
		dbo.ObjSubtype = matches[1]
		dbo.ObjSubName = matches[2]
	}

	if dbo.ObjSubtype == "COLUMN" {

		matches := rgx_normalize_subtypes_b.FindStringSubmatch(dbo.ObjSubName)

		if len(matches) > 0 {
			dbo.ObjSubtype = "TABLE"
			dbo.ObjSubName = matches[1]
		}
	}

	return nil
}

func (dbo *DbObject) normalizeIndex() error {

	matches := rgx_normalize_index.FindStringSubmatch(dbo.Content.String())

	if len(matches) > 0 {
		dbo.ObjSubtype = "TABLE"
		dbo.ObjSubName = matches[2]
	}

	return nil
}

// prepares object type-based part of the file path
// In `origin` mode it leaves names untouched
// In `custom` mode it makes names lowercase
func generateObjTypePath(typename string, iscustom bool) string {

	if iscustom {
		return strings.ToLower(typename)
	} else {
		return typename
	}
}

// generate path to the file for the dumped object
func (dbo *DbObject) generateDestinationPath() {

	var name string

	if !dbo.Paths.IsCustom {
		name = dbo.Name
	} else if dbo.ObjType == "SCHEMA" || dbo.ObjSubtype == "SCHEMA" {
		name = dbo.Name
	} else if dbo.ObjType == "DATABASE" {
		name = dbo.Name
	} else if dbo.ObjSubtype != "" {
		name = dbo.ObjSubName
	} else {
		name = dbo.Name
	}

	if dbo.ObjSubtype == "FUNCTION" || dbo.ObjType == "FUNCTION" {

		fname, args := getFuncIdentParts(dbo.Name)
		args = NormalizeFunctionIdentArgs(args)
		dbo.Name = fname + "(" + args + ")"
		name = generateFuncFilename(fname, args)
	}

	dbo.Paths.NameForFile = name

	switch dbo.Paths.IsCustom {
	case true:
		dbo.generateDestinationPathCustom()
	case false:
		dbo.generateDestinationPathOrigin()
	}

}

// Generate filename of the db function
// Special treatment is needed due to possibility of going beyond limits of file path length.
// It might happen in case of higher number of function arguments
//
// In this implementation, the function arguments are replaced by md5 hash calculated from string representing the arguments
func generateFuncFilename(fname string, args string) string {

	if args == "" {
		return fname
	}

	return fname + "-" + funcArgsToHash(args)[0:6]
}

func (dbo *DbObject) generateDestinationPathOrigin() {

	var dbpath string
	if dbo.Database != "" && !dbo.Paths.NoDbInPath {
		dbpath = dbo.Database
	}

	if dbo.ObjType == "SCHEMA" || dbo.ObjSubtype == "SCHEMA" {
		dbo.Paths.FullPath = filepath.Join(dbo.Paths.Rootpath, dbpath, dbo.Paths.NameForFile, dbo.Paths.NameForFile) + ".sql"

	} else {

		objtpename := generateObjTypePath(dbo.ObjType, dbo.Paths.IsCustom)
		dbo.Paths.FullPath = filepath.Join(dbo.Paths.Rootpath, dbpath, dbo.Schema, objtpename, dbo.Paths.NameForFile) + ".sql"
	}

}

func (dbo *DbObject) generateDestinationPathCustom() {

	var dbpath string
	var suffix = ".sql"
	var path_objtype = dbo.ObjType
	var path_objsubtype = dbo.ObjSubtype

	if dbo.AclFiles && dbo.ObjType == "ACL" {
		suffix = ".acl" + suffix
	}

	if dbo.Database != "" && !dbo.Paths.NoDbInPath {
		dbpath = dbo.Database
	}

	if dbo.ObjType == "DATABASE" {
		dbpath = dbo.Name
	}

	if dbo.ObjType == "DEFAULT ACL" {
		dbo.Paths.NameForFile = strings.ToLower(strings.Replace(dbo.Paths.NameForFile, "DEFAULT PRIVILEGES FOR ", "", -1))
	}

	if dbo.ObjType == "SEQUENCE OWNED BY" {

		dbo.Paths.NameForFile = dbo.Name
		path_objtype = "SEQUENCE"
	}

	if dbo.ObjType == "SCHEMA" || dbo.ObjSubtype == "SCHEMA" {
		dbo.Paths.FullPath = filepath.Join(dbo.Paths.Rootpath, dbpath, dbo.Paths.NameForFile, dbo.Paths.NameForFile) + suffix
	} else {

		if dbo.ObjSubtype == "" {
			dbo.Paths.FullPath = filepath.Join(dbo.Paths.Rootpath, dbpath, dbo.Schema, generateObjTypePath(path_objtype, dbo.Paths.IsCustom), dbo.Paths.NameForFile) + suffix
		} else {
			dbo.Paths.FullPath = filepath.Join(dbo.Paths.Rootpath, dbpath, dbo.Schema, generateObjTypePath(path_objsubtype, dbo.Paths.IsCustom), dbo.Paths.NameForFile) + suffix
		}

	}

}

// The function fixes content of the object due to the fact that pgdump stores data in non-consistent way.
// For example it stores information about object type (in case of ACL or COMMENT) in a name attribute.
func (dbo *DbObject) normalizeDbObject() error {

	var err error
	/*
		if !dbo.Paths.IsCustom {
			return nil
		}
	*/
	switch dbo.ObjType {
	case "COMMENT":
		err = dbo.normalizeSubtypes()
	case "ACL":
		err = dbo.normalizeSubtypes()
	case "FK CONSTRAINT":
		err = dbo.normalizeSubtypes2("TABLE")
	case "CONSTRAINT":
		err = dbo.normalizeSubtypes2("TABLE")
	case "TRIGGER":
		err = dbo.normalizeSubtypes2("TABLE")
	case "INDEX":
		err = dbo.normalizeIndex()
	case "DEFAULT":
		err = dbo.normalizeSubtypes2("TABLE")
	case "SEQUENCE SET":
		dbo.ObjSubtype = "SEQUENCE"
		dbo.ObjSubName = dbo.Name
		err = dbo.normalizeSubtypes2("SEQUENCE")
	case "SEQUENCE OWNED BY":
		err = dbo.normalizeSubtypes2("SEQUENCE")
	case "DATABASE PROPERTIES":
		dbo.ObjSubtype = "DATABASE"
		dbo.ObjSubName = dbo.Name
	case "PUBLICATION TABLE":
		if dbo.Paths.IsCustom {
			dbo.Schema = "-"
		}
		err = dbo.normalizeSubtypes2("PUBLICATION")
	}

	if err != nil {
		return err
	}

	if dbo.ObjSubtype == "SCHEMA" {
		dbo.Schema = dbo.ObjSubName
		dbo.Name = dbo.ObjSubName
	}

	return nil
}

// Generates hash replacing db function input arguments.
// It's to shorten path for the function. Otherwise it might have happen that generated path would be too long for operating system
func funcArgsToHash(input string) string {
	// Calculate the MD5 hash
	hash := md5.Sum([]byte(input))

	// Convert the hash to a hexadecimal string
	hashStr := hex.EncodeToString(hash[:])

	return hashStr
}
