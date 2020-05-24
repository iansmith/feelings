package sysdec

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type templateGroup struct {
	device       *template.Template
	bitField     *template.Template
	preamble     *template.Template
	constant     *template.Template
	intermediate *template.Template
	register     *template.Template
}

func createOutputTemplates() *templateGroup {

	//
	// Create templates
	//
	deviceTemplate := template.New("device")
	deviceTemplate = template.Must(deviceTemplate.Parse(deviceTemplateText))

	bitFieldDeclTemplate := template.New("bitFieldDecl")
	bitFieldDeclTemplate = template.Must(bitFieldDeclTemplate.Parse(bitFieldDeclTemplateText))

	preambleTemplate := template.New("preamble")
	preambleTemplate = template.Must(preambleTemplate.Parse(preambleTemplateText))

	constantTemplate := template.New("constant")
	constantTemplate = template.Must(constantTemplate.Parse(constantTemplateText))

	intermediateTemplate := template.New("intermediate")
	intermediateTemplate = template.Must(intermediateTemplate.Parse(intermediatTemplateText))

	registerTemplate := template.New("register")
	registerTemplate = template.Must(registerTemplate.Parse(registerTemplateText))

	return &templateGroup{device: deviceTemplate,
		bitField:     bitFieldDeclTemplate,
		preamble:     preambleTemplate,
		constant:     constantTemplate,
		intermediate: intermediateTemplate,
		register:     registerTemplate,
	}
}

type intermediateResult struct {
	InTags     string
	Pkg        string //intermediate package
	Root       string
	Dump       bool
	OutTags    string
	Package    string //output package
	Import     string
	SourceFile string
}

func ProcessSysdec(fp *os.File, opts *UserOptions) {

	//system:=SystemDeclaration{}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sysdec", fp, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	name := fileToName(file, opts.InputFilename)
	log.Printf("creating outputfile based on system declaration of '%s'",
		name)

	//setup template machinery
	group := createOutputTemplates()
	var output bytes.Buffer
	p := intermediateResult{
		InTags:     opts.InTags,
		Pkg:        "main",
		Root:       name,
		Dump:       opts.Dump,
		OutTags:    opts.OutTags,
		Package:    opts.Pkg,
		Import:     opts.Import,
		SourceFile: opts.InputFilename,
	}

	//create the new main
	t := filepath.Join("tmp", "int_main.go")
	if err := group.intermediate.Execute(&output, p); err != nil {
		log.Fatalf("error running intermediate template: %v", err)
	}
	out, err := os.Create(t)
	if err != nil {
		log.Fatalf("error creating intermediate main: %v", err)
	}
	copied, err := io.Copy(out, &output)
	if err != nil {
		log.Fatalf("unable to copy to intermediate main: %v", err)
	}
	out.Close()
	if opts.Leave {
		log.Printf("created intermediate file (%d bytes) %s",
			copied, t)
	}

	tmp := os.TempDir()
	outFile := compileIntermediate(t, tmp, opts)
	runIntermediate(outFile, tmp, opts)
}

func fileToName(file *ast.File, filename string) string {
	var d *ast.GenDecl
	var s *ast.ValueSpec
	ok := false
	found := false
	for _, candidate := range file.Decls {
		d, ok = candidate.(*ast.GenDecl)
		if !ok {
			log.Fatalf("%s: unexpected type %T found in "+
				"declarations, aborting", candidate)
		}
		for _, spec := range d.Specs {
			s, ok = spec.(*ast.ValueSpec)
			if ok {
				if !found {
					found = true
					continue
				}
				log.Fatalf("%s: can only process one file at a time, "+
					"one declaration at a time, aborting",
					filename)
			}
		}
	}
	if d == nil || s == nil {
		log.Fatalf("%s: no top level  declaration found, aborting",
			filename)
	}
	if len(s.Names) != 1 {
		log.Fatalf("%s: expected to find one name in the declaration "+
			"but found: %d", filename, len(s.Names))
	}
	ident := s.Names[0]
	return ident.Name
}

