package main

import (
	"flag"
	"fmt"
	"log"
	"pgdump_splitter/dbobject"
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
	flag.StringVar(&args.Docu, "doc", `/\*DOCU(.*)DOCU\*/`, "Move dump of roles into each database subdirectory.")
	flag.IntVar(&args.BufS, "buffer", 1024*1024, "Set up maximum buffer sizze if your dump contains data not feeting the scanner")
	flag.Bool("version", false, "Show program version")

	flag.Parse()

	if isFlagPassed("version") {
		fmt.Printf("pgdump_splitter %s\n", version)
		return
	}

	if flag.NFlag() == 0 {
		flag.PrintDefaults()
		return
	}

	if !(args.Mode == "" || args.Mode == "custom" || args.Mode == "origin") {
		fmt.Println("Invalid value passed to `mode` modifier")
		flag.PrintDefaults()
		return
	}

	err := dbobject.StartProcessing(&args)
	if err != nil {
		log.Fatalf("Finished with error: %s", err.Error())
	}

	// Print the output
	fmt.Println("Finished")
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
