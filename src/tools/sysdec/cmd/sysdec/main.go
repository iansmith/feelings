package main

import (
	"flag"
	"log"
	"os"

	"tools/sysdec"
)

var outfile = flag.String("o", "", "output filename")
var dump = flag.Bool("d", false, "dump human readable version (debugging use only")
var pkg = flag.String("p", "main", "package to emit generated code into")
var outtags = flag.String("b", "", "output build tags (copied verbatim to output)")
var intags = flag.String("t", "", "input build tags (in correct go format)")
var imp = flag.String("i", "runtime/volatile", "package name that has volatile.Register")
var leave = flag.Bool("l", false, "leave work products")

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		log.Fatalf("usage svd2go -d -p <pkg> -o <outputfile> <input filename, either .csvd or .svd>")
	}
	fp, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close()
	opts := &sysdec.UserOptions{
		Out:           *outfile,
		Dump:          *dump,
		Pkg:           *pkg,
		InputFilename: flag.Arg(0),
		OutTags:       *outtags,
		InTags:        *intags,
		Import:        *imp,
		Leave:         *leave,
	}
	//it was _just_ an svd
	sysdec.ProcessSysdec(fp, opts)
}
