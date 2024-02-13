package dbobjects

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	fu "pgdump_splitter/fileutils"
	"regexp"
	"strings"
)

type DbObject struct {
	Rootpath   string
	Schema     string
	Name       string
	ObjType    string
	ObjSubtype string
	ObjSubName string
	FullPath   string
	Content    string
	IsCustom   bool
}

// Extracts documentation (DOCU section) from the contect.
func (obj DbObject) extractDocu() {

	Content, err := os.ReadFile(obj.FullPath)
	if err != nil {
		fmt.Println("Error")
		log.Fatal(err)
	}

	rgx := regexp.MustCompile(`(?s)DOCU(.*)DOCU`)
	matches := rgx.FindSubmatch(Content)
	if len(matches) > 1 {

		newfile := filepath.Dir(obj.FullPath) + "/" + filepath.Base(obj.FullPath) + ".md"

		err := os.WriteFile(newfile, matches[1], 0777)
		if err != nil {
			log.Fatal(err)
		}

	}
}

// Stores objects to the file.
// It makes some minor formatting (mainly adds/removes EOLs)
// It also extracts documentation for function code if available
func (obj *DbObject) StoreObj() {

	obj.normalizeDbObject()
	obj.generateDestinationPath()

	if obj.FullPath == "" {
		fmt.Println("Emtpy path")
		fmt.Println(obj.Content)
		return
	}

	newlycreated := fu.CreateFile(obj.FullPath)
	newfile, err := os.OpenFile(obj.FullPath, os.O_APPEND|os.O_WRONLY|os.O_APPEND, 0770)

	if err != nil {
		fmt.Println("Could not open path:" + obj.FullPath)
		return
	}
	defer newfile.Close()

	var prefix string
	if !newlycreated {
		prefix = "\n"
	}

	_, err2 := newfile.WriteString(prefix + strings.Trim(obj.Content, " -\n") + "\n")

	if err2 != nil {
		fmt.Println("Could not write text to:" + obj.FullPath)
	}

	if obj.ObjType == "FUNCTION" {
		obj.extractDocu()
	}

}

// In some cases pgdump generates function identifiers containing argument names, incl OUT keyword
// this funciton stripes unwanted parts our from the identifier.
// Note, it's naive, condidering the input string is in requested format.
// There is no validation for that, so passing proper identifier will brake the result.
func removeArgNamesFromFunctionIdent(funcident string) string {
	rgx := regexp.MustCompile(", (OUT )?([\\w]+)")
	funcident = rgx.ReplaceAllLiteralString(funcident, ",")

	rgx = regexp.MustCompile("\\(([\\w]+ )")
	funcident = rgx.ReplaceAllString(funcident, "(")

	return funcident
}

// Generate filename of the db function
// Special treatment is needed due to possibility of going beyond limits of file path length.
// It might happen in case of higher number of function arguments
//
// In this implementation, the function arguments are replaced by md5 hash calculated from string representing the arguments
func generateFunctionFileName(funcident string) string {
	rgx := regexp.MustCompile("^(.*)\\((.*)\\)$")
	matches := rgx.FindStringSubmatch(funcident)

	if len(matches) > 0 {

		if matches[2] != "" {
			return matches[1] + "-" + funcArgsToHash(matches[2])[0:6]
		} else {
			return matches[1]
		}
	}

	return funcident
}

// Modifies meta information of object, of some of their data are stored name of the object
// It applies to indexes, triggers and similar objects which have no parent object type stored in object name
func (dbo *DbObject) normalizeSubtypes2(newtype string) {

	rgx := regexp.MustCompile("^(.*) (.*)$")
	matches := rgx.FindStringSubmatch(dbo.Name)

	if len(matches) > 0 {
		dbo.Name = matches[2]
		dbo.ObjSubName = matches[1]
		dbo.ObjSubtype = newtype
	}
}

// Modifies meta information of object, of some of their data are stored name of the object
// It applies to comments or ACLs
func (dbo *DbObject) normalizeSubtypes() {
	rgx := regexp.MustCompile("^([A-Z]+) (.*?)(\\.(.*))?$")
	matches := rgx.FindStringSubmatch(dbo.Name)

	if len(matches) > 0 {
		dbo.ObjSubtype = matches[1]
		dbo.ObjSubName = matches[2]

	}
}

