package main

import (
 "fmt"
 "log"
 "os/exec"
 "os"
 "regexp"
 "bufio"
 "strings"
 "path/filepath"
 "crypto/md5"
"encoding/hex"
"io/ioutil"
)

func pgdump() {
	cmd := exec.Command(
		"pg_dump",
		"-h127.0.0.1",
		"-p50032",
		"-Upostgres",
		"--file=dump.sql",
		"clients_and_terminals",
	)

	stdoutStderr, err := cmd.CombinedOutput()
	fmt.Printf("%s\n", stdoutStderr)
	
	if err != nil {
		log.Fatal(err)
	}
}


func rgxtest() {

	re := regexp.MustCompile("^-- Name: (.*); Type: (.*); Schema: (.*);")
	txt, err := os.ReadFile("dump.sql")
	if err != nil {
		fmt.Println("could not read the file content")
	}
	fmt.Printf("%s", txt)
	fmt.Printf("%q\n", re.FindAllSubmatch(txt, -1))

}

type DbObject struct {
    schema 	string
    name  	string
    objtype string
	fullpath string
}

func NormalizeAcl(dbo *DbObject) {
	rgx := regexp.MustCompile("^([A-Z]+) (.*)$")
	matches := rgx.FindStringSubmatch(dbo.name)

	if len(matches) > 0 {
		dbo.name = matches[2]

		switch matches[1] {
			case "SCHEMA":
				dbo.schema = matches[2]
				dbo.fullpath = dbo.name + "/" + dbo.name + ".sql"
			case "FUNCTION": dbo.fullpath = dbo.schema + "/" + strings.ToLower(matches[1]) + "s/" + GenerateFunctionFileName(dbo.name) + ".sql"
			default : dbo.fullpath = dbo.schema + "/" + strings.ToLower(matches[1]) + "s/" + dbo.name + ".sql"
			}
	}
}

func RedirectObject(dbo *DbObject) {
	rgx := regexp.MustCompile("^([A-Z]+) (.*)$")
	matches := rgx.FindStringSubmatch(dbo.name)

	if len(matches) > 0 {
		
		dbo.name = matches[2]

		switch matches[1] {
		case "SCHEMA":
			dbo.schema = matches[2]
			dbo.objtype = "SCHEMA"
		case "COLUMN":
			dbo.objtype = "TABLE"
		default:
			dbo.objtype = matches[1]
		}

		NormalizeDbObject(dbo)

	}
}

func GenerateFunctionFileName(funcident string) string {
	rgx := regexp.MustCompile("^(.*)\\((.*)\\)$")
	matches := rgx.FindStringSubmatch(funcident)

	if len(matches) > 0 {

		if (matches[2] != "") {
			return matches[1] + "-" + funcArgsToHash(matches[2])[0:6] + ".sql"
		} else {
			return matches[1] + ".sql"
		}
	}

	return funcident
}


func NormalizeSplit(dbo *DbObject, newtype string) {

	rgx := regexp.MustCompile("^(.*) (.*)$")
	matches := rgx.FindStringSubmatch(dbo.name)

	if len(matches) > 0 {
		dbo.name = matches[1]
		dbo.objtype = newtype
		NormalizeDbObject(dbo)
	}
}



func NormalizeDbObject(dbo *DbObject) {

	switch dbo.objtype {
	case "SCHEMA": dbo.fullpath = dbo.name + "/" + dbo.name + ".sql"
	case "COMMENT": RedirectObject(dbo)
	case "ACL": RedirectObject(dbo)
	case "FUNCTION": dbo.fullpath = dbo.schema + "/" + strings.ToLower(dbo.objtype) + "s/" + GenerateFunctionFileName(dbo.name)
	case "FK CONSTRAINT": NormalizeSplit(dbo, "TABLE")
	case "CONSTRAINT": NormalizeSplit(dbo, "TABLE")
	case "TRIGGER": NormalizeSplit(dbo, "TABLE")
	case "DEFAULT": NormalizeSplit(dbo, "TABLE")
	case "SEQUENCE SET": 
		dbo.objtype = "SEQUENCE"
		NormalizeDbObject(dbo)
	case "SEQUENCE OWNED BY": 
		dbo.objtype = "SEQUENCE"
		NormalizeDbObject(dbo)
	case "PUBLICATION TABLE":
		dbo.schema = "-"
		NormalizeSplit(dbo, "PUBLICATION")
	default : dbo.fullpath = dbo.schema + "/" + strings.ToLower(dbo.objtype) + "s/" + dbo.name + ".sql"
	}

}


