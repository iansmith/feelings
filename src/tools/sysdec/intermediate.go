package sysdec

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func runIntermediate(outFile string, tmpDir string, opts *UserOptions) {
	declFile := filepath.Join(tmpDir, opts.Out)
	args := []string{}
	if opts.Out != "" {
		args = []string{opts.Out}
		declFile = opts.Out
	}
	cmd := exec.Command(outFile, args...)
	cmd.Dir = tmpDir
	msgs, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("unable to get output of intermediate program "+
			"%s: %v\n-----------\n%s\n----------\n", outFile, err,msgs)
	}
	if opts.Leave {
		out := "<STDOUT>"
		if opts.Out != "" {
			out = declFile
		}
		log.Printf("outputfile is %s", out)
		log.Printf("intermediate compilation output\n%s", msgs)
	}
	if opts.Out == "" {
		fmt.Println(string(msgs))
	}
}

func findGoCompiler(msg string ) string {
	//try the env var we expect first
	hostgo:=strings.TrimSpace(os.Getenv("HOSTGO"))
	if hostgo=="" {
		path, err := exec.LookPath("go")
		if err != nil {
			log.Fatalf("%s: unable to find " +
				" 'go' command in your PATH", msg)
		}
		log.Printf("%s: did not find HOSTGO variable, using go in PATH")
		return path
	}
	candidate:=filepath.Join(hostgo,"go")
	stat, err:=os.Stat(candidate)
	if err!=nil && !os.IsNotExist(err) {
		log.Fatalf("%s: %v",err)
	}
	if stat!=nil && !stat.IsDir() {
		log.Printf("%s: found HOSTGO environment points at bin "+
			"directory, not typical", msg)
		return candidate
	}
	candidate=filepath.Join(hostgo,"bin","go")
	stat, err=os.Stat(candidate)
	if err!=nil && !os.IsNotExist(err) {
		log.Fatalf("%s: %v",err)
	}
	if os.IsNotExist(err) {
		//try the path
		path, err := exec.LookPath("go")
		if err!=nil {
			log.Fatalf("%s: unable to find go compiler via HOSTGO or PATH")
		}
		return path
	}
	return candidate

}

// after generation, this gets called to compile the program that results
func compileIntermediate(mainFilePath string, tmpDir string, opts *UserOptions) string {
	path:= findGoCompiler("trying to compile intermediate program:")
	tempOut := filepath.Join(tmpDir, "sysdecl_sys")
	cmd := exec.Command(path, "build", "-o", tempOut, "-tags", opts.InTags,
		mainFilePath)
	cmd.Env = []string{
		fmt.Sprintf("%s=%s", "GO111MODULE", os.Getenv("GO111MODULE")),
		fmt.Sprintf("%s=%s", "GOPATH", os.Getenv("GOPATH")),
		fmt.Sprintf("%s=%s", "HOME", os.Getenv("HOME")),
	}
	captured, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("unable to run 'go' command: %v\n %s", err, captured)
	}
	if opts.Leave {
		log.Printf("intermediate main file: %s", mainFilePath)
		log.Printf("intermediate executable: %s", tempOut)
		log.Printf("intermediate output:\n%s", captured)
	}
	return tempOut
}

// OutputPeripherals runs in the intermediate stage main and generates
// the final output.  Note the use of flag.NArg refers to the
// *intermediate* main.
func OutputPeripherals(root DeviceDef, outTags string, pkg string, imp string, sourceFile string) {
	//write the output file
	fp := os.Stdout
	if flag.NArg() > 0 {
		var err error
		fp, err = os.Create(flag.Arg(0))
		if err != nil {
			log.Fatalf("intermediate program opening output file: %v", err)
		}
	}

	//walk each one
	GenerateDeviceDecls(root, outTags, pkg, imp, sourceFile, fp)
	fp.Close()
}
