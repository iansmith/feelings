package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	flag.Parse()
	if flag.NArg() < 2 {
		log.Fatalf("unable to process input, expected arguments: " +
			"genfuncptr <infile> <outfile>")
	}
	in, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatalf("open: %v", err)
	}
	exists := true
	st, err := os.Stat(flag.Arg(1))
	if err != nil {
		err, ok := err.(*os.PathError)
		if !ok {
			log.Fatalf("%v", err)
		}
		exists = false
	}
	var lastGenTime time.Time
	if exists {
		lastGenTime = st.ModTime()
	}
	st, err = os.Stat(flag.Arg(0))
	if err != nil {
		log.Fatalf("stat 0: %v", err)
	}
	lastModTime := st.ModTime()

	log.Printf("last mod time: %s, last gen time: %s", lastModTime, lastGenTime)
	if lastModTime.After(lastGenTime) {
		generate(in, flag.Arg(1))
	}
	os.Exit(0)
}

func generate(fp *os.File, outFilename string) {
	out, err := os.Create(outFilename)
	if err != nil {
		log.Fatalf("%v", err)
	}
	wr := bufio.NewWriter(out)
	rd := bufio.NewScanner(fp)
	haveWarned := false
	for rd.Scan() {
		line := rd.Text()
		if strings.HasSuffix(line, "\n") {
			line = strings.TrimSuffix(line, "\n")
		}
		if !haveWarned {
			wr.WriteString(warn)
			haveWarned = true
		}
		wr.WriteString(fmt.Sprintf(lit, line, line, line, line))
	}
	if err := rd.Err(); err != nil {
		log.Fatalf("error reading input: %v", err)
	}
	wr.Flush()
	out.Close()
	fp.Close()
}

const lit = `
.global "ladies/%s"
.global "ladies/%sPtr"
"ladies/%sPtr":
	.dword "ladies/%s"
`
const warn = `
// DO NOT EDIT! This file is machine genarted by genfuncptrs and your
// changes will be overwritten.

`