func CreateFile(filefullpath string) {

	_, err := os.Stat(filefullpath)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(filefullpath), 0770); err != nil {
			log.Fatal(err)
		}
		os.Create(filefullpath)

	}
}


func preserveNewlines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			// Include the newline character in the token
			return i + 1, data[0:i+1], nil
		}
	}
	// If at end of file and no newline found, return the entire data
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}


func funcArgsToHash(input string) string {
	// Calculate the MD5 hash
	hash := md5.Sum([]byte(input))

	// Convert the hash to a hexadecimal string
	hashStr := hex.EncodeToString(hash[:])

	return hashStr
}

func extractDocu (dbo *DbObject) {
	
	content, err := ioutil.ReadFile("structure/" + dbo.fullpath)
	if err != nil {
		fmt.Println("Error")
		log.Fatal(err)
	}

	rgx := regexp.MustCompile(`(?s)DOCU(.*)DOCU`)
	matches := rgx.FindSubmatch(content)
	if len(matches) > 1 {


		newfile := filepath.Dir("structure/" + dbo.fullpath) + "/" + filepath.Base("structure/" + dbo.fullpath) + ".md"

		err := ioutil.WriteFile(newfile, matches[1], 0777)
		if err != nil {
			log.Fatal(err)
		}

	}

}

func ProcessDump() {
	// Open the file
	file, err := os.Open("dump.sql")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer file.Close()

	err = os.RemoveAll("structure/")
    if err != nil {
        fmt.Println("Error:", err)
        return
    }

	rgx := regexp.MustCompile("^-- Name: (?P<Name>.*); Type: (?P<Type>.*); Schema: (?P<Schema>.*);")

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)
	scanner.Split(preserveNewlines)

	var curObj DbObject
	var newfile *os.File
	// Iterate over each line
	for scanner.Scan() {
		line := scanner.Text()
		// Process the line here
		
		

		matches := rgx.FindStringSubmatch(line)
		

		if len(matches) > 0 {

			if curObj.objtype == "FUNCTION" {
				err = newfile.Sync()
				if err != nil {
					fmt.Println("Error:", err)
					log.Fatal(err)
					return
				}
				newfile.Close()
				extractDocu(&curObj)
			}



			// Iterate over each match
			result := make(map[string]string)
			for i, name := range rgx.SubexpNames() {
				if i != 0 && name != "" {
					result[name] = matches[i]
				}
			}
			curObj = DbObject{
				name : result["Name"],
				objtype : result["Type"],
				schema : result["Schema"],

			}

			NormalizeDbObject(&curObj)
			fmt.Println(curObj.objtype + " -> " + curObj.fullpath)
			CreateFile("structure/" + curObj.fullpath)
			scanner.Scan()
			scanner.Scan()
			continue
		}

	//	fmt.Println(curObj.objtype + " -> " + curObj.fullpath)
		if curObj.fullpath == "" {
			continue
		}

		newfile,err = os.OpenFile("structure/" + curObj.fullpath, os.O_APPEND|os.O_WRONLY, 0770)

		if err != nil {
			fmt.Println("Could not open path:" + curObj.fullpath)
			return
		}
		defer newfile.Close()
		
		_, err2 := newfile.WriteString(line)

		if err2 != nil {
			fmt.Println("Could not write text to example.txt")
		}
	}

	// Check for any errors that may have occurred during scanning
	if err := scanner.Err(); err != nil {
		fmt.Println("Error:", err)
	}
}

func main() {

	ProcessDump()
     // Print the output
	 fmt.Println("Finished")
}