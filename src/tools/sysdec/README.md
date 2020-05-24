The sysdec tool reads in "system declarations" and outputs files that provide
useful programming APIs to the hardware described.  Both the input and output
are go files.  

The original purpose of this tool (and its three predecessors!) was to 
automatically derive the contents of the "machine" description used in
tinygo.  The `*.sysdec.go` files in the folder `src/machine` in the 
tinygo distribution are the output of this tool.

This tool considers the files in the `sys` subdirectory "input" or
"configuration files."  When these are changed you can immediately run
sysdec again and new outputs generated.  However, this is done in
an extremely hacky way.  The tool generates an intermediate _program_
that is linked against the files in `sys` as well as parts of the
sysdec program itself.  This intermediate program is run by the 
sysdec tool and the intermediat program generates the actual
output.  You can see more about the nastiness with the `-l` option
to sysdec.

Tricky: We use special build tags to allow the code in sys to not cause
your IDE to blow up and to allow us to compile specific subsets of the
`sys` directory.
```go
// +build feelings_rpi3_qemu

```
This tag gets removed in the final result, which uses the output tag
provided in the command line.