func (dbo *DbObject) normalizeIndex() {
	rgx := regexp.MustCompile(" ON ([\\w]+)\\.([\\w]+)")
	matches := rgx.FindStringSubmatch(dbo.Content)

	if len(matches) > 0 {
		dbo.ObjSubtype = "TABLE"
		dbo.ObjSubName = matches[2]
	}
}

// prepares object type-based part of the file path
// In `origin` mode it leaves names untouched
// In `custom` mode it makes names lowercase, also it ensures plural form of the name, ie TABLE -> tables, INDEX -> indexes
func generateObjTypePath(typename string, iscustom bool) string {

	var objtpename string

	if iscustom {
		objtpename = strings.ToLower(typename)
		if objtpename == "index" {
			objtpename = "indexes"
		} else {
			objtpename = objtpename + "s"
		}
	} else {
		objtpename = typename
	}

	return objtpename
}

// generate path to the file for the dumped object
func (dbo *DbObject) generateDestinationPath() {

	if dbo.ObjSubtype == "FUNCTION" {
		dbo.ObjSubName = removeArgNamesFromFunctionIdent(dbo.ObjSubName)
		dbo.ObjSubName = generateFunctionFileName(dbo.ObjSubName)
	}

	if dbo.ObjType == "FUNCTION" {
		dbo.Name = generateFunctionFileName(dbo.Name)
	}

	if dbo.ObjType == "SCHEMA" || dbo.ObjSubtype == "SCHEMA" {
		dbo.FullPath = dbo.Rootpath + dbo.Name + "/" + dbo.Name + ".sql"
	} else {

		objtpename := generateObjTypePath(dbo.ObjType, dbo.IsCustom)

		if dbo.IsCustom {
			if dbo.ObjSubtype == "" {
				dbo.FullPath = dbo.Rootpath + dbo.Schema + "/" + objtpename + "/" + dbo.Name + ".sql"
			} else {
				dbo.FullPath = dbo.Rootpath + dbo.Schema + "/" + generateObjTypePath(dbo.ObjSubtype, dbo.IsCustom) + "/" + dbo.ObjSubName + ".sql"
			}
		} else {
			if dbo.ObjSubtype == "" {
				dbo.FullPath = dbo.Rootpath + dbo.Schema + "/" + objtpename + "/" + dbo.Name + ".sql"
			} else {
				dbo.FullPath = dbo.Rootpath + dbo.Schema + "/" + objtpename + "/" + dbo.ObjSubName + ".sql"
			}
		}

	}

}

// The function fixes content of the object due to the fact that pgdump stores data in non-consistent way.
// For example it stores information about object type (in case of ACL or COMMENT) in a name attribute.
func (dbo *DbObject) normalizeDbObject() {

	switch dbo.ObjType {
	case "COMMENT":
		dbo.normalizeSubtypes()
		if dbo.ObjSubtype == "COLUMN" {
			dbo.ObjSubtype = "TABLE"
		}
	case "ACL":
		dbo.normalizeSubtypes()
	case "FK CONSTRAINT":
		dbo.normalizeSubtypes2("TABLE")
	case "CONSTRAINT":
		dbo.normalizeSubtypes2("TABLE")
	case "TRIGGER":
		dbo.normalizeSubtypes2("TABLE")
	case "INDEX":
		dbo.normalizeIndex()
	case "DEFAULT":
		dbo.normalizeSubtypes2("TABLE")
	case "SEQUENCE SET":
		dbo.normalizeSubtypes2("SEQUENCE")
	case "SEQUENCE OWNED BY":
		dbo.normalizeSubtypes2("SEQUENCE")
	case "PUBLICATION TABLE":
		if dbo.IsCustom {
			dbo.Schema = "-"
		}
		dbo.normalizeSubtypes2("PUBLICATION")
	}

	if dbo.ObjSubtype == "SCHEMA" {
		dbo.Schema = dbo.ObjSubName
		dbo.Name = dbo.ObjSubName
	}
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
