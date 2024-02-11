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
	Rootpath string
	Schema   string
	Name     string
	ObjType  string
	FullPath string
	Content  string
}

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

func (obj DbObject) StoreObj() {

	//	NormalizeDbObject(&obj)

	if obj.FullPath == "" {
		fmt.Println("Emtpy path")
		fmt.Println(obj.Content)
		return
	}

	fu.CreateFile(obj.FullPath)
	newfile, err := os.OpenFile(obj.FullPath, os.O_APPEND|os.O_WRONLY|os.O_APPEND, 0770)

	if err != nil {
		fmt.Println("Could not open path:" + obj.FullPath)
		return
	}
	defer newfile.Close()

	_, err2 := newfile.WriteString(strings.Trim(obj.Content, " -\n") + "\n")

	if err2 != nil {
		fmt.Println("Could not write text to:" + obj.FullPath)
	}

	if obj.ObjType == "FUNCTION" {
		obj.extractDocu()
	}

}

func NormalizeAcl(dbo *DbObject) {
	rgx := regexp.MustCompile("^([A-Z]+) (.*)$")
	matches := rgx.FindStringSubmatch(dbo.Name)

	if len(matches) > 0 {
		dbo.Name = matches[2]

		switch matches[1] {
		case "SCHEMA":
			dbo.Schema = matches[2]
			dbo.FullPath = dbo.Name + "/" + dbo.Name + ".sql"
		case "FUNCTION":
			dbo.FullPath = dbo.Schema + "/" + strings.ToLower(matches[1]) + "s/" + GenerateFunctionFileName(dbo.Name) + ".sql"
		default:
			dbo.FullPath = dbo.Schema + "/" + strings.ToLower(matches[1]) + "s/" + dbo.Name + ".sql"
		}
	}
}

func RedirectObject(dbo *DbObject) {
	rgx := regexp.MustCompile("^([A-Z]+) (.*?)(\\.(.*))?$")
	matches := rgx.FindStringSubmatch(dbo.Name)

	if len(matches) > 0 {

		dbo.Name = matches[2]

		switch matches[1] {
		case "SCHEMA":
			dbo.Schema = matches[2]
			dbo.ObjType = "SCHEMA"
		case "COLUMN":
			dbo.ObjType = "TABLE"
		default:
			dbo.ObjType = matches[1]
		}

		NormalizeDbObject(dbo)

	}
}

func GenerateFunctionFileName(funcident string) string {
	rgx := regexp.MustCompile("^(.*)\\((.*)\\)$")
	matches := rgx.FindStringSubmatch(funcident)

	if len(matches) > 0 {

		if matches[2] != "" {
			return matches[1] + "-" + funcArgsToHash(matches[2])[0:6] + ".sql"
		} else {
			return matches[1] + ".sql"
		}
	}

	return funcident
}

func NormalizeSplit(dbo *DbObject, newtype string) {

	rgx := regexp.MustCompile("^(.*) (.*)$")
	matches := rgx.FindStringSubmatch(dbo.Name)

	if len(matches) > 0 {
		dbo.Name = matches[1]
		dbo.ObjType = newtype
		NormalizeDbObject(dbo)
	}
}

func NormalizeDbObject(dbo *DbObject) {

	switch dbo.ObjType {
	case "SCHEMA":
		dbo.FullPath = dbo.Rootpath + dbo.Name + "/" + dbo.Name + ".sql"
	case "COMMENT":
		RedirectObject(dbo)
	case "ACL":
		RedirectObject(dbo)
	case "FUNCTION":
		dbo.FullPath = dbo.Rootpath + dbo.Schema + "/" + strings.ToLower(dbo.ObjType) + "s/" + GenerateFunctionFileName(dbo.Name)
	case "FK CONSTRAINT":
		NormalizeSplit(dbo, "TABLE")
	case "CONSTRAINT":
		NormalizeSplit(dbo, "TABLE")
	case "TRIGGER":
		NormalizeSplit(dbo, "TABLE")
	case "DEFAULT":
		NormalizeSplit(dbo, "TABLE")
	case "SEQUENCE SET":
		dbo.ObjType = "SEQUENCE"
		NormalizeDbObject(dbo)
	case "SEQUENCE OWNED BY":
		dbo.ObjType = "SEQUENCE"
		NormalizeDbObject(dbo)
	case "PUBLICATION TABLE":
		dbo.Schema = "-"
		NormalizeSplit(dbo, "PUBLICATION")
	default:
		objtpename := strings.ToLower(dbo.ObjType)
		if objtpename == "index" {
			objtpename = "indexes"
		} else {
			objtpename = objtpename + "s"
		}
		dbo.FullPath = dbo.Rootpath + dbo.Schema + "/" + objtpename + "/" + dbo.Name + ".sql"
	}

}

func funcArgsToHash(input string) string {
	// Calculate the MD5 hash
	hash := md5.Sum([]byte(input))

	// Convert the hash to a hexadecimal string
	hashStr := hex.EncodeToString(hash[:])

	return hashStr
}
