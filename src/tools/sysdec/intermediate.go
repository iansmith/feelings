package sysdec

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
			"%s: %v", outFile, err)
	}
	if opts.Leave {
		out := "<STDOUT>"
		if opts.Out != "" {
			out = declFile
		}
		log.Printf("outputfile is %s", out)
	}
	if opts.Out == "" {
		fmt.Println(string(msgs))
	}
}

// after generation, this gets called to compile the program that results
func compileIntermediate(mainFilePath string, tmpDir string, opts *UserOptions) string {
	path, err := exec.LookPath("go")
	if err != nil {
		log.Fatalf("trying to compile: unable to find " +
			" 'go' command in your PATH")
	}
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