func GenerateDeviceDecls(device DeviceDef, outTags string, pkg string,
	imp string, source string, fp io.Writer) {
	var output bytes.Buffer

	//for intermediate main
	device.OutTags = outTags
	device.Import = imp
	device.Package = pkg
	device.SourceFilename = source

	group := createOutputTemplates()

	//remove elements of device.Peripherals that don't have memory declaration
	//for those that do, fill the mmiobase
	killList := []string{}
	for name, p := range device.Peripheral {
		addr, ok := device.MMIOBindings[name]
		if !ok {
			killList = append(killList, name)
		}
		p.Name = name //copy it from the map
		p.MMIOBase = addr
		parts := strings.Split(p.Description, "\n")
		var b bytes.Buffer
		for _, p := range parts {
			b.WriteString("//" + p + "\n")
		}
		p.Description = b.String()
		//copy the names from the map to the Registers
		for name, r := range p.Register {
			r.Name = name
		}
	}

	for _, name := range killList {
		delete(device.Peripheral, name)
	}

	constants := map[string]int{}

	//deal with peripherals that have gaps, we assume that these are
	//reserved, so we do not make them exported.  This also orders
	//the contents of RegisterDefWithReserved so it is in the correct
	//memory layout order.
	for _, p := range device.Peripheral {
		count := 0
		p.RegistersWithReserved = []*RegisterDef{}
		for i := 0; i < p.AddressBlock.Size; i += 4 {
			n, ok := findRegisterAtOffset(p.Register, i)
			if !ok {
				resv := &RegisterDef{
					Name:          fmt.Sprintf("reserved%03d", count),
					AddressOffset: i,
					Size:          32,
					IsReserved:    true,
				}
				count++
				p.RegistersWithReserved = append(p.RegistersWithReserved, resv)
				continue
			}
			def := p.Register[n]
			if def.Dim != 0 {
				def.Name = strings.TrimSuffix(def.Name, "[%s]")
				extra := (def.Dim - 1) * 4
				i += extra //it's an array, and one was already counted
				for k, v := range def.DimIndices {
					constants[k] = v
				}
				if def.DimIncrement != 4 {
					panic("unable to handle non-32 bit registers in arrays:" + def.Name)
				}
			}
			p.RegistersWithReserved = append(p.RegistersWithReserved, def)
		}
	}

	//send all the bitfields of registers to the templates so we can
	//emit nice helpers
	fields := []*FieldDef{}
	for _, p := range device.Peripheral {
		for regname, reg := range p.Register {
			for name, f := range reg.Field {
				f.Name = name
				f.RegName = regname
				if f.Access.IsSet() {
					f.CanRead = f.Access.CanRead()
					f.CanWrite = f.Access.CanWrite()
				} else {
					if !reg.Access.IsSet() {
						log.Fatalf("neither register %s nor field %s "+
							" has declared access level (r,w, or rw): ",
							regname, name)
					}
					f.CanWrite = reg.Access.CanWrite()
					f.CanRead = reg.Access.CanRead()
				}
				fields = append(fields, f)
				for ename, e := range f.EnumeratedValue {
					e.Name = ename //copy name from map
					e.Field = f    //hook up to field
				}
			}
		}
	}
	//preamble has just package, build tags, etc
	if err := group.preamble.Execute(&output, device); err != nil {
		log.Fatalf("failed to execute the preamble template: %v", err)
	}
	//this has the fun stuff
	if err := group.device.Execute(&output, device); err != nil {
		log.Fatalf("failed to execute the device template: %v", err)
	}
	//registers
	if err := group.register.Execute(&output, device); err != nil {
		log.Fatalf("failed to execute the device template: %v", err)
	}
	// do bitfields
	for _, f := range fields {
		if err := group.bitField.Execute(&output, f); err != nil {
			log.Fatalf("failed to execute the bitfield template: %v", err)
		}
	}
	if err := group.constant.Execute(&output, constants); err != nil {
		log.Fatalf("failed to execute the constants template: %v", err)
	}

	_, err := io.Copy(fp, &output)
	if err != nil {
		log.Fatalf("unable to copy output: %v", err)
	}
}

func findRegisterAtOffset(all map[string]*RegisterDef, offset int) (string, bool) {
	for name, def := range all {
		if def.AddressOffset == offset {
			return name, true
		}
	}
	return "", false
}
