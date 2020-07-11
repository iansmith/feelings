package main

import (
	"bytes"
	"fmt"
	"io"

	"machine"

	"drivers/emmc"
	"lib/trust"
)

func main() {
	if err := emmc.Impl.Init(); err != emmc.EmmcOk {
		trust.Errorf("failed to init emmc card: %s", err)
		machine.Abort()
	}
	dumpFileText("/foo/bar/nss")
	dumpFileText("/foo/baz/nss")
	dumpFileText("/hostname")
	dumpFileText("/common-auth")
	dumpFileText("/sources.list")
	dumpFileText("/lsb-release")
	dumpFileText("/resolve.conf")
	dumpFileText("/reslove.conf")
	dumpFileText("/README")
	emmc.Impl.WindUp()
	machine.Abort()
}

func dumpFileText(path string) {
	rd, err := emmc.Impl.Open(path)
	if err != nil {
		trust.Errorf("!!! can't open '%s': %s", path, err)
		return
	}
	fmt.Printf("---------\n%s\n---------\n", path)
	data := make([]byte, 827)
	var buff bytes.Buffer
	for {
		n, err := rd.Read(data)
		if err == io.EOF {
			if n != 0 {
				trust.Debugf("got EOF! but with unexpected value of n: %d", n)
			}
			break
		}
		if err != nil {
			trust.Errorf("error reading from file: %s", err)
			machine.Abort()
		}
		buff.Write(data[:n])
	}
	fmt.Printf(buff.String() + "\n")
	rd.(emmc.EmmcFile).Close()
}
