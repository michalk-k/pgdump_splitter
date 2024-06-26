package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"pgdump_splitter/dbobject"
	"pgdump_splitter/output"
)

var version = "0.0.0" // provided by build flag (VERSION file)

func main() {

	var args dbobject.Config

	flag.StringVar(&args.File, "f", "", "path to dump generated by pg_dump or pg_dumpall. If omited the program will expect data on stdin via system pipe.")
	flag.StringVar(&args.Mode, "mode", "custom", "The mode of dumping db objects. origin - for file organization as present in the database dump. custom - reorganizes db objects storing related ones into single file")
	flag.StringVar(&args.Dest, "dst", "structure", "Location where structures will be dumped to")
	flag.BoolVar(&args.NoDb, "ndb", false, "No db name in destination path. It should not be set to true if multiple databases are dumped at once")
	flag.StringVar(&args.ExDb, "blacklist-db", "^(template|postgres)", "Regular expression pattern allowing to skip extraction of matching databases. Usefull in case of processing dump files. In case of using a pipe from pg_dumpall, exclude them using pd_dumpall switch.")
	flag.StringVar(&args.WlDb, "whitelist-db", "", "Regular expression pattern allowing to whitelist databases. If set, only databases matching this expression will be processed")
	flag.BoolVar(&args.MvRl, "mc", false, "Move dump of roles into each database subdirectory")
	flag.IntVar(&args.BufS, "buffer", 1024*1024, "Set up maximum buffer sizze if your dump contains data not feeting the scanner")
	flag.BoolVar(&args.Cln, "clean", false, "If true, it will wipe out the content of the destination directory. Otherwise will attempt to add new files")
	flag.BoolVar(&args.Quiet, "quiet", false, "If true, no information is outputed to std out")
	flag.BoolVar(&args.AclFiles, "aclfiles", false, "Applicable or mode=custom only. Makes GRANTs to be outputed to separate files suffixed with .acl.sql, ie table_name.acl.sql. Otherwise acls are appended to related object file.")
	flag.StringVar(&args.ExOT, "exclude-objects", "", "Regular expression pattern allowing to skip extraction of matching database objects. The expression is matched against TYPE value found in the dumped SQL")
	flag.Bool("version", false, "Show program version")

	flag.Parse()

	output.Quiet = args.Quiet

	if isFlagPassed("version") {
		fmt.Printf("pgdump_splitter %s\n", version)
		return
	}

	if flag.NFlag() == 0 {
		flag.PrintDefaults()
		return
	}

	if !(args.Mode == "" || args.Mode == "custom" || args.Mode == "origin") {
		fmt.Fprintln(os.Stderr, "Invalid value passed to `mode` modifier")
		if !args.Quiet {
			flag.PrintDefaults()
		}
		return
	}

	err := dbobject.StartProcessing(&args)
	if err != nil {
		log.Fatalf("Finished with error: %s", err.Error())
	}

	// Print the output
	output.Println("Finished")

}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
