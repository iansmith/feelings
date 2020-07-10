package main

import (
	"bytes"
	"fmt"

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
	dumpFileText("/reslove.conf")
	dumpFileText("/resloveit.conf")
	dumpFileText("/README")
	emmc.Impl.WindUp()
	machine.Abort()
}

func dumpFileText(path string) {
	rd, err := emmc.Impl.Open(path)
	if err != emmc.EmmcOk {
		trust.Errorf("!!! can't open /hostname: %s", err)
		return
	}
	fmt.Printf("---------\n%s\n---------\n", path)
	data := make([]byte, 827)
	var buff bytes.Buffer
	for {
		n, err := rd.Read(data)
		if err == emmc.EmmcEOF {
			trust.Debugf("got EOF! with n %d", n)
			break
		}
		if err != emmc.EmmcOk {
			trust.Errorf("error reading from file: %s", err)
			machine.Abort()
		}
		buff.Write(data[:n])
	}
	fmt.Printf(buff.String() + "\n")
	rd.Close()
}
